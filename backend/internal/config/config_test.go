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
	if cfg.AdminCookiePath != "/api/admin" {
		t.Fatalf("admin cookie path=%q", cfg.AdminCookiePath)
	}
	if cfg.HealthBackendBaseURL != "http://127.0.0.1:8080" || cfg.HealthBackendServiceName != "caipu-backend" {
		t.Fatalf("unexpected health defaults: baseURL=%q service=%q", cfg.HealthBackendBaseURL, cfg.HealthBackendServiceName)
	}
	if cfg.SQLitePath != filepath.Clean("./data/app.db") || cfg.SQLiteBusyTimeoutMS != 5000 {
		t.Fatalf("unexpected sqlite defaults: path=%q timeout=%d", cfg.SQLitePath, cfg.SQLiteBusyTimeoutMS)
	}
	if cfg.CredentialsSecret == cfg.JWTSecret || cfg.AdminJWTSecret == cfg.JWTSecret || cfg.CredentialsSecret == cfg.AdminJWTSecret {
		t.Fatal("local credential, user JWT, and admin JWT secrets must remain independent")
	}
	if cfg.AIFlowchartBaseURL != cfg.AIBaseURL || cfg.AIFlowchartAPIKey != cfg.AIAPIKey {
		t.Fatal("flowchart credentials should fall back to summary AI credentials")
	}
	if cfg.AppSettingsAccessMode != "admin" || !cfg.RecipeAutoParseEnabled || !cfg.RecipeFlowchartEnabled {
		t.Fatalf("unexpected feature defaults: access=%q autoParse=%t flowchart=%t", cfg.AppSettingsAccessMode, cfg.RecipeAutoParseEnabled, cfg.RecipeFlowchartEnabled)
	}
}

func TestLoadRejectsMissingWeakOrSharedProductionSecrets(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T)
		message string
	}{
		{
			name:    "local defaults are forbidden",
			setup:   func(t *testing.T) { t.Setenv("APP_ENV", "production") },
			message: "JWT_SECRET",
		},
		{
			name: "shared secrets are forbidden",
			setup: func(t *testing.T) {
				t.Setenv("APP_ENV", "production")
				t.Setenv("JWT_SECRET", strings.Repeat("u", 40))
				t.Setenv("ADMIN_JWT_SECRET", strings.Repeat("u", 40))
				t.Setenv("CREDENTIALS_SECRET", strings.Repeat("c", 40))
			},
			message: "must be independent",
		},
		{
			name: "all settings access is forbidden",
			setup: func(t *testing.T) {
				setValidProductionSecrets(t)
				t.Setenv("APP_SETTINGS_ACCESS_MODE", "all")
			},
			message: "only allowed in local mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanConfigEnvironment(t)
			t.Chdir(t.TempDir())
			tt.setup(t)
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("expected error containing %q, got %v", tt.message, err)
			}
		})
	}
}

func TestLoadEnvironmentOverrides(t *testing.T) {
	cleanConfigEnvironment(t)
	t.Chdir(t.TempDir())

	t.Setenv("APP_SETTINGS_ACCESS_MODE", " WHITELIST ")
	t.Setenv("ADMIN_COOKIE_PATH", "/caipu-api/admin")
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
	t.Setenv("UPLOAD_PUBLIC_BASE_URL", "http://localhost:8080/uploads/")
	t.Setenv("HEALTH_BACKEND_BASE_URL", "http://127.0.0.1:9080/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load overrides: %v", err)
	}

	if cfg.AppSettingsAccessMode != "whitelist" || !reflect.DeepEqual(cfg.AdminOpenIDs, []string{"alice", "bob"}) {
		t.Fatalf("access overrides mismatch: mode=%q openids=%#v", cfg.AppSettingsAccessMode, cfg.AdminOpenIDs)
	}
	if cfg.AdminCookiePath != "/caipu-api/admin" {
		t.Fatalf("admin cookie path=%q", cfg.AdminCookiePath)
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
	if cfg.UploadPublicBaseURL != "http://localhost:8080/uploads" {
		t.Fatalf("normalized upload base URL=%q", cfg.UploadPublicBaseURL)
	}
	if cfg.HealthBackendBaseURL != "http://127.0.0.1:9080" {
		t.Fatalf("normalized health base URL=%q", cfg.HealthBackendBaseURL)
	}
}

func TestLoadValidatesUploadPublicBaseURL(t *testing.T) {
	tests := []struct {
		name        string
		appEnv      string
		baseURL     string
		wantError   string
		wantBaseURL string
	}{
		{name: "local empty fallback", appEnv: "local"},
		{name: "local HTTP", appEnv: "local", baseURL: "http://localhost:8080/uploads/", wantBaseURL: "http://localhost:8080/uploads"},
		{name: "production HTTPS", appEnv: "production", baseURL: "https://static.example.com/uploads/", wantBaseURL: "https://static.example.com/uploads"},
		{name: "production missing", appEnv: "production", wantError: "required outside local mode"},
		{name: "production HTTP", appEnv: "production", baseURL: "http://static.example.com/uploads", wantError: "must use HTTPS"},
		{name: "relative URL", appEnv: "local", baseURL: "/uploads", wantError: "absolute HTTP(S) URL"},
		{name: "userinfo", appEnv: "local", baseURL: "https://user:password@static.example.com/uploads", wantError: "without credentials"},
		{name: "query", appEnv: "local", baseURL: "https://static.example.com/uploads?token=secret", wantError: "without credentials"},
		{name: "fragment", appEnv: "local", baseURL: "https://static.example.com/uploads#asset", wantError: "without credentials"},
		{name: "unsupported scheme", appEnv: "local", baseURL: "ftp://static.example.com/uploads", wantError: "HTTP or HTTPS"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanConfigEnvironment(t)
			t.Chdir(t.TempDir())
			if test.appEnv == "production" {
				setValidProductionSecrets(t)
			} else {
				t.Setenv("APP_ENV", test.appEnv)
			}
			t.Setenv("UPLOAD_PUBLIC_BASE_URL", test.baseURL)

			cfg, err := Load()
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("error=%v, want message containing %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.UploadPublicBaseURL != test.wantBaseURL {
				t.Fatalf("base URL=%q, want=%q", cfg.UploadPublicBaseURL, test.wantBaseURL)
			}
		})
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
	if err := os.WriteFile(extra, []byte("APP_NAME=extra-env\nAI_MODEL=extra-model\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_NAME", "process-env")
	t.Setenv("APP_ENV_FILE", extra)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load env files: %v", err)
	}
	if cfg.AppName != "process-env" || cfg.AIModel != "extra-model" || cfg.AppEnvFile != extra ||
		cfg.ConfigSourceSummary != "process_env+explicit_env_file" {
		t.Fatalf("unexpected env precedence: appName=%q aiModel=%q envFile=%q", cfg.AppName, cfg.AIModel, cfg.AppEnvFile)
	}
}

func TestLoadOnlyReadsDefaultDotenvInExplicitLocalMode(t *testing.T) {
	t.Run("explicit local loads defaults", func(t *testing.T) {
		cleanConfigEnvironment(t)
		t.Chdir(t.TempDir())
		if err := os.MkdirAll("configs", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(".env", []byte("APP_NAME=dot-env\nAI_MODEL=dot-model\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("configs/local.env", []byte("APP_NAME=local-env\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("APP_ENV", "local")

		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.AppName != "local-env" || cfg.AIModel != "dot-model" || cfg.ConfigSourceSummary != "process_env+local_dotenv" {
			t.Fatalf("local dotenv config=%#v", cfg)
		}
	})

	t.Run("production ignores local defaults", func(t *testing.T) {
		cleanConfigEnvironment(t)
		t.Chdir(t.TempDir())
		if err := os.MkdirAll("configs", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("configs/local.env", []byte("APP_NAME=must-not-load\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		setValidProductionSecrets(t)

		cfg, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.AppName != "caipu-miniapp-backend" || cfg.ConfigSourceSummary != "process_env" {
			t.Fatalf("production loaded local dotenv: %#v", cfg)
		}
	})
}

func TestLoadRejectsExplicitEnvFileErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string) string
	}{
		{
			name: "missing",
			setup: func(_ *testing.T, dir string) string {
				return filepath.Join(dir, "missing.env")
			},
		},
		{
			name: "invalid",
			setup: func(t *testing.T, dir string) string {
				path := filepath.Join(dir, "invalid.env")
				if err := os.WriteFile(path, []byte("BROKEN='unterminated\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				return path
			},
		},
		{
			name: "unsafe permissions",
			setup: func(t *testing.T, dir string) string {
				path := filepath.Join(dir, "unsafe.env")
				if err := os.WriteFile(path, []byte("APP_ENV=local\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanConfigEnvironment(t)
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("APP_ENV_FILE", test.setup(t, dir))
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "APP_ENV_FILE") {
				t.Fatalf("error=%v, want sanitized APP_ENV_FILE error", err)
			}
		})
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
		{name: "invalid integer syntax", key: "AI_TIMEOUT_SECONDS", value: "1O", message: "AI_TIMEOUT_SECONDS"},
		{name: "invalid bool syntax", key: "AI_TITLE_ENABLED", value: "flase", message: "AI_TITLE_ENABLED"},
		{name: "non-finite float", key: "AI_TITLE_TEMPERATURE", value: "NaN", message: "AI_TITLE_TEMPERATURE"},
		{name: "broad admin cookie path", key: "ADMIN_COOKIE_PATH", value: "/", message: "ADMIN_COOKIE_PATH"},
		{name: "unnormalized admin cookie path", key: "ADMIN_COOKIE_PATH", value: "/api/../admin", message: "ADMIN_COOKIE_PATH"},
		{name: "health base URL path", key: "HEALTH_BACKEND_BASE_URL", value: "http://127.0.0.1:8080/base", message: "must not contain a path"},
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

func TestLoadAggregatesTypedErrorsWithoutEchoingValues(t *testing.T) {
	cleanConfigEnvironment(t)
	t.Chdir(t.TempDir())
	t.Setenv("AI_TIMEOUT_SECONDS", "1O-sensitive")
	t.Setenv("AI_TITLE_ENABLED", "flase-sensitive")

	_, err := Load()
	if err == nil {
		t.Fatal("expected typed configuration errors")
	}
	message := err.Error()
	for _, key := range []string{"AI_TIMEOUT_SECONDS", "AI_TITLE_ENABLED"} {
		if !strings.Contains(message, key) {
			t.Fatalf("error %q does not contain %s", message, key)
		}
	}
	for _, value := range []string{"1O-sensitive", "flase-sensitive"} {
		if strings.Contains(message, value) {
			t.Fatalf("error leaked raw value %q: %s", value, message)
		}
	}
}

func setValidProductionSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", strings.Repeat("u", 40))
	t.Setenv("ADMIN_JWT_SECRET", strings.Repeat("a", 40))
	t.Setenv("CREDENTIALS_SECRET", strings.Repeat("c", 40))
	t.Setenv("UPLOAD_PUBLIC_BASE_URL", "https://static.example.com/uploads")
}

func cleanConfigEnvironment(t *testing.T) {
	t.Helper()
	keys := []string{
		"APP_NAME", "APP_ENV", "APP_ADDR", "APP_ENV_FILE", "LOG_LEVEL",
		"HEALTH_NGINX_SERVICE_NAME", "HEALTH_BACKEND_SERVICE_NAME", "HEALTH_SIDECAR_SERVICE_NAME", "HEALTH_BACKEND_BASE_URL",
		"ADMIN_USERNAME", "ADMIN_PASSWORD_HASH", "ADMIN_JWT_SECRET", "ADMIN_COOKIE_PATH", "APP_ADMIN_OPENIDS",
		"APP_SETTINGS_ACCESS_MODE", "APP_SETTINGS_ALLOWED_OPENIDS", "CREDENTIALS_SECRET",
		"CREDENTIALS_KEY_VERSION", "CREDENTIALS_PREVIOUS_KEYS",
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
