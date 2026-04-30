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

func TestServiceStreamChatParsesLongCatTaggedToolMarkup(t *testing.T) {
	var requestCount int
	var sawToolResult bool
	var sawLongCatMarkupInFinalRequest bool
	var gotSearchInput RecipeSearchInput
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
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"<longcat_tool_call>search_recipes_by_name\n<longcat_arg_key>query</longcat_arg_key>\n<longcat_arg_value>清淡</longcat_arg_value>\n<longcat_arg_key>limit</longcat_arg_key>\n<longcat_arg_value>5</longcat_arg_value>\n</longcat_tool_call>"},"finish_reason":"stop"}]}`))
			return
		}

		for _, message := range req.Messages {
			if strings.Contains(openAIContentText(message.Content), "<longcat_tool_call>") {
				sawLongCatMarkupInFinalRequest = true
			}
			if message.Role != "tool" {
				continue
			}
			if strings.Contains(message.Content.(string), `"keyword":"清淡"`) && strings.Contains(message.Content.(string), `"searchScope":"title_or_ingredient"`) {
				sawToolResult = true
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"给你找了几道清淡口味的菜。\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	service := NewService(Options{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "LongCat-2.0-Preview",
		Timeout: 3 * time.Second,
		SearchRecipes: func(ctx context.Context, input RecipeSearchInput) ([]RecipeToolItem, error) {
			gotSearchInput = input
			return []RecipeToolItem{{
				ID:       "rec_1",
				Title:    "清蒸鲈鱼",
				MealType: "main",
				Status:   "wishlist",
			}}, nil
		},
	})

	var content strings.Builder
	err := service.StreamChat(context.Background(), ChatContext{UserID: 5, KitchenID: 6}, []ChatMessage{
		{Role: "user", Content: "来点清淡点的"},
	}, func(event StreamEvent) error {
		if event.Type == "delta" {
			content.WriteString(event.Delta)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChat returned error: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("requestCount = %d, want 2", requestCount)
	}
	if !sawToolResult {
		t.Fatal("final request did not include parsed longcat tool result")
	}
	if sawLongCatMarkupInFinalRequest {
		t.Fatal("final request should not carry raw longcat tool markup")
	}
	if gotSearchInput.Keyword != "清淡" || gotSearchInput.SearchScope != "title_or_ingredient" || gotSearchInput.Limit != 5 {
		t.Fatalf("unexpected search input: %#v", gotSearchInput)
	}
	if got, want := content.String(), "给你找了几道清淡口味的菜。"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
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
	if !names["get_recipe_by_id"] {
		t.Fatal("get_recipe_by_id should be exposed as a diet assistant tool")
	}
	if got, want := toolStatusMessage("get_recipe_by_id", "start"), "正在读取菜谱详情"; got != want {
		t.Fatalf("get_recipe_by_id start status = %q, want %q", got, want)
	}
}

func TestExecuteGetRecipeByID(t *testing.T) {
	var gotInput RecipeGetInput
	service := NewService(Options{
		GetRecipeByID: func(ctx context.Context, input RecipeGetInput) (RecipeDetailToolItem, error) {
			gotInput = input
			return RecipeDetailToolItem{
				RecipeToolItem: RecipeToolItem{
					ID:         input.RecipeID,
					Title:      "番茄炒蛋",
					MealType:   "main",
					Status:     "wishlist",
					Ingredient: "番茄、鸡蛋",
					Summary:    "家常快手菜",
					Link:       "https://example.com/recipe",
				},
				MainIngredients:      []string{"番茄 2 个", "鸡蛋 3 个"},
				SecondaryIngredients: []string{"盐 少许"},
				Steps: []RecipeStepToolItem{{
					Title:  "炒蛋",
					Detail: "鸡蛋炒到凝固后盛出。",
				}},
				StepsCount: 1,
			}, nil
		},
	})

	missingID := service.executeGetRecipeByID(context.Background(), ChatContext{UserID: 1, KitchenID: 2}, openAIToolCall{
		Function: openAIToolCallFunction{
			Name:      "get_recipe_by_id",
			Arguments: `{}`,
		},
	})
	if missingID["ok"] != false {
		t.Fatalf("missingID ok = %#v, want false", missingID["ok"])
	}

	result := service.executeGetRecipeByID(context.Background(), ChatContext{UserID: 7, KitchenID: 8}, openAIToolCall{
		Function: openAIToolCallFunction{
			Name:      "get_recipe_by_id",
			Arguments: `{"recipeId":"rec_123"}`,
		},
	})
	if result["ok"] != true {
		t.Fatalf("result ok = %#v, want true", result["ok"])
	}
	if gotInput.UserID != 7 || gotInput.KitchenID != 8 || gotInput.RecipeID != "rec_123" {
		t.Fatalf("input = %#v", gotInput)
	}
	recipe, ok := result["recipe"].(RecipeDetailToolItem)
	if !ok {
		t.Fatalf("recipe = %#v, want RecipeDetailToolItem", result["recipe"])
	}
	if recipe.ID != "rec_123" || recipe.Title != "番茄炒蛋" || recipe.StepsCount != 1 {
		t.Fatalf("recipe = %#v", recipe)
	}
	if len(recipe.MainIngredients) != 2 || len(recipe.Steps) != 1 {
		t.Fatalf("recipe detail = %#v", recipe)
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
			Arguments: `{"keyword":"鸡蛋","searchScope":"ingredient","mealType":"all","status":"wishlist","limit":30}`,
		},
	})
	if result["ok"] != true {
		t.Fatalf("result ok = %#v, want true", result["ok"])
	}
	if gotInput.UserID != 7 || gotInput.KitchenID != 8 {
		t.Fatalf("input context = %#v", gotInput)
	}
	if gotInput.Keyword != "鸡蛋" || gotInput.SearchScope != "ingredient" || gotInput.MealType != "" || gotInput.Status != "wishlist" || gotInput.Limit != 10 {
		t.Fatalf("input filters = %#v", gotInput)
	}
	if result["limit"] != 10 {
		t.Fatalf("result limit = %#v, want 10", result["limit"])
	}
	if result["count"] != 1 {
		t.Fatalf("result count = %#v, want 1", result["count"])
	}
}

func TestExecuteSearchRecipesByNameDefaultsLimit(t *testing.T) {
	var gotInput RecipeSearchInput
	service := NewService(Options{
		SearchRecipes: func(ctx context.Context, input RecipeSearchInput) ([]RecipeToolItem, error) {
			gotInput = input
			return nil, nil
		},
	})

	result := service.executeSearchRecipesByName(context.Background(), ChatContext{UserID: 7, KitchenID: 8}, openAIToolCall{
		Function: openAIToolCallFunction{
			Name:      "search_recipes_by_name",
			Arguments: `{"keyword":"番茄","searchScope":"title_or_ingredient","mealType":"all","status":"all"}`,
		},
	})
	if result["ok"] != true {
		t.Fatalf("result ok = %#v, want true", result["ok"])
	}
	if gotInput.Keyword != "番茄" || gotInput.SearchScope != "title_or_ingredient" || gotInput.Limit != 5 {
		t.Fatalf("input filters = %#v", gotInput)
	}
	if result["limit"] != 5 {
		t.Fatalf("result limit = %#v, want 5", result["limit"])
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

func TestServiceListStoredMessagesSkipsLongCatToolMarkup(t *testing.T) {
	db := openDietAssistantTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	if err := repo.AddTurn(context.Background(), 1, 2, "今晚吃啥", "<longcat_tool_call>search_recipes_by_name\n<longcat_arg_key>query</longcat_arg_key>\n<longcat_arg_value>家常</longcat_arg_value>\n</longcat_tool_call>", "2026-05-01T00:00:00Z"); err != nil {
		t.Fatalf("AddTurn error = %v", err)
	}

	service := NewService(Options{
		Repo: repo,
	})

	items, err := service.ListStoredMessages(context.Background(), ChatContext{UserID: 1, KitchenID: 2}, 50)
	if err != nil {
		t.Fatalf("ListStoredMessages error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Role != "user" || items[0].Content != "今晚吃啥" {
		t.Fatalf("unexpected item = %#v", items[0])
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
