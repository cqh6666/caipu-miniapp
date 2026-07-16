package dietassistant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upstream"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestCreateChatCompletionRejectsOversizedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeRepeatedBytes(w, maxDietAssistantJSONResponseBytes+1)
	}))
	defer server.Close()

	service := NewService(Options{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 3 * time.Second,
	})
	_, err := service.createChatCompletion(context.Background(), openAIChatRequest{Model: "test-model"})
	var appErr *common.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadGateway || appErr.Message != "diet assistant upstream response exceeded size limit" {
		t.Fatalf("unexpected error: %T %v", err, err)
	}
	if !errors.Is(err, upstream.ErrResponseTooLarge) {
		t.Fatalf("error does not retain size cause: %v", err)
	}
}

func TestConsumeOpenAIStreamStopsInfiniteEventAtLimit(t *testing.T) {
	reader := &countingRepeatReader{value: 'x'}
	err := consumeOpenAIStream(reader, func(StreamEvent) error { return nil })
	if !errors.Is(err, errStreamEventTooLarge) {
		t.Fatalf("error = %v, want stream event limit", err)
	}
	if reader.read > int64(maxDietAssistantStreamEventBytes+64*1024) {
		t.Fatalf("reader consumed %d bytes after limit", reader.read)
	}
}

func TestConsumeOpenAIStreamPreservesContextErrors(t *testing.T) {
	for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
		err := consumeOpenAIStream(errorReader{err: want}, func(StreamEvent) error { return nil })
		if !errors.Is(err, want) {
			t.Fatalf("error = %v, want %v", err, want)
		}
	}
}

func TestStreamFinalChatLimitsCumulativeVisibleContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		delta := strings.Repeat("a", 1024)
		for index := 0; index <= maxDietAssistantVisibleBytes/len(delta); index++ {
			writeLimitTestStreamDelta(t, w, delta)
		}
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	service := NewService(Options{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 3 * time.Second,
	})
	emitted := 0
	_, err := service.streamFinalChat(context.Background(), ChatContext{UserID: 1, KitchenID: 2}, nil, func(event StreamEvent) error {
		if event.Type == "delta" {
			emitted += len(event.Delta)
		}
		return nil
	})
	if !errors.Is(err, errVisibleTextTooLarge) {
		t.Fatalf("error = %v, want visible content limit", err)
	}
	if emitted != maxDietAssistantVisibleBytes {
		t.Fatalf("emitted bytes = %d, want %d", emitted, maxDietAssistantVisibleBytes)
	}
}

func TestDietAssistantLimitsToolPayloads(t *testing.T) {
	filter := newLongCatStreamFilter(nil)
	err := filter.Push(longCatToolOpenTag + strings.Repeat("x", maxDietAssistantToolBlockBytes+1))
	if !errors.Is(err, errToolPayloadTooLarge) {
		t.Fatalf("tool block error = %v", err)
	}

	_, err = parseToolArguments(strings.Repeat("x", maxDietAssistantToolArgumentsBytes+1))
	if !errors.Is(err, errToolPayloadTooLarge) {
		t.Fatalf("tool arguments error = %v", err)
	}
}

func TestStreamHandlerHidesProviderBodyAndIncludesRequestID(t *testing.T) {
	const requestID = "req-ai-sensitive-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `<html>provider internal.ai.example failed api_key=sk-raw-secret</html>`)
	}))
	defer server.Close()

	service := NewService(Options{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 3 * time.Second,
	})
	handler := NewHandler(service)
	body := `{"messages":[{"role":"user","content":"你好"}],"kitchenId":2}`
	req := httptest.NewRequest(http.MethodPost, "/api/diet-assistant/chat/stream", strings.NewReader(body))
	req.Header.Set(chimiddleware.RequestIDHeader, requestID)
	req = req.WithContext(common.WithCurrentUserID(req.Context(), 1))
	recorder := &observedStreamRecorder{ResponseRecorder: httptest.NewRecorder()}

	chimiddleware.RequestID(http.HandlerFunc(handler.StreamChat)).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.observed == nil {
		t.Fatal("expected stream failure to reach response observer")
	}
	raw := recorder.Body.String()
	for _, secret := range []string{"internal.ai.example", "sk-raw-secret", "<html>"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("SSE leaked %q: %s", secret, raw)
		}
	}

	var failure StreamEvent
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "data:"))
		if line == "" || json.Unmarshal([]byte(line), &failure) != nil || failure.Type != "error" {
			continue
		}
		break
	}
	if failure.Type != "error" || failure.Message != "饮食管家暂时不可用，请稍后再试" || failure.RequestID != requestID {
		t.Fatalf("unexpected SSE failure: %#v; raw=%s", failure, raw)
	}
}

type countingRepeatReader struct {
	value byte
	read  int64
}

func (r *countingRepeatReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = r.value
	}
	r.read += int64(len(buffer))
	return len(buffer), nil
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

type observedStreamRecorder struct {
	*httptest.ResponseRecorder
	observed error
}

func (r *observedStreamRecorder) ObserveError(err error) {
	r.observed = err
}

func writeRepeatedBytes(w io.Writer, size int64) {
	chunk := strings.Repeat("x", 32*1024)
	for remaining := size; remaining > 0; {
		count := int64(len(chunk))
		if count > remaining {
			count = remaining
		}
		if _, err := io.WriteString(w, chunk[:count]); err != nil {
			return
		}
		remaining -= count
	}
}

func writeLimitTestStreamDelta(t *testing.T, w io.Writer, text string) {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"choices": []map[string]any{{
			"delta": map[string]any{"content": text},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(w, "data: "+string(data)+"\n\n")
}
