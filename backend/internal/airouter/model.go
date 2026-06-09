package airouter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Scene string

const (
	SceneSummary   Scene = "summary"
	SceneTitle     Scene = "title"
	SceneFlowchart Scene = "flowchart"
)

type Strategy string

const (
	StrategyPriorityFailover   Strategy = "priority_failover"
	StrategyRoundRobinFailover Strategy = "round_robin_failover"
)

type ProviderEndpointMode string

const (
	EndpointModeChatCompletions   ProviderEndpointMode = "chat_completions"
	EndpointModeImagesGenerations ProviderEndpointMode = "images_generations"
)

type ProviderResponseFormat string

const (
	ResponseFormatAuto     ProviderResponseFormat = "auto"
	ResponseFormatImageURL ProviderResponseFormat = "image_url"
	ResponseFormatB64JSON  ProviderResponseFormat = "b64_json"
)

const (
	AdapterOpenAICompatible = "openai-compatible"

	ErrorTypeTimeout         = "timeout"
	ErrorTypeNetwork         = "network"
	ErrorTypeRateLimit       = "rate_limit"
	ErrorTypeAuth            = "auth"
	ErrorTypeUpstream        = "upstream"
	ErrorTypeInvalidResponse = "invalid_response"
	ErrorTypeBadRequest      = "bad_request"
	ErrorTypeBusiness        = "business_validation"
	ErrorTypeBreakerOpen     = "breaker_open"
	ErrorTypeUnknown         = "unknown"
)

type BreakerConfig struct {
	FailureThreshold int `json:"failureThreshold"`
	CooldownSeconds  int `json:"cooldownSeconds"`
}

type RequestOptions struct {
	Stream      bool    `json:"stream,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"maxTokens,omitempty"`
}

type ProviderConfig struct {
	ID             string                 `json:"id"`
	Scene          Scene                  `json:"scene,omitempty"`
	Name           string                 `json:"name"`
	Adapter        string                 `json:"adapter"`
	Enabled        bool                   `json:"enabled"`
	Priority       int                    `json:"priority"`
	Weight         int                    `json:"weight,omitempty"`
	BaseURL        string                 `json:"baseURL"`
	APIKey         string                 `json:"apiKey,omitempty"`
	APIKeyMasked   string                 `json:"apiKeyMasked,omitempty"`
	HasAPIKey      bool                   `json:"hasAPIKey"`
	ClearAPIKey    bool                   `json:"clearApiKey,omitempty"`
	Model          string                 `json:"model"`
	TimeoutSeconds int                    `json:"timeoutSeconds"`
	EndpointMode   ProviderEndpointMode   `json:"endpointMode,omitempty"`
	ResponseFormat ProviderResponseFormat `json:"responseFormat,omitempty"`
	Extra          map[string]any         `json:"extra,omitempty"`
	UpdatedBy      string                 `json:"updatedBySubject,omitempty"`
	UpdatedAt      string                 `json:"updatedAt,omitempty"`
}

type ImageGenerationOptions struct {
	Size              string
	Quality           string
	Background        string
	OutputFormat      string
	OutputCompression *int
	N                 *int
}

type ChatCompletionOptions struct {
	ThinkingType    string
	ReasoningEffort string
}

type SceneConfig struct {
	Scene             Scene            `json:"scene"`
	Enabled           bool             `json:"enabled"`
	Strategy          Strategy         `json:"strategy"`
	MaxAttempts       int              `json:"maxAttempts"`
	RetryOn           []string         `json:"retryOn"`
	Breaker           BreakerConfig    `json:"breaker"`
	RequestOptions    RequestOptions   `json:"requestOptions"`
	Providers         []ProviderConfig `json:"providers"`
	UpdatedBy         string           `json:"updatedBySubject,omitempty"`
	UpdatedAt         string           `json:"updatedAt,omitempty"`
	Source            string           `json:"source,omitempty"`
	CompatibilityMode bool             `json:"compatibilityMode,omitempty"`
}

type SceneSummaryView struct {
	Scene               Scene    `json:"scene"`
	Enabled             bool     `json:"enabled"`
	Strategy            Strategy `json:"strategy"`
	ProviderCount       int      `json:"providerCount"`
	ActiveProviderCount int      `json:"activeProviderCount"`
	UpdatedBy           string   `json:"updatedBySubject,omitempty"`
	UpdatedAt           string   `json:"updatedAt,omitempty"`
	Source              string   `json:"source,omitempty"`
	CompatibilityMode   bool     `json:"compatibilityMode,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionInput struct {
	Messages        []ChatMessage
	Temperature     *float64
	Stream          *bool
	MaxTokens       *int
	ContentKind     string
	AdditionalMeta  map[string]any
	ValidateContent func(string) error
}

type AttemptResult struct {
	ProviderID       string `json:"providerId"`
	ProviderName     string `json:"providerName"`
	Model            string `json:"model"`
	Status           string `json:"status"`
	HTTPStatus       int    `json:"httpStatus"`
	ErrorType        string `json:"errorType,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
	LatencyMS        int64  `json:"latencyMs"`
	SkippedByBreaker bool   `json:"skippedByBreaker,omitempty"`
	BreakerOpenUntil string `json:"breakerOpenUntil,omitempty"`
}

type ChatCompletionResult struct {
	Content         string          `json:"content"`
	ProviderID      string          `json:"providerId"`
	ProviderName    string          `json:"providerName"`
	Model           string          `json:"model"`
	Strategy        Strategy        `json:"strategy"`
	StartedProvider string          `json:"startedProvider"`
	FallbackUsed    bool            `json:"fallbackUsed"`
	AttemptCount    int             `json:"attemptCount"`
	Attempts        []AttemptResult `json:"attempts"`
}

type TestResult struct {
	OK            bool            `json:"ok"`
	Message       string          `json:"message"`
	FinalProvider string          `json:"finalProvider,omitempty"`
	FinalModel    string          `json:"finalModel,omitempty"`
	Attempts      []AttemptResult `json:"attempts"`
}

type typedError struct {
	errorType  string
	message    string
	httpStatus int
	cause      error
}

func (e *typedError) Error() string {
	return strings.TrimSpace(e.message)
}

func (e *typedError) Unwrap() error {
	return e.cause
}

func (e *typedError) HTTPStatus() int {
	return e.httpStatus
}

func (e *typedError) AuditErrorType() string {
	return e.errorType
}

func AllScenes() []Scene {
	return []Scene{SceneSummary, SceneTitle, SceneFlowchart}
}

func IsValidScene(value string) bool {
	switch Scene(strings.TrimSpace(value)) {
	case SceneSummary, SceneTitle, SceneFlowchart:
		return true
	default:
		return false
	}
}

func IsValidStrategy(value string) bool {
	switch Strategy(strings.TrimSpace(value)) {
	case StrategyPriorityFailover, StrategyRoundRobinFailover:
		return true
	default:
		return false
	}
}

func DefaultRetryOn() []string {
	return []string{
		ErrorTypeTimeout,
		ErrorTypeNetwork,
		ErrorTypeRateLimit,
		ErrorTypeUpstream,
		ErrorTypeAuth,
		ErrorTypeInvalidResponse,
	}
}

func ParseProviderEndpointMode(value string) (ProviderEndpointMode, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "chat", "chat_completions", "chat/completions":
		return EndpointModeChatCompletions, true
	case "images", "images_generations", "images/generations":
		return EndpointModeImagesGenerations, true
	default:
		return "", false
	}
}

func NormalizeProviderEndpointMode(value string) ProviderEndpointMode {
	mode, ok := ParseProviderEndpointMode(value)
	if !ok {
		return EndpointModeChatCompletions
	}
	return mode
}

func IsValidProviderEndpointMode(value string) bool {
	_, ok := ParseProviderEndpointMode(value)
	return ok
}

func ParseProviderResponseFormat(value string) (ProviderResponseFormat, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return ResponseFormatAuto, true
	case "image_url", "image-url", "url":
		return ResponseFormatImageURL, true
	case "b64_json", "b64-json", "base64":
		return ResponseFormatB64JSON, true
	default:
		return "", false
	}
}

func NormalizeProviderResponseFormat(value string) ProviderResponseFormat {
	format, ok := ParseProviderResponseFormat(value)
	if !ok {
		return ResponseFormatAuto
	}
	return format
}

func IsValidProviderResponseFormat(value string) bool {
	_, ok := ParseProviderResponseFormat(value)
	return ok
}

func DefaultImageGenerationOptions() ImageGenerationOptions {
	return ImageGenerationOptions{
		OutputFormat: "png",
	}
}

func ImageGenerationOptionsFromExtra(extra map[string]any) ImageGenerationOptions {
	options := DefaultImageGenerationOptions()
	if len(extra) == 0 {
		return options
	}
	if value := extraStringValue(extra, providerExtraKeyImageSize); value != "" {
		options.Size = value
	}
	if value := extraStringValue(extra, providerExtraKeyImageQuality); value != "" {
		options.Quality = value
	}
	if value := extraStringValue(extra, providerExtraKeyImageBackground); value != "" {
		options.Background = value
	}
	if value := extraStringValue(extra, providerExtraKeyImageOutputFormat); value != "" {
		options.OutputFormat = value
	}
	if value, ok := extraIntValue(extra, providerExtraKeyImageOutputCompression); ok {
		options.OutputCompression = &value
	}
	if value, ok := extraIntValue(extra, providerExtraKeyImageN); ok {
		options.N = &value
	}
	return options
}

func ChatCompletionOptionsFromExtra(extra map[string]any) ChatCompletionOptions {
	var options ChatCompletionOptions
	if len(extra) == 0 {
		return options
	}
	if value := extraStringValue(extra, providerExtraKeyThinkingType); value != "" {
		options.ThinkingType = strings.ToLower(strings.TrimSpace(value))
	}
	if value := extraStringValue(extra, providerExtraKeyReasoningEffort); value != "" {
		options.ReasoningEffort = strings.ToLower(strings.TrimSpace(value))
	}
	return options
}

func NormalizeImageOutputFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "jpg", "jpeg":
		return "jpeg"
	case "webp":
		return "webp"
	case "", "png":
		return "png"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func ImageMIMEType(outputFormat string) string {
	switch NormalizeImageOutputFormat(outputFormat) {
	case "jpeg":
		return "jpeg"
	case "webp":
		return "webp"
	default:
		return "png"
	}
}

func IsGPTImageModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "gpt-image-")
}

func ShouldSendImageResponseFormat(model string, responseFormat ProviderResponseFormat) bool {
	if responseFormat == ResponseFormatAuto {
		return false
	}
	return !IsGPTImageModel(model)
}

func ValidateImageGenerationOptions(options ImageGenerationOptions) error {
	if options.Size != "" && !isValidImageSize(options.Size) {
		return fmt.Errorf("provider image size is invalid")
	}
	if options.Quality != "" && !isValidImageQuality(options.Quality) {
		return fmt.Errorf("provider image quality is invalid")
	}
	if options.Background != "" && !isValidImageBackground(options.Background) {
		return fmt.Errorf("provider image background is invalid")
	}
	if options.OutputFormat != "" && !isValidImageOutputFormat(options.OutputFormat) {
		return fmt.Errorf("provider image outputFormat is invalid")
	}
	if options.OutputCompression != nil {
		if *options.OutputCompression < 0 || *options.OutputCompression > 100 {
			return fmt.Errorf("provider image outputCompression must be between 0 and 100")
		}
		if NormalizeImageOutputFormat(options.OutputFormat) == "png" {
			return fmt.Errorf("provider image outputCompression only applies to jpeg or webp")
		}
	}
	if options.N != nil && (*options.N < 1 || *options.N > 10) {
		return fmt.Errorf("provider image n must be between 1 and 10")
	}
	return nil
}

func ValidateChatCompletionOptions(options ChatCompletionOptions) error {
	if options.ThinkingType != "" && !isValidThinkingType(options.ThinkingType) {
		return fmt.Errorf("provider thinkingType must be auto, enabled or disabled")
	}
	if options.ReasoningEffort != "" && !isValidReasoningEffort(options.ReasoningEffort) {
		return fmt.Errorf("provider reasoningEffort must be high or max")
	}
	if normalizeThinkingType(options.ThinkingType) == "disabled" && options.ReasoningEffort != "" {
		return fmt.Errorf("provider reasoningEffort cannot be set when thinking is disabled")
	}
	return nil
}

func ValidateProviderExtra(extra map[string]any, endpointMode ProviderEndpointMode) error {
	if endpointMode == EndpointModeImagesGenerations {
		return ValidateImageGenerationOptions(ImageGenerationOptionsFromExtra(extra))
	}
	return ValidateChatCompletionOptions(ChatCompletionOptionsFromExtra(extra))
}

func ImageGenerationExtraForPersistence(extra map[string]any, endpointMode ProviderEndpointMode) (map[string]any, error) {
	cloned := cloneProviderExtra(extra)
	if cloned == nil {
		cloned = make(map[string]any)
	}
	if endpointMode != EndpointModeImagesGenerations {
		deleteImageGenerationExtra(cloned)
		return cloned, nil
	}

	options := ImageGenerationOptionsFromExtra(cloned)
	options.OutputFormat = NormalizeImageOutputFormat(options.OutputFormat)
	if err := ValidateImageGenerationOptions(options); err != nil {
		return nil, err
	}

	setOrDeleteString(cloned, providerExtraKeyImageSize, options.Size)
	setOrDeleteString(cloned, providerExtraKeyImageQuality, strings.ToLower(strings.TrimSpace(options.Quality)))
	setOrDeleteString(cloned, providerExtraKeyImageBackground, strings.ToLower(strings.TrimSpace(options.Background)))
	setOrDeleteString(cloned, providerExtraKeyImageOutputFormat, options.OutputFormat)
	if options.OutputCompression == nil {
		delete(cloned, providerExtraKeyImageOutputCompression)
	} else {
		cloned[providerExtraKeyImageOutputCompression] = *options.OutputCompression
	}
	if options.N == nil {
		delete(cloned, providerExtraKeyImageN)
	} else {
		cloned[providerExtraKeyImageN] = *options.N
	}
	return cloned, nil
}

func ChatCompletionExtraForPersistence(extra map[string]any, endpointMode ProviderEndpointMode) (map[string]any, error) {
	cloned := cloneProviderExtra(extra)
	if cloned == nil {
		cloned = make(map[string]any)
	}
	if endpointMode == EndpointModeImagesGenerations {
		deleteChatCompletionExtra(cloned)
		return cloned, nil
	}

	options := ChatCompletionOptionsFromExtra(cloned)
	if err := ValidateChatCompletionOptions(options); err != nil {
		return nil, err
	}
	setOrDeleteString(cloned, providerExtraKeyThinkingType, normalizeThinkingType(options.ThinkingType))
	setOrDeleteString(cloned, providerExtraKeyReasoningEffort, options.ReasoningEffort)
	return cloned, nil
}

func DefaultBreakerConfig() BreakerConfig {
	return BreakerConfig{
		FailureThreshold: 3,
		CooldownSeconds:  60,
	}
}

func isValidImageSize(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "auto" {
		return true
	}
	parts := strings.Split(value, "x")
	if len(parts) != 2 {
		return false
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil || width <= 0 {
		return false
	}
	height, err := strconv.Atoi(parts[1])
	return err == nil && height > 0
}

func isValidImageQuality(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto", "low", "medium", "high", "standard", "hd":
		return true
	default:
		return false
	}
}

func isValidImageBackground(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto", "opaque", "transparent":
		return true
	default:
		return false
	}
}

func isValidImageOutputFormat(value string) bool {
	switch NormalizeImageOutputFormat(value) {
	case "", "png", "jpeg", "webp":
		return true
	default:
		return false
	}
}

func isValidThinkingType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto", "enabled", "disabled":
		return true
	default:
		return false
	}
}

func normalizeThinkingType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "enabled", "disabled":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func isValidReasoningEffort(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "high", "max":
		return true
	default:
		return false
	}
}

func deleteImageGenerationExtra(extra map[string]any) {
	delete(extra, providerExtraKeyImageSize)
	delete(extra, providerExtraKeyImageQuality)
	delete(extra, providerExtraKeyImageBackground)
	delete(extra, providerExtraKeyImageOutputFormat)
	delete(extra, providerExtraKeyImageOutputCompression)
	delete(extra, providerExtraKeyImageN)
}

func deleteChatCompletionExtra(extra map[string]any) {
	delete(extra, providerExtraKeyThinkingType)
	delete(extra, providerExtraKeyReasoningEffort)
}

func setOrDeleteString(extra map[string]any, key, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		delete(extra, key)
		return
	}
	extra[key] = value
}

func extraIntValue(extra map[string]any, key string) (int, bool) {
	if len(extra) == 0 {
		return 0, false
	}
	value, ok := extra[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case uint:
		return int(typed), true
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return int(typed), true
	case uint64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := strconv.Atoi(typed.String())
		return parsed, err == nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		parsed, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(typed)))
		return parsed, err == nil
	}
}
