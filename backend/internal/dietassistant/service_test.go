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
	var events []StreamEvent
	err := service.StreamChat(context.Background(), ChatContext{UserID: 11, KitchenID: 22}, []ChatMessage{
		{Role: "user", Content: "正餐有多少道？"},
	}, func(event StreamEvent) error {
		events = append(events, event)
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
	if !hasStreamEvent(events, "status", "", "正在判断需要调用的能力") {
		t.Fatalf("events missing planning status: %#v", events)
	}
	if !hasStreamEvent(events, "tool_start", "get_recipe_count", "正在统计美食库") {
		t.Fatalf("events missing tool_start: %#v", events)
	}
	if !hasStreamEvent(events, "tool_done", "get_recipe_count", "已完成菜谱统计") {
		t.Fatalf("events missing tool_done: %#v", events)
	}
}

func TestServiceStreamChatParsesURLOnlyMessageWithoutPlanningRequest(t *testing.T) {
	var requestCount int
	var sawToolResult bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount += 1
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !req.Stream {
			t.Fatal("url-only message should skip non-streaming planning request")
		}
		if len(req.Tools) != 0 {
			t.Fatal("final request should not include tools")
		}
		for _, message := range req.Messages {
			if message.Role != "tool" {
				continue
			}
			if strings.Contains(message.Content.(string), `"recipe"`) && strings.Contains(message.Content.(string), "番茄炒蛋") {
				sawToolResult = true
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"已保存番茄炒蛋。\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	var gotInput RecipeFromURLInput
	service := NewService(Options{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "LongCat-2.0-Preview",
		Timeout: 3 * time.Second,
		CreateFromURL: func(ctx context.Context, input RecipeFromURLInput) (RecipeFromURLResult, error) {
			gotInput = input
			return RecipeFromURLResult{
				Recipe: RecipeToolItem{
					ID:       "rec_url",
					Title:    "番茄炒蛋",
					MealType: input.MealType,
					Status:   input.Status,
				},
				MainIngredients: []string{"番茄 2 个", "鸡蛋 3 个"},
				StepsCount:      3,
			}, nil
		},
	})

	var content strings.Builder
	var events []StreamEvent
	err := service.StreamChat(context.Background(), ChatContext{UserID: 3, KitchenID: 4}, []ChatMessage{
		{Role: "user", Content: "https://www.bilibili.com/video/BV1xx411c7mD"},
	}, func(event StreamEvent) error {
		events = append(events, event)
		if event.Type == "delta" {
			content.WriteString(event.Delta)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChat returned error: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("requestCount = %d, want 1", requestCount)
	}
	if !sawToolResult {
		t.Fatal("final request did not include url parse tool result")
	}
	if gotInput.UserID != 3 || gotInput.KitchenID != 4 || gotInput.URL != "https://www.bilibili.com/video/BV1xx411c7mD" {
		t.Fatalf("unexpected tool input: %#v", gotInput)
	}
	if gotInput.MealType != "main" || gotInput.Status != "wishlist" {
		t.Fatalf("unexpected default filters: %#v", gotInput)
	}
	if got, want := content.String(), "已保存番茄炒蛋。"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
	if hasStreamEvent(events, "status", "", "正在判断需要调用的能力") {
		t.Fatalf("url-only path should skip planning status: %#v", events)
	}
	if !hasStreamEvent(events, "tool_start", "parse_and_add_recipe_from_url", "正在解析链接并保存食材") {
		t.Fatalf("events missing url tool_start: %#v", events)
	}
	if !hasStreamEvent(events, "tool_done", "parse_and_add_recipe_from_url", "已解析并保存食材") {
		t.Fatalf("events missing url tool_done: %#v", events)
	}
	mutation := findStreamMutation(events, "recipe_created")
	if mutation == nil {
		t.Fatalf("events missing recipe_created mutation: %#v", events)
	}
	if mutation.RecipeID != "rec_url" || mutation.RecipeTitle != "番茄炒蛋" {
		t.Fatalf("unexpected mutation: %#v", mutation)
	}
}

func TestExecuteParseAndAddRecipeFromURL(t *testing.T) {
	var gotInput RecipeFromURLInput
	service := NewService(Options{
		CreateFromURL: func(ctx context.Context, input RecipeFromURLInput) (RecipeFromURLResult, error) {
			gotInput = input
			return RecipeFromURLResult{
				Recipe: RecipeToolItem{
					ID:         "rec_test",
					Title:      "番茄炒蛋",
					MealType:   input.MealType,
					Status:     input.Status,
					Ingredient: "番茄、鸡蛋",
					Summary:    "家常快手菜",
				},
				Source:          "bilibili",
				SummaryMode:     "ai",
				MainIngredients: []string{"番茄 2 个", "鸡蛋 3 个"},
				StepsCount:      4,
			}, nil
		},
	})

	missingURL := service.executeParseAndAddRecipeFromURL(context.Background(), ChatContext{UserID: 1, KitchenID: 2}, openAIToolCall{
		Function: openAIToolCallFunction{
			Name:      "parse_and_add_recipe_from_url",
			Arguments: `{"mealType":"main","status":"wishlist"}`,
		},
	})
	if missingURL["ok"] != false {
		t.Fatalf("missingURL ok = %#v, want false", missingURL["ok"])
	}

	result := service.executeParseAndAddRecipeFromURL(context.Background(), ChatContext{UserID: 1, KitchenID: 2}, openAIToolCall{
		Function: openAIToolCallFunction{
			Name:      "parse_and_add_recipe_from_url",
			Arguments: `{"url":"https://www.bilibili.com/video/BV1xx411c7mD","mealType":"main","status":"wishlist"}`,
		},
	})
	if result["ok"] != true {
		t.Fatalf("result ok = %#v, want true", result["ok"])
	}
	recipe, ok := result["recipe"].(RecipeToolItem)
	if !ok {
		t.Fatalf("recipe = %#v, want RecipeToolItem", result["recipe"])
	}
	if recipe.ID != "rec_test" || recipe.Title != "番茄炒蛋" {
		t.Fatalf("recipe = %#v", recipe)
	}
	if gotInput.UserID != 1 || gotInput.KitchenID != 2 {
		t.Fatalf("input context = %#v", gotInput)
	}
	if gotInput.URL != "https://www.bilibili.com/video/BV1xx411c7mD" || gotInput.MealType != "main" || gotInput.Status != "wishlist" {
		t.Fatalf("input fields = %#v", gotInput)
	}
	if result["stepsCount"] != 4 {
		t.Fatalf("stepsCount = %#v, want 4", result["stepsCount"])
	}
}

func TestDietAssistantToolsExposeURLParserWithoutDirectAddRecipe(t *testing.T) {
	names := make(map[string]bool)
	for _, tool := range dietAssistantTools() {
		names[tool.Function.Name] = true
	}
	if names["add_recipe"] {
		t.Fatal("add_recipe should not be exposed as a diet assistant tool")
	}
	if !names["parse_and_add_recipe_from_url"] {
		t.Fatal("parse_and_add_recipe_from_url should be exposed as a diet assistant tool")
	}
}

func TestExecuteSearchRecipesByName(t *testing.T) {
	var gotInput RecipeSearchInput
	service := NewService(Options{
		SearchRecipes: func(ctx context.Context, input RecipeSearchInput) ([]RecipeToolItem, error) {
			gotInput = input
			return []RecipeToolItem{{
				ID:       "rec_1",
				Title:    "番茄炒蛋",
				MealType: "main",
				Status:   "wishlist",
			}}, nil
		},
	})

	result := service.executeSearchRecipesByName(context.Background(), ChatContext{UserID: 7, KitchenID: 8}, openAIToolCall{
		Function: openAIToolCallFunction{
			Name:      "search_recipes_by_name",
			Arguments: `{"titleKeyword":"番茄","mealType":"all","status":"wishlist","limit":30}`,
		},
	})
	if result["ok"] != true {
		t.Fatalf("result ok = %#v, want true", result["ok"])
	}
	if gotInput.UserID != 7 || gotInput.KitchenID != 8 {
		t.Fatalf("input context = %#v", gotInput)
	}
	if gotInput.TitleKeyword != "番茄" || gotInput.MealType != "" || gotInput.Status != "wishlist" || gotInput.Limit != 10 {
		t.Fatalf("input filters = %#v", gotInput)
	}
	if result["count"] != 1 {
		t.Fatalf("result count = %#v, want 1", result["count"])
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

func hasStreamEvent(events []StreamEvent, eventType, toolName, message string) bool {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		if toolName != "" && event.ToolName != toolName {
			continue
		}
		if message != "" && event.Message != message {
			continue
		}
		return true
	}
	return false
}

func findStreamMutation(events []StreamEvent, mutationType string) *StreamMutation {
	for _, event := range events {
		if event.Mutation != nil && event.Mutation.Type == mutationType {
			return event.Mutation
		}
	}
	return nil
}
