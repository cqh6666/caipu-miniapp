package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestRequestLoggerPreservesFlusher(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("wrapped response writer should preserve http.Flusher")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data: {\"type\":\"done\"}\n\n"))
		flusher.Flush()
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/diet-assistant/chat/stream", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !recorder.Flushed {
		t.Fatal("expected underlying response recorder to be flushed")
	}
}

func TestRequestLoggerUsesRoutePatternAndOmitsSensitiveURL(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := chi.NewRouter()
	router.Get("/api/invites/{token}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := RequestLogger(logger)(router)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/invites/sensitive-invite-token?code=sensitive-query-code",
		nil,
	)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode request log: %v; output=%s", err, output.String())
	}
	if got := entry["route"]; got != "/api/invites/{token}" {
		t.Fatalf("route=%v, want route template", got)
	}
	for _, forbiddenKey := range []string{"path", "query"} {
		if _, ok := entry[forbiddenKey]; ok {
			t.Fatalf("request log must not contain %q: %s", forbiddenKey, output.String())
		}
	}
	for _, secret := range []string{"sensitive-invite-token", "sensitive-query-code"} {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("request log leaked %q: %s", secret, output.String())
		}
	}
}

func TestRequestLoggerDoesNotLogUnmatchedRawPath(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := chi.NewRouter()
	handler := RequestLogger(logger)(router)
	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodGet, "/missing/sensitive-path-token?secret=value", nil),
	)

	if !strings.Contains(output.String(), `"route":"unmatched"`) || !strings.Contains(output.String(), `"status":404`) {
		t.Fatalf("unmatched request log=%s", output.String())
	}
	for _, secret := range []string{"sensitive-path-token", "secret=value"} {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("unmatched request log leaked %q: %s", secret, output.String())
		}
	}
}

func TestRequestLoggerRecordsSafeErrorChainAndBusinessID(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := chi.NewRouter()
	router.Get("/api/recipes/{recipeID}", func(w http.ResponseWriter, _ *http.Request) {
		rootErr := errors.New("database failed password=database-secret Bearer bearer-secret")
		common.WriteError(w, common.ErrInternal.WithErr(fmt.Errorf("load recipe: %w", rootErr)))
	})
	handler := chimiddleware.RequestID(RequestLogger(logger)(router))
	request := httptest.NewRequest(http.MethodGet, "/api/recipes/recipe_42?token=query-secret", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("response does not expose X-Request-ID")
	}
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("decode request log: %v; output=%s", err, output.String())
	}
	for key, want := range map[string]any{
		"route":      "/api/recipes/{recipeID}",
		"status":     float64(http.StatusInternalServerError),
		"recipe_id":  "recipe_42",
		"error_code": float64(common.CodeInternalServer),
	} {
		if got := entry[key]; got != want {
			t.Errorf("%s=%v, want=%v; log=%s", key, got, want, output.String())
		}
	}
	for _, key := range []string{"request_id", "error_type", "error_chain", "error_message"} {
		if strings.TrimSpace(fmt.Sprint(entry[key])) == "" {
			t.Errorf("%s is empty: %s", key, output.String())
		}
	}
	for _, secret := range []string{"database-secret", "bearer-secret", "query-secret"} {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("request error log leaked %q: %s", secret, output.String())
		}
	}
}
