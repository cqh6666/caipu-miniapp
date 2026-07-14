package dietassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Options struct {
	BaseURL         string
	APIKey          string
	Model           string
	ThinkingType    string
	ReasoningEffort string
	Timeout         time.Duration
	HTTPClient      *http.Client
	CountRecipes    CountRecipesFunc
	SearchRecipes   SearchRecipesFunc
	GetRecipeByID   GetRecipeByIDFunc
	CreateFromURL   CreateFromURLFunc
	Repo            *Repository
	EnsureMember    EnsureMemberFunc
	NowForTest      func() time.Time
	DisableTimeout  bool
}

type CountRecipesFunc func(context.Context, RecipeCountInput) (int, error)
type SearchRecipesFunc func(context.Context, RecipeSearchInput) ([]RecipeToolItem, error)
type GetRecipeByIDFunc func(context.Context, RecipeGetInput) (RecipeDetailToolItem, error)
type CreateFromURLFunc func(context.Context, RecipeFromURLInput) (RecipeFromURLResult, error)
type EnsureMemberFunc func(context.Context, int64, int64) error

type Service struct {
	baseURL         string
	apiKey          string
	model           string
	thinkingType    string
	reasoningEffort string
	timeout         time.Duration
	httpClient      *http.Client
	countRecipes    CountRecipesFunc
	searchRecipes   SearchRecipesFunc
	getRecipeByID   GetRecipeByIDFunc
	createFromURL   CreateFromURLFunc
	repo            *Repository
	ensureMember    EnsureMemberFunc
	now             func() time.Time
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
		baseURL:         strings.TrimRight(strings.TrimSpace(options.BaseURL), "/"),
		apiKey:          strings.TrimSpace(options.APIKey),
		model:           strings.TrimSpace(options.Model),
		thinkingType:    normalizeThinkingType(options.ThinkingType),
		reasoningEffort: normalizeReasoningEffort(options.ReasoningEffort),
		timeout:         timeout,
		httpClient:      client,
		countRecipes:    options.CountRecipes,
		searchRecipes:   options.SearchRecipes,
		getRecipeByID:   options.GetRecipeByID,
		createFromURL:   options.CreateFromURL,
		repo:            options.Repo,
		ensureMember:    options.EnsureMember,
		now:             now,
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
		longcatMarkupFound := false
		if len(toolCalls) == 0 {
			toolCalls, longcatMarkupFound = parseLongCatToolCalls(openAIContentText(assistantMessage.Content))
		}
		if len(toolCalls) > 0 {
			finalMessages = append([]openAIChatMessage{}, upstreamMessages...)
			finalMessages, err = s.appendToolResults(ctx, chatCtx, finalMessages, openAIChatMessage{
				Role:             valueOrDefault(assistantMessage.Role, "assistant"),
				Content:          sanitizeAssistantVisibleContent(openAIContentText(assistantMessage.Content)),
				ReasoningContent: assistantMessage.ReasoningContent,
				ToolCalls:        toolCalls,
			}, toolCalls, emit)
			if err != nil {
				return err
			}
		} else if longcatMarkupFound {
			return common.NewAppError(common.CodeInternalServer, "diet assistant upstream returned invalid tool call markup", http.StatusBadGateway)
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

const maxFinalToolRounds = 3

type finalStreamResult struct {
	VisibleContent string
	ToolCalls      []openAIToolCall
	MarkupFound    bool
}

func (s *Service) streamFinalChat(ctx context.Context, chatCtx ChatContext, messages []openAIChatMessage, emit func(StreamEvent) error) (string, error) {
	finalMessages := append([]openAIChatMessage{}, messages...)
	var assistantContent strings.Builder
	for round := 0; round <= maxFinalToolRounds; round += 1 {
		result, err := s.streamFinalChatOnce(ctx, chatCtx, finalMessages, emit)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(result.VisibleContent) != "" {
			assistantContent.WriteString(result.VisibleContent)
		}
		if len(result.ToolCalls) == 0 {
			if result.MarkupFound {
				return "", common.NewAppError(common.CodeInternalServer, "diet assistant upstream returned invalid tool call markup", http.StatusBadGateway)
			}
			return assistantContent.String(), nil
		}
		if round >= maxFinalToolRounds {
			return "", common.NewAppError(common.CodeInternalServer, "diet assistant upstream returned too many nested tool calls", http.StatusBadGateway)
		}
		toolCalls := renameToolCallIDs(result.ToolCalls, fmt.Sprintf("call_diet_assistant_longcat_stream_%d", round+1))
		finalMessages, err = s.appendToolResults(ctx, chatCtx, finalMessages, openAIChatMessage{
			Role:      "assistant",
			Content:   sanitizeAssistantVisibleContent(result.VisibleContent),
			ToolCalls: toolCalls,
		}, toolCalls, emit)
		if err != nil {
			return "", err
		}
	}
	return "", common.NewAppError(common.CodeInternalServer, "diet assistant upstream returned too many nested tool calls", http.StatusBadGateway)
}

func (s *Service) streamFinalChatOnce(ctx context.Context, chatCtx ChatContext, messages []openAIChatMessage, emit func(StreamEvent) error) (finalStreamResult, error) {
	payload := openAIChatRequest{
		Model:       s.model,
		User:        buildUpstreamUser(chatCtx),
		Messages:    messages,
		Stream:      true,
		MaxTokens:   1200,
		Temperature: floatPtr(0.7),
	}
	s.applyRequestOptions(&payload)
	body, err := json.Marshal(payload)
	if err != nil {
		return finalStreamResult{}, common.ErrInternal.WithErr(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return finalStreamResult{}, common.ErrInternal.WithErr(err)
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
		return finalStreamResult{}, common.NewAppError(common.CodeInternalServer, "diet assistant upstream request failed", http.StatusBadGateway).WithErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = fmt.Sprintf("diet assistant upstream returned status %d", resp.StatusCode)
		}
		return finalStreamResult{}, common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway)
	}

	filter := newLongCatStreamFilter(func(delta string) error {
		return emit(StreamEvent{Type: "delta", Delta: delta})
	})
	err = consumeOpenAIStream(resp.Body, func(event StreamEvent) error {
		if event.Type == "delta" {
			return filter.Push(event.Delta)
		}
		if event.Type == "done" {
			return filter.Flush()
		}
		return emit(event)
	})
	if err != nil {
		return finalStreamResult{}, err
	}
	if err := filter.Flush(); err != nil {
		return finalStreamResult{}, err
	}
	return finalStreamResult{
		VisibleContent: filter.VisibleContent(),
		ToolCalls:      filter.ToolCalls(),
		MarkupFound:    filter.MarkupFound(),
	}, nil
}

func (s *Service) ListStoredMessages(ctx context.Context, chatCtx ChatContext, limit int) ([]StoredMessage, error) {
	if err := s.ensureStorageContext(ctx, chatCtx); err != nil {
		return nil, err
	}
	if s == nil || s.repo == nil {
		return nil, nil
	}
	items, err := s.repo.ListMessages(ctx, chatCtx.UserID, chatCtx.KitchenID, normalizeMessageLimit(limit))
	if err != nil {
		return nil, err
	}
	return sanitizeStoredMessages(items), nil
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
	assistantContent = sanitizeAssistantVisibleContent(assistantContent)
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
		if role == "assistant" {
			content = sanitizeAssistantVisibleContent(content)
		}
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

func openAIContentText(content any) string {
	switch value := content.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case []any:
		values := make([]string, 0, len(value))
		for _, item := range value {
			text := openAIContentText(item)
			if text != "" {
				values = append(values, text)
			}
		}
		return strings.TrimSpace(strings.Join(values, ""))
	case map[string]any:
		text := strings.TrimSpace(fmt.Sprint(value["text"]))
		if text != "" && text != "<nil>" {
			return text
		}
	}

	data, err := json.Marshal(content)
	if err != nil {
		return strings.TrimSpace(fmt.Sprint(content))
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		return strings.TrimSpace(text)
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &parts); err == nil {
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if strings.TrimSpace(part.Text) != "" {
				values = append(values, part.Text)
			}
		}
		return strings.TrimSpace(strings.Join(values, ""))
	}
	return strings.TrimSpace(fmt.Sprint(content))
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
