package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
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
