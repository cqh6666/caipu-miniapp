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

const systemPrompt = `你是菜谱小程序里的“饮食管家”。请用中文自然对话，回答要简洁、具体、适合家庭做饭场景。
你可以提供菜谱灵感、食材搭配、备菜建议、菜单安排思路和记录菜谱的文字草稿。
如果用户让你保存、下单、真正写入美食库或调用外部链接解析，请说明当前只能给出建议，需要用户点击页面里的记录入口继续操作。
不要编造已经完成保存、已访问用户美食库、已解析外部链接或已安排菜单。`

type Options struct {
	BaseURL        string
	APIKey         string
	Model          string
	Timeout        time.Duration
	HTTPClient     *http.Client
	NowForTest     func() time.Time
	DisableTimeout bool
}

type Service struct {
	baseURL    string
	apiKey     string
	model      string
	timeout    time.Duration
	httpClient *http.Client
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
	return &Service{
		baseURL:    strings.TrimRight(strings.TrimSpace(options.BaseURL), "/"),
		apiKey:     strings.TrimSpace(options.APIKey),
		model:      strings.TrimSpace(options.Model),
		timeout:    timeout,
		httpClient: client,
	}
}

func (s *Service) StreamChat(ctx context.Context, messages []ChatMessage, emit func(StreamEvent) error) error {
	if s == nil {
		return common.ErrInternal
	}
	if strings.TrimSpace(s.baseURL) == "" || strings.TrimSpace(s.model) == "" || strings.TrimSpace(s.apiKey) == "" {
		return common.NewAppError(common.CodeInternalServer, "diet assistant ai is not configured", http.StatusServiceUnavailable)
	}
	if emit == nil {
		return common.ErrInternal
	}

	payload := openAIChatRequest{
		Model:    s.model,
		Messages: buildUpstreamMessages(messages),
		Stream:   true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return common.ErrInternal.WithErr(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return common.ErrInternal.WithErr(err)
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
		return common.NewAppError(common.CodeInternalServer, "diet assistant upstream request failed", http.StatusBadGateway).WithErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = fmt.Sprintf("diet assistant upstream returned status %d", resp.StatusCode)
		}
		return common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway)
	}

	return consumeOpenAIStream(resp.Body, emit)
}

func buildUpstreamMessages(messages []ChatMessage) []ChatMessage {
	result := []ChatMessage{{Role: "system", Content: systemPrompt}}
	for _, message := range trimHistory(messages, 16) {
		role := strings.TrimSpace(strings.ToLower(message.Role))
		content := strings.TrimSpace(message.Content)
		if content == "" {
			continue
		}
		if role != "user" && role != "assistant" {
			continue
		}
		result = append(result, ChatMessage{
			Role:    role,
			Content: content,
		})
	}
	return result
}

func trimHistory(messages []ChatMessage, max int) []ChatMessage {
	if max <= 0 || len(messages) <= max {
		return messages
	}
	return messages[len(messages)-max:]
}

type openAIChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
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
