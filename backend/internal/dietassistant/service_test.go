package dietassistant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestServiceStreamChatConsumesOpenAICompatibleSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"你好\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"，想吃什么？\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	service := NewService(Options{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "dots-ai",
		Timeout: 3 * time.Second,
	})

	var events []StreamEvent
	err := service.StreamChat(context.Background(), []ChatMessage{
		{Role: "user", Content: "你好"},
	}, func(event StreamEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChat returned error: %v", err)
	}

	var content strings.Builder
	for _, event := range events {
		if event.Type == "delta" {
			content.WriteString(event.Delta)
		}
	}
	if got, want := content.String(), "你好，想吃什么？"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
	if events[len(events)-1].Type != "done" {
		t.Fatalf("last event = %q, want done", events[len(events)-1].Type)
	}
}

func TestServiceStreamChatRequiresConfig(t *testing.T) {
	service := NewService(Options{
		BaseURL: "https://example.com/v1",
		Model:   "dots-ai",
	})

	err := service.StreamChat(context.Background(), []ChatMessage{
		{Role: "user", Content: "你好"},
	}, func(StreamEvent) error { return nil })
	if err == nil {
		t.Fatal("expected config error")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}
