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

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type openAIChatRequest struct {
	Model           string                `json:"model"`
	User            string                `json:"user,omitempty"`
	Messages        []openAIChatMessage   `json:"messages"`
	Tools           []openAITool          `json:"tools,omitempty"`
	ToolChoice      any                   `json:"tool_choice,omitempty"`
	Thinking        *openAIThinkingConfig `json:"thinking,omitempty"`
	ReasoningEffort string                `json:"reasoning_effort,omitempty"`
	Stream          bool                  `json:"stream"`
	MaxTokens       int                   `json:"max_tokens,omitempty"`
	Temperature     *float64              `json:"temperature,omitempty"`
}

type openAIChatMessage struct {
	Role             string           `json:"role"`
	Content          any              `json:"content"`
	ReasoningContent *string          `json:"reasoning_content,omitempty"`
	ToolCalls        []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string           `json:"tool_call_id,omitempty"`
	Name             string           `json:"name,omitempty"`
}

type openAIThinkingConfig struct {
	Type string `json:"type"`
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
- 用户要求查找、搜索、确认是否已有某道菜或某种食材相关菜谱时，调用 search_recipes_by_name；只按菜谱名或食材模糊查询，默认返回 5 条，最多 10 条。
- 用户提供菜谱 ID 并要求查看菜谱详情、食材或步骤时，调用 get_recipe_by_id。
- 用户只发送或要求保存 B 站 / 小红书菜谱链接时，调用 parse_and_add_recipe_from_url。该工具会解析链接内容，提取食材和步骤，并真正写入当前空间的美食库。
- 当前不提供单独添加食材工具；如果用户只说添加食材但没有菜谱链接或菜名，先追问需要记录哪道菜或让用户提供链接。
- 不要编造菜谱数量、搜索结果或保存状态；涉及数量、查询和保存必须基于工具结果。
- 最终回复使用中文，简洁、自然、可执行。`

func (s *Service) createChatCompletion(ctx context.Context, payload openAIChatRequest) (openAIChatResponse, error) {
	s.applyRequestOptions(&payload)
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

func normalizeThinkingType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "enabled", "disabled":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizeReasoningEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "high", "max":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func (s *Service) applyRequestOptions(payload *openAIChatRequest) {
	if s == nil || payload == nil {
		return
	}
	if s.thinkingType != "" {
		payload.Thinking = &openAIThinkingConfig{Type: s.thinkingType}
	}
	if s.reasoningEffort != "" && s.thinkingType != "disabled" {
		payload.ReasoningEffort = s.reasoningEffort
	}
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
