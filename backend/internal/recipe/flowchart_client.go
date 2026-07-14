package recipe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const defaultFlowchartImageOutputFormat = "png"

var (
	flowchartMarkdownImagePattern = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)\)`)
	flowchartPlainURLPattern      = regexp.MustCompile(`https?://[^\s)]+`)
	flowchartDataImageURLPattern  = regexp.MustCompile(`data:image/[a-zA-Z0-9.+-]+;base64,[A-Za-z0-9+/=]+`)
)

type flowchartClient struct {
	baseURL        string
	apiKey         string
	model          string
	endpointMode   airouter.ProviderEndpointMode
	responseFormat airouter.ProviderResponseFormat
	httpClient     *http.Client
	tracker        audit.Tracker
}

type flowchartChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []flowchartChatMessage `json:"messages"`
	Temperature float64                `json:"temperature"`
}

type flowchartChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type flowchartChatResponse struct {
	Choices []struct {
		Message struct {
			Content json.RawMessage `json:"content"`
			Images  []struct {
				Type     string `json:"type"`
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"images"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type flowchartImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Quality        string `json:"quality,omitempty"`
	OutputFormat   string `json:"output_format,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type flowchartImageGenerationResponse struct {
	Data []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *flowchartClient) generate(ctx context.Context, prompt string) (string, error) {
	startedAt := time.Now()
	endpointPath := "/chat/completions"
	if c != nil && c.endpointMode == airouter.EndpointModeImagesGenerations {
		endpointPath = "/images/generations"
	}
	logCall := func(status string, httpStatus int, err error) {
		if c == nil || c.tracker == nil {
			return
		}
		jobCtx, ok := audit.CurrentJobContext(ctx)
		if !ok || jobCtx.JobRunID <= 0 {
			return
		}
		_ = c.tracker.LogCall(ctx, audit.CallLogInput{
			JobRunID:     jobCtx.JobRunID,
			Scene:        jobCtx.Scene,
			Provider:     "openai-compatible",
			Endpoint:     endpointPath,
			Model:        c.model,
			Status:       status,
			HTTPStatus:   httpStatus,
			LatencyMS:    time.Since(startedAt).Milliseconds(),
			ErrorType:    audit.ErrorTypeFromError(err),
			ErrorMessage: flowchartErrorMessage(err),
			RequestID:    common.RequestID(ctx),
			Meta: map[string]any{
				"content_kind": "flowchart",
			},
		})
	}

	body := []byte{}
	switch c.endpointMode {
	case airouter.EndpointModeImagesGenerations:
		payload := flowchartImageGenerationRequest{
			Model:        c.model,
			Prompt:       strings.TrimSpace(prompt),
			OutputFormat: defaultFlowchartImageOutputFormat,
		}
		if payload.Prompt == "" {
			callErr := common.NewAppError(common.CodeBadRequest, "flowchart prompt is empty", http.StatusBadRequest)
			logCall(audit.CallStatusFailed, 0, callErr)
			return "", callErr
		}
		if airouter.ShouldSendImageResponseFormat(c.model, c.responseFormat) {
			payload.ResponseFormat = string(c.responseFormat)
		}
		marshaled, err := json.Marshal(payload)
		if err != nil {
			logCall(audit.CallStatusFailed, 0, err)
			return "", common.ErrInternal.WithErr(fmt.Errorf("marshal flowchart image request: %w", err))
		}
		body = marshaled
	default:
		payload := flowchartChatRequest{
			Model:       c.model,
			Temperature: 0.4,
			Messages: []flowchartChatMessage{
				{
					Role:    "system",
					Content: "你是一个料理流程图生成助手。请严格按用户要求生成手绘风格料理流程信息图，不要输出额外解释。",
				},
				{
					Role:    "user",
					Content: prompt,
				},
			},
		}

		marshaled, err := json.Marshal(payload)
		if err != nil {
			logCall(audit.CallStatusFailed, 0, err)
			return "", common.ErrInternal.WithErr(fmt.Errorf("marshal flowchart request: %w", err))
		}
		body = marshaled
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpointPath, bytes.NewReader(body))
	if err != nil {
		logCall(audit.CallStatusFailed, 0, err)
		return "", common.ErrInternal.WithErr(fmt.Errorf("build flowchart request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		callErr := newFlowchartRequestError(err, c.httpClient.Timeout)
		logCall(audit.CallStatusFromError(err), 0, callErr)
		return "", callErr
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = fmt.Sprintf("flowchart request failed with status %d", resp.StatusCode)
		}
		callErr := common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return "", callErr
	}

	content, err := c.decodeResponse(resp)
	if err != nil {
		callErr := common.NewAppError(common.CodeInternalServer, err.Error(), http.StatusBadGateway).WithErr(err)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return "", callErr
	}

	logCall(audit.CallStatusSuccess, resp.StatusCode, nil)

	return content, nil
}

func (c *flowchartClient) decodeResponse(resp *http.Response) (string, error) {
	switch c.endpointMode {
	case airouter.EndpointModeImagesGenerations:
		var parsed flowchartImageGenerationResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return "", fmt.Errorf("invalid flowchart image response: %w", err)
		}
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return "", errors.New(strings.TrimSpace(parsed.Error.Message))
		}
		content := extractFlowchartGeneratedImageContent(
			parsed.Data,
			c.responseFormat,
			airouter.ImageMIMEType(defaultFlowchartImageOutputFormat),
		)
		if content == "" {
			return "", fmt.Errorf("flowchart image response contained no image")
		}
		return content, nil
	default:
		var parsed flowchartChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return "", fmt.Errorf("invalid flowchart response: %w", err)
		}
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return "", errors.New(strings.TrimSpace(parsed.Error.Message))
		}
		if len(parsed.Choices) == 0 {
			return "", fmt.Errorf("flowchart response contained no choices")
		}
		content := extractFlowchartMessageContent(parsed.Choices[0].Message.Content)
		if imageURL := extractFlowchartMessageImageURL(parsed.Choices[0].Message.Images); imageURL != "" {
			content = imageURL
		}
		if content == "" {
			return "", fmt.Errorf("flowchart response was empty")
		}
		return content, nil
	}
}

func newFlowchartRequestError(err error, timeout time.Duration) error {
	cause := flowchartErrorCause(err)
	if isFlowchartTimeoutError(err) {
		message := "流程图生成超时，上游生图响应较慢"
		if timeout > 0 {
			message = fmt.Sprintf("%s（已等待 %s）", message, timeout.Round(time.Second))
		}
		if cause != "" {
			message += ": " + cause
		}
		return common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway).WithErr(err)
	}

	if cause == "" {
		cause = "unknown error"
	}

	return common.NewAppError(
		common.CodeInternalServer,
		"flowchart request failed: "+truncateString(cause, 180),
		http.StatusBadGateway,
	).WithErr(err)
}

func isFlowchartTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func flowchartErrorCause(err error) string {
	if err == nil {
		return ""
	}

	var appErr *common.AppError
	if errors.As(err, &appErr) {
		parts := make([]string, 0, 2)
		message := strings.TrimSpace(appErr.Message)
		if message != "" {
			parts = append(parts, message)
		}
		if appErr.Err != nil {
			cause := deepestError(appErr.Err)
			if cause != "" && (message == "" || !strings.Contains(message, cause)) {
				parts = append(parts, cause)
			}
		}
		return strings.Join(parts, ": ")
	}

	return deepestError(err)
}

func flowchartErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func deepestError(err error) string {
	if err == nil {
		return ""
	}

	current := err
	for {
		next := errors.Unwrap(current)
		if next == nil {
			break
		}
		current = next
	}

	return strings.TrimSpace(current.Error())
}

func extractFlowchartMessageContent(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		items := make([]string, 0, len(parts))
		for _, part := range parts {
			if strings.TrimSpace(part.Text) == "" {
				continue
			}
			items = append(items, strings.TrimSpace(part.Text))
		}
		return strings.TrimSpace(strings.Join(items, "\n"))
	}

	return strings.TrimSpace(string(raw))
}

func extractFlowchartGeneratedImageContent(items []struct {
	URL     string `json:"url"`
	B64JSON string `json:"b64_json"`
}, responseFormat airouter.ProviderResponseFormat, imageMIMEType string) string {
	imageMIMEType = strings.TrimSpace(imageMIMEType)
	if imageMIMEType == "" {
		imageMIMEType = airouter.ImageMIMEType(defaultFlowchartImageOutputFormat)
	}
	for _, item := range items {
		url := normalizeFlowchartImageReference(item.URL)
		b64 := strings.TrimSpace(item.B64JSON)
		switch responseFormat {
		case airouter.ResponseFormatImageURL:
			if url != "" {
				return url
			}
		case airouter.ResponseFormatB64JSON:
			if b64 != "" {
				return "data:image/" + imageMIMEType + ";base64," + b64
			}
		default:
			if url != "" {
				return url
			}
			if b64 != "" {
				return "data:image/" + imageMIMEType + ";base64," + b64
			}
		}
	}
	return ""
}

func extractFlowchartMessageImageURL(images []struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}) string {
	for _, image := range images {
		if value := normalizeFlowchartImageReference(image.ImageURL.URL); value != "" {
			return value
		}
	}
	return ""
}

func extractFlowchartImageURL(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	if matches := flowchartMarkdownImagePattern.FindStringSubmatch(content); len(matches) == 2 {
		if value := normalizeFlowchartImageReference(matches[1]); value != "" {
			return value
		}
	}
	if dataURL := flowchartDataImageURLPattern.FindString(content); dataURL != "" {
		return normalizeFlowchartImageReference(dataURL)
	}

	for _, candidate := range flowchartPlainURLPattern.FindAllString(content, -1) {
		if value := normalizeFlowchartImageReference(candidate); value != "" {
			return value
		}
	}

	return ""
}

func normalizeFlowchartImageReference(value string) string {
	value = strings.TrimSpace(strings.TrimRight(value, "])}>.,;!\"'"))
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "data:image/"):
		return value
	default:
		return ""
	}
}
