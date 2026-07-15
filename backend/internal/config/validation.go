package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
)

func validateAndFinalize(cfg Config) (Config, error) {
	positiveValues := []struct {
		value int64
		name  string
	}{
		{int64(cfg.SQLiteBusyTimeoutMS), "SQLITE_BUSY_TIMEOUT_MS"},
		{cfg.UploadMaxImageMB, "UPLOAD_MAX_IMAGE_MB"},
		{int64(cfg.AITimeoutSeconds), "AI_TIMEOUT_SECONDS"},
		{int64(cfg.AIFlowchartTimeoutSeconds), "AI_FLOWCHART_TIMEOUT_SECONDS"},
		{int64(cfg.AITitleTimeoutSeconds), "AI_TITLE_TIMEOUT_SECONDS"},
		{int64(cfg.AITitleMaxTokens), "AI_TITLE_MAX_TOKENS"},
		{int64(cfg.AIAlertFailureThreshold), "AI_ALERT_FAILURE_THRESHOLD"},
		{int64(cfg.AIAlertSMTPPort), "AI_ALERT_SMTP_PORT"},
		{int64(cfg.DietAssistantAITimeoutSec), "DIET_ASSISTANT_AI_TIMEOUT_SECONDS"},
		{int64(cfg.LinkparseSidecarTimeoutSec), "LINKPARSE_SIDECAR_TIMEOUT_SECONDS"},
		{int64(cfg.AMapPlacePreviewTimeoutSeconds), "AMAP_PLACE_PREVIEW_TIMEOUT_SECONDS"},
		{int64(cfg.AMapPlacePreviewMaxAttempts), "AMAP_PLACE_PREVIEW_MAX_ATTEMPTS"},
		{int64(cfg.InviteDefaultExpireHours), "INVITE_DEFAULT_EXPIRE_HOURS"},
		{int64(cfg.InviteDefaultMaxUses), "INVITE_DEFAULT_MAX_USES"},
		{int64(cfg.RecipeAutoParseInterval), "RECIPE_AUTO_PARSE_INTERVAL_SECONDS"},
		{int64(cfg.RecipeAutoParseBatchSize), "RECIPE_AUTO_PARSE_BATCH_SIZE"},
		{int64(cfg.RecipeAutoParseMaxAttempts), "RECIPE_AUTO_PARSE_MAX_ATTEMPTS"},
		{int64(cfg.RecipeAutoParseRetryBaseSec), "RECIPE_AUTO_PARSE_RETRY_BASE_SECONDS"},
		{int64(cfg.RecipeAutoParseStaleSec), "RECIPE_AUTO_PARSE_STALE_SECONDS"},
		{int64(cfg.RecipeFlowchartInterval), "RECIPE_FLOWCHART_INTERVAL_SECONDS"},
		{int64(cfg.RecipeFlowchartBatchSize), "RECIPE_FLOWCHART_BATCH_SIZE"},
		{int64(cfg.RecipeImageMirrorInterval), "RECIPE_IMAGE_MIRROR_INTERVAL_SECONDS"},
		{int64(cfg.RecipeImageMirrorBatchSize), "RECIPE_IMAGE_MIRROR_BATCH_SIZE"},
	}
	for _, item := range positiveValues {
		if item.value <= 0 {
			return Config{}, errors.New(item.name + " must be positive")
		}
	}

	if cfg.AMapPlacePreviewQPSDelayMS < 0 {
		return Config{}, errors.New("AMAP_PLACE_PREVIEW_QPS_DELAY_MS must be zero or positive")
	}
	if !isValidDietAssistantThinkingType(cfg.DietAssistantAIThinkingType) {
		return Config{}, errors.New("DIET_ASSISTANT_AI_THINKING_TYPE must be empty, enabled, or disabled")
	}
	if !isValidDietAssistantReasoningEffort(cfg.DietAssistantAIReasoningEffort) {
		return Config{}, errors.New("DIET_ASSISTANT_AI_REASONING_EFFORT must be empty, high, or max")
	}

	switch cfg.AppSettingsAccessMode {
	case "", "all":
		cfg.AppSettingsAccessMode = "all"
	case "admin", "whitelist":
	default:
		return Config{}, errors.New("APP_SETTINGS_ACCESS_MODE must be one of all, admin, whitelist")
	}

	cfg.AppEnv = strings.ToLower(strings.TrimSpace(cfg.AppEnv))
	if cfg.AppEnv == "" {
		cfg.AppEnv = "local"
	}
	if cfg.AppEnv != "local" {
		secrets := []struct {
			name  string
			value string
		}{
			{name: "JWT_SECRET", value: cfg.JWTSecret},
			{name: "ADMIN_JWT_SECRET", value: cfg.AdminJWTSecret},
			{name: "CREDENTIALS_SECRET", value: cfg.CredentialsSecret},
		}
		for _, item := range secrets {
			if isWeakProductionSecret(item.value) {
				return Config{}, fmt.Errorf("%s must be explicitly set to at least 32 non-default characters outside local mode", item.name)
			}
		}
		if cfg.JWTSecret == cfg.AdminJWTSecret || cfg.JWTSecret == cfg.CredentialsSecret || cfg.AdminJWTSecret == cfg.CredentialsSecret {
			return Config{}, errors.New("JWT_SECRET, ADMIN_JWT_SECRET, and CREDENTIALS_SECRET must be independent outside local mode")
		}
	}
	previousKeys, err := credentialcipher.ParsePreviousKeys(cfg.CredentialsPreviousKeys)
	if err != nil {
		return Config{}, err
	}
	if _, err := credentialcipher.New(credentialcipher.Key{
		Version: cfg.CredentialsKeyVersion,
		Secret:  cfg.CredentialsSecret,
	}, previousKeys); err != nil {
		return Config{}, fmt.Errorf("invalid credential keyring: %w", err)
	}
	if cfg.AIFlowchartBaseURL == "" {
		cfg.AIFlowchartBaseURL = cfg.AIBaseURL
	}
	if cfg.AIFlowchartAPIKey == "" {
		cfg.AIFlowchartAPIKey = cfg.AIAPIKey
	}

	return cfg, nil
}

func isWeakProductionSecret(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 32 {
		return true
	}
	switch value {
	case localUserJWTSecret, localAdminJWTSecret, localCredentialsKey, "dev-secret-change-me", "replace-me":
		return true
	default:
		return false
	}
}

func isValidDietAssistantThinkingType(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "enabled", "disabled":
		return true
	default:
		return false
	}
}

func isValidDietAssistantReasoningEffort(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "high", "max":
		return true
	default:
		return false
	}
}
