package airouter

import (
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
	ID             string         `json:"id"`
	Scene          Scene          `json:"scene,omitempty"`
	Name           string         `json:"name"`
	Adapter        string         `json:"adapter"`
	Enabled        bool           `json:"enabled"`
	Priority       int            `json:"priority"`
	Weight         int            `json:"weight,omitempty"`
	BaseURL        string         `json:"baseURL"`
	APIKey         string         `json:"apiKey,omitempty"`
	APIKeyMasked   string         `json:"apiKeyMasked,omitempty"`
	HasAPIKey      bool           `json:"hasAPIKey"`
	ClearAPIKey    bool           `json:"clearApiKey,omitempty"`
	Model          string         `json:"model"`
	TimeoutSeconds int            `json:"timeoutSeconds"`
	Extra          map[string]any `json:"extra,omitempty"`
	UpdatedBy      string         `json:"updatedBySubject,omitempty"`
	UpdatedAt      string         `json:"updatedAt,omitempty"`
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

func DefaultBreakerConfig() BreakerConfig {
	return BreakerConfig{
		FailureThreshold: 3,
		CooldownSeconds:  60,
	}
}
