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
	Repo           *Repository
	EnsureMember   EnsureMemberFunc
	NowForTest     func() time.Time
	DisableTimeout bool
}

type CountRecipesFunc func(context.Context, RecipeCountInput) (int, error)
type EnsureMemberFunc func(context.Context, int64, int64) error

type Service struct {
	baseURL      string
	apiKey       string
	model        string
	timeout      time.Duration
	httpClient   *http.Client
	countRecipes CountRecipesFunc
	repo         *Repository
	ensureMember EnsureMemberFunc
	now          func() time.Time
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
		baseURL:      strings.TrimRight(strings.TrimSpace(options.BaseURL), "/"),
		apiKey:       strings.TrimSpace(options.APIKey),
		model:        strings.TrimSpace(options.Model),
		timeout:      timeout,
		httpClient:   client,
		countRecipes: options.CountRecipes,
		repo:         options.Repo,
		ensureMember: options.EnsureMember,
		now:          now,
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
	finalMessages := upstreamMessages
	if len(toolCalls) == 0 {
		finalMessages = upstreamMessages
	} else {
		finalMessages = append([]openAIChatMessage{}, upstreamMessages...)
		finalMessages = append(finalMessages, openAIChatMessage{
			Role:      valueOrDefault(assistantMessage.Role, "assistant"),
			Content:   assistantMessage.Content,
			ToolCalls: toolCalls,
		})
		for _, call := range toolCalls {
			result := s.executeTool(ctx, chatCtx, call)
			finalMessages = append(finalMessages, openAIChatMessage{
				Role:       "tool",
				Content:    mustJSON(result),
				ToolCallID: call.ID,
				Name:       call.Function.Name,
			})
		}
	}

	assistantContent, err := s.streamFinalChat(ctx, chatCtx, finalMessages, emit)
	if err != nil {
		return err
	}
	_ = s.storeCompletedTurn(ctx, chatCtx, lastUserContent, assistantContent)
	return emit(StreamEvent{Type: "done"})
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
- 用户要求添加、记录、保存一道菜时，调用 add_recipe_mock。这个工具只模拟添加，不会真正写入数据库。
- 如果 add_recipe_mock 返回成功，最终回复必须明确说明“本次只是模拟添加，还没有真正保存到美食库”。
- 不要编造菜谱数量；涉及数量必须基于工具结果。
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
				Name:        "add_recipe_mock",
				Description: "模拟添加一道菜谱。仅返回将要保存的字段，不真正写入美食库。",
				Parameters: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"title": map[string]any{
							"type":        "string",
							"description": "菜名，必须尽量简短。",
						},
						"mealType": map[string]any{
							"type":        "string",
							"description": "餐别：breakfast=早餐，main=正餐。",
							"enum":        []string{"breakfast", "main"},
						},
						"status": map[string]any{
							"type":        "string",
							"description": "状态：wishlist=想吃，done=吃过。",
							"enum":        []string{"wishlist", "done"},
						},
						"ingredient": map[string]any{
							"type":        "string",
							"description": "主要食材，未知时可为空字符串。",
						},
						"summary": map[string]any{
							"type":        "string",
							"description": "一句话摘要，未知时可为空字符串。",
						},
						"note": map[string]any{
							"type":        "string",
							"description": "用户备注或做法草稿，未知时可为空字符串。",
						},
					},
					"required": []string{"title", "mealType", "status", "ingredient", "summary", "note"},
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
	case "add_recipe_mock":
		return executeAddRecipeMock(call)
	default:
		return map[string]any{
			"ok":    false,
			"error": "unknown tool: " + name,
		}
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

func executeAddRecipeMock(call openAIToolCall) map[string]any {
	args, err := parseToolArguments(call.Function.Arguments)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}

	title := toolStringArg(args, "title")
	if title == "" {
		return map[string]any{"ok": false, "error": "title is required"}
	}
	mealType := normalizeToolEnum(fmt.Sprint(args["mealType"]), "main")
	status := normalizeToolEnum(fmt.Sprint(args["status"]), "wishlist")
	if mealType != "breakfast" && mealType != "main" {
		mealType = "main"
	}
	if status != "wishlist" && status != "done" {
		status = "wishlist"
	}

	return map[string]any{
		"ok":        true,
		"simulated": true,
		"message":   "模拟添加成功，未真正写入数据库。",
		"recipe": map[string]any{
			"title":      title,
			"mealType":   mealType,
			"status":     status,
			"ingredient": toolStringArg(args, "ingredient"),
			"summary":    toolStringArg(args, "summary"),
			"note":       toolStringArg(args, "note"),
		},
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
