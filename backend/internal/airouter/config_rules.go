package airouter

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
)

func extraStringValue(extra map[string]any, key string) string {
	if len(extra) == 0 {
		return ""
	}
	value, ok := extra[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func cloneProviderExtra(extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(extra))
	for key, value := range extra {
		cloned[key] = value
	}
	return cloned
}

func providerExtraForPersistence(extra map[string]any, endpointMode ProviderEndpointMode, responseFormat ProviderResponseFormat) map[string]any {
	cloned, err := ImageGenerationExtraForPersistence(extra, endpointMode)
	if err != nil {
		cloned = cloneProviderExtra(extra)
	}
	if normalized, chatErr := ChatCompletionExtraForPersistence(cloned, endpointMode); chatErr == nil {
		cloned = normalized
	}
	if cloned == nil {
		cloned = make(map[string]any)
	}
	if endpointMode == EndpointModeChatCompletions {
		delete(cloned, providerExtraKeyEndpointMode)
	} else {
		cloned[providerExtraKeyEndpointMode] = string(endpointMode)
	}
	if responseFormat == ResponseFormatAuto {
		delete(cloned, providerExtraKeyResponseFormat)
	} else {
		cloned[providerExtraKeyResponseFormat] = string(responseFormat)
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func enabledProviders(items []ProviderConfig) []ProviderConfig {
	enabled := make([]ProviderConfig, 0, len(items))
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		if strings.TrimSpace(item.BaseURL) == "" || strings.TrimSpace(item.Model) == "" {
			continue
		}
		enabled = append(enabled, item)
	}
	sort.SliceStable(enabled, func(i, j int) bool {
		if enabled[i].Priority != enabled[j].Priority {
			return enabled[i].Priority < enabled[j].Priority
		}
		return enabled[i].ID < enabled[j].ID
	})
	return enabled
}

func normalizeSceneConfig(config *SceneConfig) {
	if config == nil {
		return
	}
	if config.Scene == "" {
		config.Scene = SceneSummary
	}
	if !IsValidStrategy(string(config.Strategy)) {
		config.Strategy = StrategyPriorityFailover
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 2
	}
	if len(config.RetryOn) == 0 {
		config.RetryOn = DefaultRetryOn()
	}
	config.RetryOn = normalizeRetryOn(config.RetryOn)
	if config.Breaker.FailureThreshold <= 0 {
		config.Breaker.FailureThreshold = DefaultBreakerConfig().FailureThreshold
	}
	if config.Breaker.CooldownSeconds <= 0 {
		config.Breaker.CooldownSeconds = DefaultBreakerConfig().CooldownSeconds
	}
	for index := range config.Providers {
		config.Providers[index].BaseURL = strings.TrimRight(strings.TrimSpace(config.Providers[index].BaseURL), "/")
		config.Providers[index].Model = strings.TrimSpace(config.Providers[index].Model)
		if config.Providers[index].Adapter == "" {
			config.Providers[index].Adapter = AdapterOpenAICompatible
		}
		config.Providers[index].EndpointMode = NormalizeProviderEndpointMode(string(config.Providers[index].EndpointMode))
		config.Providers[index].ResponseFormat = NormalizeProviderResponseFormat(string(config.Providers[index].ResponseFormat))
		if config.Providers[index].EndpointMode != EndpointModeImagesGenerations {
			config.Providers[index].ResponseFormat = ResponseFormatAuto
		}
		config.Providers[index].Extra = providerExtraForPersistence(
			config.Providers[index].Extra,
			config.Providers[index].EndpointMode,
			config.Providers[index].ResponseFormat,
		)
		if config.Providers[index].Priority <= 0 {
			config.Providers[index].Priority = (index + 1) * 10
		}
		if config.Providers[index].TimeoutSeconds <= 0 {
			config.Providers[index].TimeoutSeconds = 30
		}
	}
	sort.SliceStable(config.Providers, func(i, j int) bool {
		if config.Providers[i].Priority != config.Providers[j].Priority {
			return config.Providers[i].Priority < config.Providers[j].Priority
		}
		return config.Providers[i].ID < config.Providers[j].ID
	})
}

func defaultSceneConfig(scene Scene) SceneConfig {
	config := SceneConfig{
		Scene:       scene,
		Enabled:     false,
		Strategy:    StrategyPriorityFailover,
		MaxAttempts: 2,
		RetryOn:     DefaultRetryOn(),
		Breaker:     DefaultBreakerConfig(),
		Providers:   []ProviderConfig{},
		Source:      "empty",
	}
	if scene == SceneTitle {
		config.RequestOptions.MaxTokens = 64
	}
	return config
}

func sceneAuditSummary(config SceneConfig) string {
	if config.Scene == "" {
		return ""
	}
	return fmt.Sprintf(
		"enabled=%t strategy=%s maxAttempts=%d retryOn=%s breaker=%d/%ds providers=%d requestOptions=%s",
		config.Enabled,
		config.Strategy,
		config.MaxAttempts,
		strings.Join(config.RetryOn, ","),
		config.Breaker.FailureThreshold,
		config.Breaker.CooldownSeconds,
		len(config.Providers),
		requestOptionsSummary(config.RequestOptions),
	)
}

func providerAuditSummary(provider ProviderConfig) string {
	if provider.ID == "" {
		return ""
	}
	apiKey := ""
	if provider.HasAPIKey {
		apiKey = provider.APIKeyMasked
		if apiKey == "" {
			apiKey = "****"
		}
	}
	return fmt.Sprintf(
		"name=%s enabled=%t priority=%d adapter=%s baseURL=%s model=%s timeout=%ds endpoint=%s responseFormat=%s apiKey=%s",
		provider.Name,
		provider.Enabled,
		provider.Priority,
		provider.Adapter,
		provider.BaseURL,
		provider.Model,
		provider.TimeoutSeconds,
		provider.EndpointMode,
		provider.ResponseFormat,
		apiKey,
	)
}

func requestOptionsSummary(options RequestOptions) string {
	parts := make([]string, 0, 3)
	if options.Stream {
		parts = append(parts, "stream=true")
	}
	if options.Temperature != 0 {
		parts = append(parts, fmt.Sprintf("temperature=%g", options.Temperature))
	}
	if options.MaxTokens > 0 {
		parts = append(parts, fmt.Sprintf("maxTokens=%d", options.MaxTokens))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ",")
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func shouldRetry(retryOn []string, errorType string) bool {
	for _, item := range retryOn {
		if strings.TrimSpace(item) == strings.TrimSpace(errorType) {
			return true
		}
	}
	return false
}

func routeErrorType(err error) string {
	if err == nil {
		return ""
	}
	type typed interface {
		AuditErrorType() string
	}
	var typedErr typed
	if errors.As(err, &typedErr) {
		return strings.TrimSpace(typedErr.AuditErrorType())
	}
	return audit.ErrorTypeFromError(err)
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func intPtr(value int) *int {
	return &value
}
