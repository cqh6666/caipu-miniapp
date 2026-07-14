package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadDefaultsAndFallbacks(t *testing.T) {
	cleanConfigEnvironment(t)
	t.Chdir(t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}

	if cfg.AppName != "caipu-miniapp-backend" || cfg.AppEnv != "local" || cfg.AppAddr != ":8080" {
		t.Fatalf("unexpected app defaults: %#v", cfg)
	}
	if cfg.SQLitePath != filepath.Clean("./data/app.db") || cfg.SQLiteBusyTimeoutMS != 5000 {
		t.Fatalf("unexpected sqlite defaults: path=%q timeout=%d", cfg.SQLitePath, cfg.SQLiteBusyTimeoutMS)
	}
	if cfg.CredentialsSecret != cfg.JWTSecret || cfg.AdminJWTSecret != cfg.JWTSecret {
		t.Fatal("credentials and admin JWT secrets should fall back to JWT_SECRET")
	}
	if cfg.AIFlowchartBaseURL != cfg.AIBaseURL || cfg.AIFlowchartAPIKey != cfg.AIAPIKey {
		t.Fatal("flowchart credentials should fall back to summary AI credentials")
	}
	if cfg.AppSettingsAccessMode != "all" || !cfg.RecipeAutoParseEnabled || !cfg.RecipeFlowchartEnabled {
		t.Fatalf("unexpected feature defaults: access=%q autoParse=%t flowchart=%t", cfg.AppSettingsAccessMode, cfg.RecipeAutoParseEnabled, cfg.RecipeFlowchartEnabled)
	}
}

func TestLoadEnvironmentOverrides(t *testing.T) {
	cleanConfigEnvironment(t)
	t.Chdir(t.TempDir())

	t.Setenv("APP_SETTINGS_ACCESS_MODE", " WHITELIST ")
	t.Setenv("APP_ADMIN_OPENIDS", " alice, ,bob ")
	t.Setenv("AI_TITLE_ENABLED", "yes")
	t.Setenv("AI_TITLE_STREAM", "off")
	t.Setenv("AI_TITLE_TEMPERATURE", "0.35")
	t.Setenv("AI_TITLE_MAX_TOKENS", "96")
	t.Setenv("AI_BASE_URL", " https://summary.example.com/v1 ")
	t.Setenv("AI_API_KEY", " summary-secret ")
	t.Setenv("AI_FLOWCHART_ENDPOINT_MODE", "images_generations")
	t.Setenv("DIET_ASSISTANT_AI_THINKING_TYPE", " ENABLED ")
	t.Setenv("DIET_ASSISTANT_AI_REASONING_EFFORT", " MAX ")
	t.Setenv("RECIPE_IMAGE_MIRROR_ENABLED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load overrides: %v", err)
	}

	if cfg.AppSettingsAccessMode != "whitelist" || !reflect.DeepEqual(cfg.AdminOpenIDs, []string{"alice", "bob"}) {
		t.Fatalf("access overrides mismatch: mode=%q openids=%#v", cfg.AppSettingsAccessMode, cfg.AdminOpenIDs)
	}
	if !cfg.AITitleEnabled || cfg.AITitleStream || cfg.AITitleTemperature != 0.35 || cfg.AITitleMaxTokens != 96 {
		t.Fatalf("title overrides mismatch: %#v", cfg)
	}
	if cfg.AIFlowchartBaseURL != "https://summary.example.com/v1" || cfg.AIFlowchartAPIKey != "summary-secret" {
		t.Fatalf("flowchart fallback mismatch: baseURL=%q apiKey=%q", cfg.AIFlowchartBaseURL, cfg.AIFlowchartAPIKey)
	}
	if cfg.DietAssistantAIThinkingType != "enabled" || cfg.DietAssistantAIReasoningEffort != "max" || cfg.RecipeImageMirrorEnabled {
		t.Fatalf("normalized overrides mismatch: thinking=%q effort=%q mirror=%t", cfg.DietAssistantAIThinkingType, cfg.DietAssistantAIReasoningEffort, cfg.RecipeImageMirrorEnabled)
	}
}

func TestLoadEnvFilePrecedence(t *testing.T) {
	cleanConfigEnvironment(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll("configs", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".env", []byte("APP_NAME=dot-env\nAI_MODEL=dot-model\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("configs/local.env", []byte("APP_NAME=local-env\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(dir, "override.env")
	if err := os.WriteFile(extra, []byte("APP_NAME=extra-env\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_ENV_FILE", extra)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load env files: %v", err)
	}
	if cfg.AppName != "extra-env" || cfg.AIModel != "dot-model" || cfg.AppEnvFile != extra {
		t.Fatalf("unexpected env precedence: appName=%q aiModel=%q envFile=%q", cfg.AppName, cfg.AIModel, cfg.AppEnvFile)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		message string
	}{
		{name: "non-positive timeout", key: "SQLITE_BUSY_TIMEOUT_MS", value: "0", message: "SQLITE_BUSY_TIMEOUT_MS must be positive"},
		{name: "negative qps delay", key: "AMAP_PLACE_PREVIEW_QPS_DELAY_MS", value: "-1", message: "AMAP_PLACE_PREVIEW_QPS_DELAY_MS must be zero or positive"},
		{name: "thinking type", key: "DIET_ASSISTANT_AI_THINKING_TYPE", value: "auto", message: "DIET_ASSISTANT_AI_THINKING_TYPE"},
		{name: "access mode", key: "APP_SETTINGS_ACCESS_MODE", value: "owner", message: "APP_SETTINGS_ACCESS_MODE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanConfigEnvironment(t)
			t.Chdir(t.TempDir())
			t.Setenv(tt.key, tt.value)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("expected error containing %q, got %v", tt.message, err)
			}
		})
	}
}

func cleanConfigEnvironment(t *testing.T) {
	t.Helper()
	keys := []string{
		"APP_NAME", "APP_ENV", "APP_ADDR", "APP_ENV_FILE", "LOG_LEVEL",
		"ADMIN_USERNAME", "ADMIN_PASSWORD_HASH", "ADMIN_JWT_SECRET", "APP_ADMIN_OPENIDS",
		"APP_SETTINGS_ACCESS_MODE", "APP_SETTINGS_ALLOWED_OPENIDS", "CREDENTIALS_SECRET",
		"JWT_SECRET", "JWT_EXPIRE_HOURS", "AI_BASE_URL", "AI_API_KEY", "AI_MODEL", "AI_TIMEOUT_SECONDS",
		"AI_FLOWCHART_BASE_URL", "AI_FLOWCHART_API_KEY", "AI_FLOWCHART_MODEL", "AI_FLOWCHART_ENDPOINT_MODE",
		"AI_FLOWCHART_RESPONSE_FORMAT", "AI_FLOWCHART_TIMEOUT_SECONDS", "AI_TITLE_ENABLED", "AI_TITLE_BASE_URL",
		"AI_TITLE_API_KEY", "AI_TITLE_MODEL", "AI_TITLE_STREAM", "AI_TITLE_TEMPERATURE", "AI_TITLE_MAX_TOKENS",
		"AI_TITLE_TIMEOUT_SECONDS", "AI_ALERT_ENABLED", "AI_ALERT_FAILURE_THRESHOLD", "AI_ALERT_SMTP_HOST",
		"AI_ALERT_SMTP_PORT", "AI_ALERT_SMTP_USERNAME", "AI_ALERT_SMTP_PASSWORD", "AI_ALERT_FROM_EMAIL",
		"AI_ALERT_TO_EMAILS", "DIET_ASSISTANT_AI_BASE_URL", "DIET_ASSISTANT_AI_API_KEY", "DIET_ASSISTANT_AI_MODEL",
		"DIET_ASSISTANT_AI_THINKING_TYPE", "DIET_ASSISTANT_AI_REASONING_EFFORT", "DIET_ASSISTANT_AI_TIMEOUT_SECONDS",
		"LINKPARSE_SIDECAR_ENABLED", "LINKPARSE_SIDECAR_BASE_URL", "LINKPARSE_SIDECAR_TIMEOUT_SECONDS",
		"LINKPARSE_SIDECAR_API_KEY", "AMAP_PLACE_PREVIEW_ENABLED", "AMAP_WEB_SERVICE_KEY",
		"AMAP_PLACE_PREVIEW_DEFAULT_CITY", "AMAP_PLACE_PREVIEW_TIMEOUT_SECONDS", "AMAP_PLACE_PREVIEW_MAX_ATTEMPTS",
		"AMAP_PLACE_PREVIEW_QPS_DELAY_MS", "WECHAT_APP_ID", "WECHAT_APP_SECRET", "SQLITE_PATH",
		"SQLITE_BUSY_TIMEOUT_MS", "MIGRATION_DIR", "UPLOAD_DIR", "UPLOAD_PUBLIC_BASE_URL", "UPLOAD_MAX_IMAGE_MB",
		"INVITE_DEFAULT_EXPIRE_HOURS", "INVITE_DEFAULT_MAX_USES", "INVITE_SHARE_FONT_PATH", "INVITE_SHARE_FONT_BOLD_PATH",
		"RECIPE_AUTO_PARSE_ENABLED", "RECIPE_AUTO_PARSE_INTERVAL_SECONDS", "RECIPE_AUTO_PARSE_BATCH_SIZE",
		"RECIPE_AUTO_PARSE_MAX_ATTEMPTS", "RECIPE_AUTO_PARSE_RETRY_BASE_SECONDS", "RECIPE_AUTO_PARSE_STALE_SECONDS",
		"RECIPE_FLOWCHART_ENABLED", "RECIPE_FLOWCHART_AUTO_ENQUEUE_ENABLED", "RECIPE_FLOWCHART_INTERVAL_SECONDS",
		"RECIPE_FLOWCHART_BATCH_SIZE", "RECIPE_IMAGE_MIRROR_ENABLED", "RECIPE_IMAGE_MIRROR_INTERVAL_SECONDS",
		"RECIPE_IMAGE_MIRROR_BATCH_SIZE",
	}
	for _, key := range keys {
		t.Setenv(key, "")
	}
}
