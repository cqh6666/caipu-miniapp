package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConditionalTimeoutUsesOverrideForMatchingRoute(t *testing.T) {
	t.Parallel()

	var got time.Duration
	handler := ConditionalTimeout(30*time.Second, []TimeoutOverride{
		{
			Method:  http.MethodPost,
			Prefix:  "/api/admin/ai-routing/scenes/",
			Suffix:  "/test",
			Timeout: 3 * time.Minute,
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Fatal("expected request deadline")
		}
		got = time.Until(deadline)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/admin/ai-routing/scenes/flowchart/test", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got < 175*time.Second || got > 181*time.Second {
		t.Fatalf("override deadline = %s, want about 3m", got)
	}
}

func TestConditionalTimeoutFallsBackToDefaultForOtherRoutes(t *testing.T) {
	t.Parallel()

	var got time.Duration
	handler := ConditionalTimeout(30*time.Second, []TimeoutOverride{
		{
			Method:  http.MethodPost,
			Prefix:  "/api/admin/ai-routing/scenes/",
			Suffix:  "/test",
			Timeout: 3 * time.Minute,
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Fatal("expected request deadline")
		}
		got = time.Until(deadline)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/admin/ai-routing/scenes/flowchart", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got < 25*time.Second || got > 31*time.Second {
		t.Fatalf("default deadline = %s, want about 30s", got)
	}
}
