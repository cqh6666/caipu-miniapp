package dietassistant

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Options struct {
	BaseURL        string
	APIKey         string
	Model          string
	Timeout        time.Duration
	HTTPClient     *http.Client
	CountRecipes   CountRecipesFunc
	SearchRecipes  SearchRecipesFunc
	CreateFromURL  CreateFromURLFunc
	Repo           *Repository
	EnsureMember   EnsureMemberFunc
	NowForTest     func() time.Time
	DisableTimeout bool
}

type CountRecipesFunc func(context.Context, RecipeCountInput) (int, error)
type SearchRecipesFunc func(context.Context, RecipeSearchInput) ([]RecipeToolItem, error)
type CreateFromURLFunc func(context.Context, RecipeFromURLInput) (RecipeFromURLResult, error)
type EnsureMemberFunc func(context.Context, int64, int64) error

type Service struct {
	baseURL       string
	apiKey        string
	model         string
	timeout       time.Duration
	httpClient    *http.Client
	countRecipes  CountRecipesFunc
	searchRecipes SearchRecipesFunc
	createFromURL CreateFromURLFunc
	repo          *Repository
	ensureMember  EnsureMemberFunc
	now           func() time.Time
}

func NewService(options Options) *Service {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	client := options.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	now := options.NowForTest
	if now == nil {
		now = time.Now
	}
	return &Service{
		baseURL:       strings.TrimRight(strings.TrimSpace(options.BaseURL), "/"),
		apiKey:        strings.TrimSpace(options.APIKey),
		model:         strings.TrimSpace(options.Model),
		timeout:       timeout,
		httpClient:    client,
		countRecipes:  options.CountRecipes,
		searchRecipes: options.SearchRecipes,
		createFromURL: options.CreateFromURL,
		repo:          options.Repo,
		ensureMember:  options.EnsureMember,
		now:           now,
	}
}

func (s *Service) StreamChat(ctx context.Context, chatCtx ChatContext, messages []ChatMessage, emit func(StreamEvent) error) error {
	if s == nil {
		return common.ErrInternal
	}
	if strings.TrimSpace(s.baseURL) == "" || strings.TrimSpace(s.model) == "" || strings.TrimSpace(s.apiKey) == "" {
		return common.NewAppError(common.CodeInternalServer, "diet assistant ai is not configured", http.StatusServiceUnavailable)
	}
	if emit == nil {
		return common.ErrInternal
	}
	if err := s.ensureStorageContext(ctx, chatCtx); err != nil {
		return err
	}

	upstreamMessages, err := buildAgentUpstreamMessages(messages)
	if err != nil {
		return err
	}
	lastUserContent := lastUserMessageContent(messages)

	finalMessages := upstreamMessages
	if forcedCall, ok := buildURLOnlyParseToolCall(lastUserContent); ok {
		finalMessages = append([]openAIChatMessage{}, upstreamMessages...)
		finalMessages, err = s.appendToolResults(ctx, chatCtx, finalMessages, openAIChatMessage{
			Role:      "assistant",
			Content:   "",
			ToolCalls: []openAIToolCall{forcedCall},
		}, []openAIToolCall{forcedCall}, emit)
		if err != nil {
			return err
		}
	} else {
		if err := emit(StreamEvent{Type: "status", Message: "正在判断需要调用的能力"}); err != nil {
			return err
		}
		toolResponse, err := s.createChatCompletion(ctx, openAIChatRequest{
			Model:       s.model,
			User:        buildUpstreamUser(chatCtx),
			Messages:    upstreamMessages,
			Tools:       dietAssistantTools(),
			ToolChoice:  "auto",
			Stream:      false,
			MaxTokens:   900,
			Temperature: floatPtr(0.2),
		})
		if err != nil {
			return err
		}

		if len(toolResponse.Choices) == 0 {
			return common.NewAppError(common.CodeInternalServer, "diet assistant upstream returned no choices", http.StatusBadGateway)
		}

		assistantMessage := toolResponse.Choices[0].Message
		toolCalls := normalizeToolCallIDs(assistantMessage.ToolCalls)
		if len(toolCalls) > 0 {
			finalMessages = append([]openAIChatMessage{}, upstreamMessages...)
			finalMessages, err = s.appendToolResults(ctx, chatCtx, finalMessages, openAIChatMessage{
				Role:      valueOrDefault(assistantMessage.Role, "assistant"),
				Content:   assistantMessage.Content,
				ToolCalls: toolCalls,
			}, toolCalls, emit)
			if err != nil {
				return err
			}
		}
	}

	assistantContent, err := s.streamFinalChat(ctx, chatCtx, finalMessages, emit)
	if err != nil {
		return err
	}
	if err := emit(StreamEvent{Type: "done"}); err != nil {
		return err
	}
	_ = s.storeCompletedTurn(ctx, chatCtx, lastUserContent, assistantContent)
	return nil
}

func (s *Service) appendToolResults(ctx context.Context, chatCtx ChatContext, messages []openAIChatMessage, assistantMessage openAIChatMessage, toolCalls []openAIToolCall, emit func(StreamEvent) error) ([]openAIChatMessage, error) {
	messages = append(messages, assistantMessage)
	for _, call := range toolCalls {
		toolName := strings.TrimSpace(call.Function.Name)
		if emit != nil {
			if err := emit(StreamEvent{Type: "tool_start", ToolName: toolName, Message: toolStatusMessage(toolName, "start")}); err != nil {
				return nil, err
			}
		}
		result := s.executeTool(ctx, chatCtx, call)
		if emit != nil {
			eventType := "tool_done"
			message := toolStatusMessage(toolName, "done")
			var mutation *StreamMutation
			if toolResultFailed(result) {
				eventType = "tool_error"
				message = toolStatusMessage(toolName, "error")
			} else {
				mutation = buildToolMutation(toolName, result)
			}
			if err := emit(StreamEvent{Type: eventType, ToolName: toolName, Message: message, Mutation: mutation}); err != nil {
				return nil, err
			}
		}
		messages = append(messages, openAIChatMessage{
			Role:       "tool",
			Content:    mustJSON(result),
			ToolCallID: call.ID,
			Name:       call.Function.Name,
		})
	}
	return messages, nil
}

func (s *Service) streamFinalChat(ctx context.Context, chatCtx ChatContext, messages []openAIChatMessage, emit func(StreamEvent) error) (string, error) {
	payload := openAIChatRequest{
		Model:       s.model,
		User:        buildUpstreamUser(chatCtx),
		Messages:    messages,
		Stream:      true,
		MaxTokens:   1200,
		Temperature: floatPtr(0.7),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", common.ErrInternal.WithErr(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", common.ErrInternal.WithErr(err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: s.timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", common.NewAppError(common.CodeInternalServer, "diet assistant upstream request failed", http.StatusBadGateway).WithErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = fmt.Sprintf("diet assistant upstream returned status %d", resp.StatusCode)
		}
		return "", common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway)
	}

	var content strings.Builder
	err = consumeOpenAIStream(resp.Body, func(event StreamEvent) error {
		if event.Type == "delta" {
			content.WriteString(event.Delta)
			return emit(event)
		}
		if event.Type == "done" {
			return nil
		}
		return emit(event)
	})
	if err != nil {
		return "", err
	}
	return content.String(), nil
}

func (s *Service) ListStoredMessages(ctx context.Context, chatCtx ChatContext, limit int) ([]StoredMessage, error) {
	if err := s.ensureStorageContext(ctx, chatCtx); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.ListMessages(ctx, chatCtx.UserID, chatCtx.KitchenID, normalizeMessageLimit(limit))
}

func (s *Service) ClearStoredMessages(ctx context.Context, chatCtx ChatContext) error {
	if err := s.ensureStorageContext(ctx, chatCtx); err != nil {
		return err
	}
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.ClearMessages(ctx, chatCtx.UserID, chatCtx.KitchenID)
}

func (s *Service) storeCompletedTurn(ctx context.Context, chatCtx ChatContext, userContent, assistantContent string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if err := s.ensureStorageContext(ctx, chatCtx); err != nil {
		return err
	}
	userContent = strings.TrimSpace(userContent)
	assistantContent = strings.TrimSpace(assistantContent)
	if userContent == "" || assistantContent == "" {
		return nil
	}
	now := time.Now
	if s.now != nil {
		now = s.now
	}
	return s.repo.AddTurn(ctx, chatCtx.UserID, chatCtx.KitchenID, userContent, assistantContent, now().UTC().Format(time.RFC3339))
}

func (s *Service) ensureStorageContext(ctx context.Context, chatCtx ChatContext) error {
	if err := validateStorageContext(chatCtx); err != nil {
		return err
	}
	if s != nil && s.ensureMember != nil {
		return s.ensureMember(ctx, chatCtx.UserID, chatCtx.KitchenID)
	}
	return nil
}

func buildAgentUpstreamMessages(messages []ChatMessage) ([]openAIChatMessage, error) {
	result := []openAIChatMessage{{
		Role:    "system",
		Content: dietAssistantSystemPrompt,
	}}
	hasUser := false
	for _, message := range messages {
		role := strings.TrimSpace(strings.ToLower(message.Role))
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			continue
		}
		if role == "user" {
			hasUser = true
		}
		result = append(result, openAIChatMessage{
			Role:    role,
			Content: content,
		})
	}
	if hasUser {
		return result, nil
	}

	return nil, common.NewAppError(common.CodeBadRequest, "user message is required", http.StatusBadRequest)
}

func lastUserMessageContent(messages []ChatMessage) string {
	for index := len(messages) - 1; index >= 0; index -= 1 {
		message := messages[index]
		if strings.EqualFold(strings.TrimSpace(message.Role), "user") {
			return strings.TrimSpace(message.Content)
		}
	}
	return ""
}

func buildURLOnlyParseToolCall(content string) (openAIToolCall, bool) {
	rawURL, ok := singleURLOnly(content)
	if !ok {
		return openAIToolCall{}, false
	}
	return openAIToolCall{
		ID:   "call_diet_assistant_url_parse",
		Type: "function",
		Function: openAIToolCallFunction{
			Name: "parse_and_add_recipe_from_url",
			Arguments: map[string]any{
				"url":      rawURL,
				"mealType": "main",
				"status":   "wishlist",
			},
		},
	}, true
}

func singleURLOnly(content string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(content))
	if len(fields) != 1 {
		return "", false
	}
	rawURL := strings.Trim(fields[0], "<>")
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if parsed.Host == "" {
		return "", false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return rawURL, true
	default:
		return "", false
	}
}

func validateStorageContext(chatCtx ChatContext) error {
	if chatCtx.UserID <= 0 {
		return common.ErrUnauthorized
	}
	if chatCtx.KitchenID <= 0 {
		return common.NewAppError(common.CodeBadRequest, "kitchenId is required", http.StatusBadRequest)
	}
	return nil
}

func normalizeMessageLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	User        string              `json:"user,omitempty"`
	Messages    []openAIChatMessage `json:"messages"`
	Tools       []openAITool        `json:"tools,omitempty"`
	ToolChoice  any                 `json:"tool_choice,omitempty"`
	Stream      bool                `json:"stream"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature *float64            `json:"temperature,omitempty"`
}

type openAIChatMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openAIToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function openAIToolCallFunction `json:"function"`
}

type openAIToolCallFunction struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

type openAIChatResponse struct {
	Choices []struct {
		FinishReason string            `json:"finish_reason"`
		Message      openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const dietAssistantSystemPrompt = `你是“饮食管家”，服务于一个家庭/共享空间的菜谱小程序。

能力边界：
- 用户问美食库里有多少菜、早餐/正餐数量、想吃/吃过数量时，调用 get_recipe_count。
- 用户要求查找、搜索、确认是否已有某道菜时，调用 search_recipes_by_name；只按菜谱名模糊查询。
- 用户只发送或要求保存 B 站 / 小红书菜谱链接时，调用 parse_and_add_recipe_from_url。该工具会解析链接内容，提取食材和步骤，并真正写入当前空间的美食库。
- 当前不提供单独添加食材工具；如果用户只说添加食材但没有菜谱链接或菜名，先追问需要记录哪道菜或让用户提供链接。
- 不要编造菜谱数量、搜索结果或保存状态；涉及数量、查询和保存必须基于工具结果。
- 最终回复使用中文，简洁、自然、可执行。`

func (s *Service) createChatCompletion(ctx context.Context, payload openAIChatRequest) (openAIChatResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return openAIChatResponse{}, common.ErrInternal.WithErr(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return openAIChatResponse{}, common.ErrInternal.WithErr(err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := s.httpClient
	if client == nil {
		client = &http.Client{Timeout: s.timeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return openAIChatResponse{}, common.NewAppError(common.CodeInternalServer, "diet assistant upstream request failed", http.StatusBadGateway).WithErr(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return openAIChatResponse{}, common.NewAppError(common.CodeInternalServer, "diet assistant upstream response failed", http.StatusBadGateway).WithErr(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = fmt.Sprintf("diet assistant upstream returned status %d", resp.StatusCode)
		}
		return openAIChatResponse{}, common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway)
	}

	var result openAIChatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return openAIChatResponse{}, common.NewAppError(common.CodeInternalServer, "invalid diet assistant upstream response", http.StatusBadGateway).WithErr(err)
	}
	if result.Error != nil && strings.TrimSpace(result.Error.Message) != "" {
		return openAIChatResponse{}, common.NewAppError(common.CodeInternalServer, strings.TrimSpace(result.Error.Message), http.StatusBadGateway)
	}
	return result, nil
}

func dietAssistantTools() []openAITool {
	return []openAITool{
		{
			Type: "function",
			Function: openAIToolFunction{
				Name:        "get_recipe_count",
				Description: "查询当前美食库中符合条件的菜谱数量。",
				Parameters: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"mealType": map[string]any{
							"type":        "string",
							"description": "餐别过滤：breakfast=早餐，main=正餐，all=全部。",
							"enum":        []string{"breakfast", "main", "all"},
						},
						"status": map[string]any{
							"type":        "string",
							"description": "状态过滤：wishlist=想吃，done=吃过，all=全部。",
							"enum":        []string{"wishlist", "done", "all"},
						},
					},
					"required": []string{"mealType", "status"},
				},
			},
		},
		{
			Type: "function",
			Function: openAIToolFunction{
				Name:        "parse_and_add_recipe_from_url",
				Description: "根据 B 站或小红书菜谱链接解析内容，提取食材和步骤，并保存为当前空间的一道菜谱。用户只发送链接或明确要求保存链接菜谱时调用。",
				Parameters: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"url": map[string]any{
							"type":        "string",
							"description": "B 站或小红书菜谱链接，必须是用户提供的原始 URL。",
						},
						"mealType": map[string]any{
							"type":        "string",
							"description": "餐别：breakfast=早餐，main=正餐。无法判断时用 main。",
							"enum":        []string{"breakfast", "main"},
						},
						"status": map[string]any{
							"type":        "string",
							"description": "状态：wishlist=想吃，done=吃过。无法判断时用 wishlist。",
							"enum":        []string{"wishlist", "done"},
						},
					},
					"required": []string{"url", "mealType", "status"},
				},
			},
		},
		{
			Type: "function",
			Function: openAIToolFunction{
				Name:        "search_recipes_by_name",
				Description: "按菜谱名模糊查询当前空间的菜谱。用户询问是否已有某道菜、查找菜谱、按名称搜索时调用。",
				Parameters: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"titleKeyword": map[string]any{
							"type":        "string",
							"description": "菜名关键词，例如“番茄”“鸡胸肉沙拉”。",
						},
						"mealType": map[string]any{
							"type":        "string",
							"description": "餐别过滤：breakfast=早餐，main=正餐，all=全部。",
							"enum":        []string{"breakfast", "main", "all"},
						},
						"status": map[string]any{
							"type":        "string",
							"description": "状态过滤：wishlist=想吃，done=吃过，all=全部。",
							"enum":        []string{"wishlist", "done", "all"},
						},
						"limit": map[string]any{
							"type":        "integer",
							"description": "最多返回数量，建议 5，最大 10。",
						},
					},
					"required": []string{"titleKeyword", "mealType", "status", "limit"},
				},
			},
		},
	}
}

func (s *Service) executeTool(ctx context.Context, chatCtx ChatContext, call openAIToolCall) map[string]any {
	name := strings.TrimSpace(call.Function.Name)
	switch name {
	case "get_recipe_count":
		return s.executeGetRecipeCount(ctx, chatCtx, call)
	case "search_recipes_by_name":
		return s.executeSearchRecipesByName(ctx, chatCtx, call)
	case "parse_and_add_recipe_from_url":
		return s.executeParseAndAddRecipeFromURL(ctx, chatCtx, call)
	default:
		return map[string]any{
			"ok":    false,
			"error": "unknown tool: " + name,
		}
	}
}

func buildToolMutation(name string, result map[string]any) *StreamMutation {
	switch name {
	case "parse_and_add_recipe_from_url":
		recipe, ok := result["recipe"].(RecipeToolItem)
		if !ok || strings.TrimSpace(recipe.ID) == "" {
			return nil
		}
		return &StreamMutation{
			Type:        "recipe_created",
			RecipeID:    strings.TrimSpace(recipe.ID),
			RecipeTitle: strings.TrimSpace(recipe.Title),
			MealType:    strings.TrimSpace(recipe.MealType),
			Status:      strings.TrimSpace(recipe.Status),
		}
	default:
		return nil
	}
}

func toolResultFailed(result map[string]any) bool {
	value, ok := result["ok"]
	if !ok {
		return false
	}
	if passed, ok := value.(bool); ok {
		return !passed
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprint(value)), "false")
}

func toolStatusMessage(name, stage string) string {
	displayName := toolDisplayName(name)
	switch stage {
	case "start":
		switch name {
		case "get_recipe_count":
			return "正在统计美食库"
		case "search_recipes_by_name":
			return "正在查找菜谱"
		case "parse_and_add_recipe_from_url":
			return "正在解析链接并保存食材"
		default:
			return "正在调用" + displayName
		}
	case "done":
		switch name {
		case "get_recipe_count":
			return "已完成菜谱统计"
		case "search_recipes_by_name":
			return "已完成菜谱查找"
		case "parse_and_add_recipe_from_url":
			return "已解析并保存食材"
		default:
			return displayName + "调用完成"
		}
	case "error":
		switch name {
		case "get_recipe_count":
			return "菜谱统计失败，正在整理说明"
		case "search_recipes_by_name":
			return "菜谱查找失败，正在整理说明"
		case "parse_and_add_recipe_from_url":
			return "链接解析保存失败，正在整理说明"
		default:
			return displayName + "调用失败，正在整理说明"
		}
	default:
		return displayName
	}
}

func toolDisplayName(name string) string {
	switch name {
	case "get_recipe_count":
		return "美食库统计"
	case "search_recipes_by_name":
		return "菜谱查找"
	case "parse_and_add_recipe_from_url":
		return "链接解析保存"
	default:
		if strings.TrimSpace(name) == "" {
			return "工具"
		}
		return strings.TrimSpace(name)
	}
}

func (s *Service) executeGetRecipeCount(ctx context.Context, chatCtx ChatContext, call openAIToolCall) map[string]any {
	if s.countRecipes == nil {
		return map[string]any{"ok": false, "error": "recipe count tool is not configured"}
	}
	if chatCtx.UserID <= 0 {
		return map[string]any{"ok": false, "error": "user is required"}
	}
	if chatCtx.KitchenID <= 0 {
		return map[string]any{"ok": false, "error": "current kitchen is required"}
	}

	args, err := parseToolArguments(call.Function.Arguments)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	mealType := normalizeToolEnum(fmt.Sprint(args["mealType"]), "all")
	status := normalizeToolEnum(fmt.Sprint(args["status"]), "all")
	if !isAllowedRecipeCountMealType(mealType) {
		return map[string]any{"ok": false, "error": "invalid mealType: " + mealType}
	}
	if !isAllowedRecipeCountStatus(status) {
		return map[string]any{"ok": false, "error": "invalid status: " + status}
	}

	input := RecipeCountInput{
		UserID:    chatCtx.UserID,
		KitchenID: chatCtx.KitchenID,
		MealType:  emptyIfAll(mealType),
		Status:    emptyIfAll(status),
	}
	count, err := s.countRecipes(ctx, input)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{
		"ok":        true,
		"count":     count,
		"mealType":  mealType,
		"status":    status,
		"kitchenId": chatCtx.KitchenID,
	}
}

func (s *Service) executeParseAndAddRecipeFromURL(ctx context.Context, chatCtx ChatContext, call openAIToolCall) map[string]any {
	if s.createFromURL == nil {
		return map[string]any{"ok": false, "error": "recipe url create tool is not configured"}
	}
	if chatCtx.UserID <= 0 {
		return map[string]any{"ok": false, "error": "user is required"}
	}
	if chatCtx.KitchenID <= 0 {
		return map[string]any{"ok": false, "error": "current kitchen is required"}
	}

	args, err := parseToolArguments(call.Function.Arguments)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	rawURL := truncateRunes(toolStringArg(args, "url"), 500)
	if rawURL == "" {
		return map[string]any{"ok": false, "error": "url is required"}
	}
	mealType := normalizeToolEnum(fmt.Sprint(args["mealType"]), "main")
	status := normalizeToolEnum(fmt.Sprint(args["status"]), "wishlist")
	if mealType != "breakfast" && mealType != "main" {
		mealType = "main"
	}
	if status != "wishlist" && status != "done" {
		status = "wishlist"
	}

	result, err := s.createFromURL(ctx, RecipeFromURLInput{
		UserID:    chatCtx.UserID,
		KitchenID: chatCtx.KitchenID,
		URL:       rawURL,
		MealType:  mealType,
		Status:    status,
	})
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	return map[string]any{
		"ok":                   true,
		"message":              "链接已解析，菜谱和食材已保存到美食库。",
		"kitchenId":            chatCtx.KitchenID,
		"recipe":               result.Recipe,
		"source":               result.Source,
		"sourceDetail":         result.SourceDetail,
		"summaryMode":          result.SummaryMode,
		"mainIngredients":      result.MainIngredients,
		"secondaryIngredients": result.SecondaryIngredients,
		"stepsCount":           result.StepsCount,
		"warnings":             result.Warnings,
	}
}

func (s *Service) executeSearchRecipesByName(ctx context.Context, chatCtx ChatContext, call openAIToolCall) map[string]any {
	if s.searchRecipes == nil {
		return map[string]any{"ok": false, "error": "recipe search tool is not configured"}
	}
	if chatCtx.UserID <= 0 {
		return map[string]any{"ok": false, "error": "user is required"}
	}
	if chatCtx.KitchenID <= 0 {
		return map[string]any{"ok": false, "error": "current kitchen is required"}
	}

	args, err := parseToolArguments(call.Function.Arguments)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	titleKeyword := toolStringArg(args, "titleKeyword")
	if titleKeyword == "" {
		return map[string]any{"ok": false, "error": "titleKeyword is required"}
	}
	mealType := normalizeToolEnum(fmt.Sprint(args["mealType"]), "all")
	status := normalizeToolEnum(fmt.Sprint(args["status"]), "all")
	if !isAllowedRecipeCountMealType(mealType) {
		return map[string]any{"ok": false, "error": "invalid mealType: " + mealType}
	}
	if !isAllowedRecipeCountStatus(status) {
		return map[string]any{"ok": false, "error": "invalid status: " + status}
	}

	limit := normalizeToolLimit(args["limit"], 5, 10)
	items, err := s.searchRecipes(ctx, RecipeSearchInput{
		UserID:       chatCtx.UserID,
		KitchenID:    chatCtx.KitchenID,
		TitleKeyword: titleKeyword,
		MealType:     emptyIfAll(mealType),
		Status:       emptyIfAll(status),
		Limit:        limit,
	})
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	return map[string]any{
		"ok":           true,
		"titleKeyword": titleKeyword,
		"mealType":     mealType,
		"status":       status,
		"count":        len(items),
		"items":        items,
	}
}

func parseToolArguments(value any) (map[string]any, error) {
	switch v := value.(type) {
	case string:
		return parseToolArgumentBytes([]byte(v))
	case map[string]any:
		return v, nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return parseToolArgumentBytes(data)
	}
}

func parseToolArgumentBytes(data []byte) (map[string]any, error) {
	var args map[string]any
	if err := json.Unmarshal(data, &args); err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}
	return args, nil
}

func normalizeToolCallIDs(calls []openAIToolCall) []openAIToolCall {
	result := append([]openAIToolCall{}, calls...)
	for index := range result {
		if strings.TrimSpace(result[index].ID) == "" {
			result[index].ID = fmt.Sprintf("call_diet_assistant_%d", index+1)
		}
		if strings.TrimSpace(result[index].Type) == "" {
			result[index].Type = "function"
		}
	}
	return result
}

func normalizeToolEnum(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "<nil>" {
		return fallback
	}
	return value
}

func normalizeToolLimit(value any, fallback, max int) int {
	limit := 0
	switch v := value.(type) {
	case float64:
		limit = int(v)
	case int:
		limit = v
	case int64:
		limit = int(v)
	case json.Number:
		parsed, err := v.Int64()
		if err == nil {
			limit = int(parsed)
		}
	default:
		text := strings.TrimSpace(fmt.Sprint(v))
		if text != "" && text != "<nil>" {
			_, _ = fmt.Sscanf(text, "%d", &limit)
		}
	}
	if limit <= 0 {
		limit = fallback
	}
	if max > 0 && limit > max {
		limit = max
	}
	return limit
}

func isAllowedRecipeCountMealType(value string) bool {
	switch value {
	case "all", "breakfast", "main":
		return true
	default:
		return false
	}
}

func isAllowedRecipeCountStatus(value string) bool {
	switch value {
	case "all", "wishlist", "done":
		return true
	default:
		return false
	}
}

func emptyIfAll(value string) string {
	if value == "all" {
		return ""
	}
	return value
}

func toolStringArg(args map[string]any, key string) string {
	value, ok := args[key]
	if !ok || value == nil {
		return ""
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "<nil>" {
		return ""
	}
	return text
}

func truncateRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func buildUpstreamUser(chatCtx ChatContext) string {
	if chatCtx.UserID <= 0 {
		return ""
	}
	return fmt.Sprintf("user-%d", chatCtx.UserID)
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func floatPtr(value float64) *float64 {
	return &value
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf(`{"ok":false,"error":%q}`, err.Error())
	}
	return string(data)
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content json.RawMessage `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func consumeOpenAIStream(reader io.Reader, emit func(StreamEvent) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			return emit(StreamEvent{Type: "done"})
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return common.NewAppError(common.CodeInternalServer, "invalid diet assistant stream chunk", http.StatusBadGateway).WithErr(err)
		}
		if chunk.Error != nil && strings.TrimSpace(chunk.Error.Message) != "" {
			return common.NewAppError(common.CodeInternalServer, strings.TrimSpace(chunk.Error.Message), http.StatusBadGateway)
		}

		for _, choice := range chunk.Choices {
			delta := extractDeltaContent(choice.Delta.Content)
			if delta == "" {
				continue
			}
			if err := emit(StreamEvent{Type: "delta", Delta: delta}); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return common.NewAppError(common.CodeInternalServer, "diet assistant stream interrupted", http.StatusBadGateway).WithErr(err)
	}
	return emit(StreamEvent{Type: "done"})
}

func extractDeltaContent(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if strings.TrimSpace(part.Text) != "" {
				values = append(values, part.Text)
			}
		}
		return strings.Join(values, "")
	}

	return strings.TrimSpace(string(raw))
}
