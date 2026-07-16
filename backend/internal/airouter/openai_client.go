package airouter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/logging"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upstream"
)

const (
	maxAIChatResponseBytes            int64 = 2 << 20
	maxAIImageGenerationResponseBytes int64 = 16 << 20
)

type openAIChatRequest struct {
	Model           string                `json:"model"`
	Messages        []ChatMessage         `json:"messages"`
	Temperature     *float64              `json:"temperature,omitempty"`
	Stream          *bool                 `json:"stream,omitempty"`
	MaxTokens       *int                  `json:"max_tokens,omitempty"`
	Thinking        *openAIThinkingConfig `json:"thinking,omitempty"`
	ReasoningEffort string                `json:"reasoning_effort,omitempty"`
}

type openAIThinkingConfig struct {
	Type string `json:"type"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
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

type openAIImageGenerationRequest struct {
	Model             string `json:"model"`
	Prompt            string `json:"prompt"`
	Size              string `json:"size,omitempty"`
	Quality           string `json:"quality,omitempty"`
	Background        string `json:"background,omitempty"`
	N                 *int   `json:"n,omitempty"`
	OutputFormat      string `json:"output_format,omitempty"`
	OutputCompression *int   `json:"output_compression,omitempty"`
	ResponseFormat    string `json:"response_format,omitempty"`
}

type openAIImageGenerationResponse struct {
	Data []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *Service) callOpenAICompatible(ctx context.Context, config SceneConfig, provider orderedProvider, input ChatCompletionInput) (string, string, int, int64, error) {
	startedAt := time.Now()
	endpointMode := NormalizeProviderEndpointMode(string(provider.EndpointMode))
	endpointPath := "/chat/completions"
	var body []byte

	switch endpointMode {
	case EndpointModeImagesGenerations:
		imageOptions := ImageGenerationOptionsFromExtra(provider.Extra)
		imageOptions.OutputFormat = NormalizeImageOutputFormat(imageOptions.OutputFormat)
		endpointPath = "/images/generations"
		request := openAIImageGenerationRequest{
			Model:             provider.Model,
			Prompt:            buildImageGenerationPrompt(input.Messages),
			Size:              strings.TrimSpace(imageOptions.Size),
			Quality:           strings.TrimSpace(imageOptions.Quality),
			Background:        strings.TrimSpace(imageOptions.Background),
			N:                 imageOptions.N,
			OutputFormat:      imageOptions.OutputFormat,
			OutputCompression: imageOptions.OutputCompression,
		}
		if request.Prompt == "" {
			return "", endpointPath, 0, 0, common.NewAppError(common.CodeBadRequest, "image generation prompt is required", http.StatusBadRequest)
		}
		responseFormat := NormalizeProviderResponseFormat(string(provider.ResponseFormat))
		if ShouldSendImageResponseFormat(provider.Model, responseFormat) {
			request.ResponseFormat = string(responseFormat)
		}
		marshaled, err := json.Marshal(request)
		if err != nil {
			return "", endpointPath, 0, 0, common.ErrInternal.WithErr(err)
		}
		body = marshaled
	default:
		chatOptions := ChatCompletionOptionsFromExtra(provider.Extra)
		request := openAIChatRequest{
			Model:    provider.Model,
			Messages: input.Messages,
		}

		stream := config.RequestOptions.Stream
		if input.Stream != nil {
			stream = *input.Stream
		}
		temperature := config.RequestOptions.Temperature
		if input.Temperature != nil {
			temperature = *input.Temperature
		}
		maxTokens := config.RequestOptions.MaxTokens
		if input.MaxTokens != nil {
			maxTokens = *input.MaxTokens
		}

		request.Stream = &stream
		if provider.Scene == SceneTitle || input.Temperature != nil || config.RequestOptions.Temperature != 0 {
			request.Temperature = &temperature
		}
		if maxTokens > 0 {
			request.MaxTokens = &maxTokens
		}
		if chatOptions.ThinkingType != "" {
			request.Thinking = &openAIThinkingConfig{Type: chatOptions.ThinkingType}
		}
		if chatOptions.ReasoningEffort != "" && chatOptions.ThinkingType != "disabled" {
			request.ReasoningEffort = chatOptions.ReasoningEffort
		}

		marshaled, err := json.Marshal(request)
		if err != nil {
			return "", endpointPath, 0, 0, common.ErrInternal.WithErr(err)
		}
		body = marshaled
	}

	timeout := time.Duration(provider.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, strings.TrimRight(provider.BaseURL, "/")+endpointPath, bytes.NewReader(body))
	if err != nil {
		return "", endpointPath, 0, 0, common.ErrInternal.WithErr(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(provider.APIKey) != "" {
		plain := strings.TrimSpace(provider.APIKey)
		if provider.apiKeyEncrypted {
			decrypted, decryptErr := s.cipherBox.Decrypt(provider.APIKey)
			if decryptErr != nil {
				return "", endpointPath, 0, time.Since(startedAt).Milliseconds(), &typedError{
					errorType:  ErrorTypeAuth,
					message:    "provider credential could not be decrypted",
					httpStatus: http.StatusBadGateway,
					cause:      decryptErr,
				}
			}
			plain = strings.TrimSpace(decrypted)
		}
		req.Header.Set("Authorization", "Bearer "+plain)
	}

	doer := normalizeHTTPDoer(s.httpDoer)
	resp, err := doer.Do(req)
	if err != nil {
		return "", endpointPath, 0, time.Since(startedAt).Milliseconds(), classifyRequestError(err, timeout)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), classifyHTTPError(resp.StatusCode, strings.TrimSpace(string(data)))
	}

	switch endpointMode {
	case EndpointModeImagesGenerations:
		var parsed openAIImageGenerationResponse
		if err := upstream.DecodeJSON(resp.Body, maxAIImageGenerationResponseBytes, &parsed); err != nil {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), invalidUpstreamResponseError("invalid image generation response", err)
		}
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), classifyHTTPError(resp.StatusCode, strings.TrimSpace(parsed.Error.Message))
		}
		content := extractGeneratedImageContent(
			parsed.Data,
			NormalizeProviderResponseFormat(string(provider.ResponseFormat)),
			ImageMIMEType(ImageGenerationOptionsFromExtra(provider.Extra).OutputFormat),
		)
		if content == "" {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), &typedError{
				errorType:  ErrorTypeInvalidResponse,
				message:    "image generation response contained no image",
				httpStatus: http.StatusBadGateway,
			}
		}
		return content, endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), nil
	default:
		var parsed openAIChatResponse
		if err := upstream.DecodeJSON(resp.Body, maxAIChatResponseBytes, &parsed); err != nil {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), invalidUpstreamResponseError("invalid chat completion response", err)
		}
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), classifyHTTPError(resp.StatusCode, strings.TrimSpace(parsed.Error.Message))
		}
		if len(parsed.Choices) == 0 {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), &typedError{
				errorType:  ErrorTypeInvalidResponse,
				message:    "chat completion response contained no choices",
				httpStatus: http.StatusBadGateway,
			}
		}

		content := extractMessageContent(parsed.Choices[0].Message.Content)
		if provider.Scene == SceneFlowchart {
			if imageURL := extractMessageImageURL(parsed.Choices[0].Message.Images); imageURL != "" {
				content = imageURL
			}
		}
		if content == "" {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), &typedError{
				errorType:  ErrorTypeInvalidResponse,
				message:    "chat completion response was empty",
				httpStatus: http.StatusBadGateway,
			}
		}

		return content, endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), nil
	}
}

func (s *Service) logCall(ctx context.Context, config SceneConfig, provider orderedProvider, attempt int, endpoint string, httpStatus int, latencyMS int64, err error, input ChatCompletionInput) {
	if s == nil || s.tracker == nil {
		return
	}
	jobCtx, ok := audit.CurrentJobContext(ctx)
	if !ok || jobCtx.JobRunID <= 0 {
		return
	}

	meta := map[string]any{
		"scene":               string(config.Scene),
		"route_strategy":      string(config.Strategy),
		"attempt":             attempt,
		"provider_adapter":    provider.Adapter,
		"is_fallback_attempt": attempt > 1,
	}
	if input.ContentKind != "" {
		meta["content_kind"] = input.ContentKind
	}
	for key, value := range input.AdditionalMeta {
		meta[key] = value
	}

	status := audit.CallStatusSuccess
	if err != nil {
		status = audit.CallStatusFromError(err)
	}
	_ = s.tracker.LogCall(ctx, audit.CallLogInput{
		JobRunID:     jobCtx.JobRunID,
		Scene:        jobCtx.Scene,
		Provider:     provider.ID,
		Endpoint:     endpoint,
		Model:        provider.Model,
		Status:       status,
		HTTPStatus:   httpStatus,
		LatencyMS:    latencyMS,
		ErrorType:    routeErrorType(err),
		ErrorMessage: logging.SafeErrorSummary(err),
		RequestID:    common.RequestID(ctx),
		Meta:         meta,
	})
}

func invalidUpstreamResponseError(message string, err error) *typedError {
	errorType := ErrorTypeInvalidResponse
	if upstream.IsResponseTooLarge(err) {
		errorType = ErrorTypeResponseTooLarge
		message = "upstream response exceeded size limit"
	}
	return &typedError{
		errorType:  errorType,
		message:    message,
		httpStatus: http.StatusBadGateway,
		cause:      err,
	}
}

func extractMessageContent(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		lines := make([]string, 0, len(parts))
		for _, part := range parts {
			if strings.TrimSpace(part.Text) == "" {
				continue
			}
			lines = append(lines, strings.TrimSpace(part.Text))
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}

	return strings.TrimSpace(string(raw))
}

func buildImageGenerationPrompt(messages []ChatMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		text := strings.TrimSpace(message.Content)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func extractGeneratedImageContent(items []struct {
	URL     string `json:"url"`
	B64JSON string `json:"b64_json"`
}, responseFormat ProviderResponseFormat, imageMIMEType string) string {
	imageMIMEType = strings.TrimSpace(imageMIMEType)
	if imageMIMEType == "" {
		imageMIMEType = ImageMIMEType(defaultImageOutputFormat)
	}
	for _, item := range items {
		url := normalizeImageReference(item.URL)
		b64 := strings.TrimSpace(item.B64JSON)
		switch responseFormat {
		case ResponseFormatImageURL:
			if url != "" {
				return url
			}
		case ResponseFormatB64JSON:
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

func extractMessageImageURL(images []struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}) string {
	for _, image := range images {
		if value := normalizeImageReference(image.ImageURL.URL); value != "" {
			return value
		}
	}
	return ""
}
