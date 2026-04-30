package dietassistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestServiceStreamChatConsumesOpenAICompatibleSSE(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount += 1
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if requestCount == 1 {
			if req.Stream {
				t.Fatal("first request should be non-streaming tool planning")
			}
			if len(req.Tools) == 0 {
				t.Fatal("first request should include tools")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"你好，想吃什么？"},"finish_reason":"stop"}]}`))
			return
		}

		if !req.Stream {
			t.Fatal("final request should be streaming")
		}
		if len(req.Tools) != 0 {
			t.Fatal("final request should not include tools")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"你好\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"，想吃什么？\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	db := openDietAssistantTestDB(t)
	defer db.Close()
	service := NewService(Options{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "dots-ai",
		Timeout: 3 * time.Second,
		Repo:    NewRepository(db),
	})

	var events []StreamEvent
	err := service.StreamChat(context.Background(), ChatContext{UserID: 1, KitchenID: 2}, []ChatMessage{
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
	if requestCount != 2 {
		t.Fatalf("requestCount = %d, want 2", requestCount)
	}
	stored, err := service.ListStoredMessages(context.Background(), ChatContext{UserID: 1, KitchenID: 2}, 50)
	if err != nil {
		t.Fatalf("ListStoredMessages error = %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("len(stored) = %d, want 2", len(stored))
	}
	if stored[0].Role != "user" || stored[0].Content != "你好" {
		t.Fatalf("stored user = %#v", stored[0])
	}
	if stored[1].Role != "assistant" || stored[1].Content != "你好，想吃什么？" {
		t.Fatalf("stored assistant = %#v", stored[1])
	}
}

func TestServiceStreamChatExecutesRecipeCountTool(t *testing.T) {
	var requestCount int
	var sawToolResult bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount += 1
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		if requestCount == 1 {
			if req.Stream {
				t.Fatal("first request should be non-streaming")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_recipe_count","arguments":"{\"mealType\":\"main\",\"status\":\"all\"}"}}]},"finish_reason":"tool_calls"}]}`))
			return
		}

		for _, message := range req.Messages {
			if message.Role != "tool" {
				continue
			}
			if strings.Contains(message.Content.(string), `"count":7`) {
				sawToolResult = true
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"正餐共有 7 道。\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	service := NewService(Options{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "LongCat-2.0-Preview",
		Timeout: 3 * time.Second,
		CountRecipes: func(ctx context.Context, input RecipeCountInput) (int, error) {
			if input.UserID != 11 || input.KitchenID != 22 {
				t.Fatalf("unexpected count context: %#v", input)
			}
			if input.MealType != "main" || input.Status != "" {
				t.Fatalf("unexpected count filter: %#v", input)
			}
			return 7, nil
		},
	})

	var content strings.Builder
	err := service.StreamChat(context.Background(), ChatContext{UserID: 11, KitchenID: 22}, []ChatMessage{
		{Role: "user", Content: "正餐有多少道？"},
	}, func(event StreamEvent) error {
		if event.Type == "delta" {
			content.WriteString(event.Delta)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChat returned error: %v", err)
	}
	if !sawToolResult {
		t.Fatal("final request did not include recipe count tool result")
	}
	if got, want := content.String(), "正餐共有 7 道。"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
}

func TestExecuteAddRecipeMockRejectsMissingTitleAndAvoidsNilText(t *testing.T) {
	missingTitle := executeAddRecipeMock(openAIToolCall{
		Function: openAIToolCallFunction{
			Name:      "add_recipe_mock",
			Arguments: `{"mealType":"main","status":"wishlist"}`,
		},
	})
	if missingTitle["ok"] != false {
		t.Fatalf("missingTitle ok = %#v, want false", missingTitle["ok"])
	}

	result := executeAddRecipeMock(openAIToolCall{
		Function: openAIToolCallFunction{
			Name:      "add_recipe_mock",
			Arguments: `{"title":"番茄炒蛋","mealType":"main","status":"wishlist"}`,
		},
	})
	if result["ok"] != true {
		t.Fatalf("result ok = %#v, want true", result["ok"])
	}
	recipe, ok := result["recipe"].(map[string]any)
	if !ok {
		t.Fatalf("recipe = %#v, want map", result["recipe"])
	}
	for _, key := range []string{"ingredient", "summary", "note"} {
		if recipe[key] != "" {
			t.Fatalf("recipe[%s] = %#v, want empty string", key, recipe[key])
		}
	}
}

func TestServiceStreamChatRequiresConfig(t *testing.T) {
	service := NewService(Options{
		BaseURL: "https://example.com/v1",
		Model:   "dots-ai",
	})

	err := service.StreamChat(context.Background(), ChatContext{UserID: 1, KitchenID: 2}, []ChatMessage{
		{Role: "user", Content: "你好"},
	}, func(StreamEvent) error { return nil })
	if err == nil {
		t.Fatal("expected config error")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}
