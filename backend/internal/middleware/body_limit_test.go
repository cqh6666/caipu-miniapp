package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestBodyLimitRejectsDeclaredOversizeBeforeHandler(t *testing.T) {
	t.Parallel()

	called := false
	handler := RequestBodyLimit(8, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/demo", strings.NewReader("123456789"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge || called {
		t.Fatalf("status=%d called=%t", response.Code, called)
	}
}

func TestRequestBodyLimitUsesRouteOverrideAndLimitsUnknownLength(t *testing.T) {
	t.Parallel()

	handler := RequestBodyLimit(4, []BodyLimitOverride{
		{Method: http.MethodPost, Path: "/api/upload", MaxBytes: 8},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/upload", strings.NewReader("12345678"))
	request.ContentLength = -1
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("override status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/demo", strings.NewReader("12345"))
	request.ContentLength = -1
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("default status=%d body=%s", response.Code, response.Body.String())
	}
}
