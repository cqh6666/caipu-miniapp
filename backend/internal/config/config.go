package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName                    string
	AppEnv                     string
	AppAddr                    string
	AppEnvFile                 string
	LogLevel                   string
	AdminOpenIDs               []string
	AppSettingsAccessMode      string
	AppSettingsAllowedOpenIDs  []string
	CredentialsSecret          string
	JWTSecret                  string
	JWTExpireHours             int
	AIBaseURL                  string
	AIAPIKey                   string
	AIModel                    string
	AITimeoutSeconds           int
	AITitleEnabled             bool
	AITitleModel               string
	AITitleTimeoutSeconds      int
	XHSSidecarEnabled          bool
	XHSSidecarBaseURL          string
	XHSSidecarTimeoutSeconds   int
	XHSSidecarProvider         string
	XHSSidecarAPIKey           string
	WechatAppID                string
	WechatAppSecret            string
	SQLitePath                 string
	SQLiteBusyTimeoutMS        int
	MigrationDir               string
	UploadDir                  string
	UploadPublicBaseURL        string
	UploadMaxImageMB           int64
	InviteDefaultExpireHours   int
	InviteDefaultMaxUses       int
	RecipeAutoParseEnabled     bool
	RecipeAutoParseInterval    int
	RecipeAutoParseBatchSize   int
	RecipeImageMirrorEnabled   bool
	RecipeImageMirrorInterval  int
	RecipeImageMirrorBatchSize int
}

func Load() (Config, error) {
	loadEnvFiles()

	cfg := Config{
		AppName:                    getEnv("APP_NAME", "caipu-miniapp-backend"),
		AppEnv:                     getEnv("APP_ENV", "local"),
		AppAddr:                    getEnv("APP_ADDR", ":8080"),
		AppEnvFile:                 os.Getenv("APP_ENV_FILE"),
		LogLevel:                   getEnv("LOG_LEVEL", "info"),
		AdminOpenIDs:               splitCSV(os.Getenv("APP_ADMIN_OPENIDS")),
		AppSettingsAccessMode:      strings.TrimSpace(strings.ToLower(getEnv("APP_SETTINGS_ACCESS_MODE", "all"))),
		AppSettingsAllowedOpenIDs:  splitCSV(os.Getenv("APP_SETTINGS_ALLOWED_OPENIDS")),
		CredentialsSecret:          strings.TrimSpace(os.Getenv("CREDENTIALS_SECRET")),
		JWTSecret:                  getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpireHours:             getInt("JWT_EXPIRE_HOURS", 720),
		AIBaseURL:                  strings.TrimSpace(getEnv("AI_BASE_URL", "https://api.openai.com/v1")),
		AIAPIKey:                   strings.TrimSpace(os.Getenv("AI_API_KEY")),
		AIModel:                    strings.TrimSpace(os.Getenv("AI_MODEL")),
		AITimeoutSeconds:           getInt("AI_TIMEOUT_SECONDS", 30),
		AITitleEnabled:             getBool("AI_TITLE_ENABLED", false),
		AITitleModel:               strings.TrimSpace(os.Getenv("AI_TITLE_MODEL")),
		AITitleTimeoutSeconds:      getInt("AI_TITLE_TIMEOUT_SECONDS", 3),
		XHSSidecarEnabled:          getBool("XHS_SIDECAR_ENABLED", false),
		XHSSidecarBaseURL:          strings.TrimSpace(os.Getenv("XHS_SIDECAR_BASE_URL")),
		XHSSidecarTimeoutSeconds:   getInt("XHS_SIDECAR_TIMEOUT_SECONDS", 25),
		XHSSidecarProvider:         strings.TrimSpace(strings.ToLower(getEnv("XHS_SIDECAR_PROVIDER", "auto"))),
		XHSSidecarAPIKey:           strings.TrimSpace(os.Getenv("XHS_SIDECAR_API_KEY")),
		WechatAppID:                os.Getenv("WECHAT_APP_ID"),
		WechatAppSecret:            os.Getenv("WECHAT_APP_SECRET"),
		SQLitePath:                 filepath.Clean(getEnv("SQLITE_PATH", "./data/app.db")),
		SQLiteBusyTimeoutMS:        getInt("SQLITE_BUSY_TIMEOUT_MS", 5000),
		MigrationDir:               filepath.Clean(getEnv("MIGRATION_DIR", "./migrations")),
		UploadDir:                  filepath.Clean(getEnv("UPLOAD_DIR", "./data/uploads")),
		UploadPublicBaseURL:        strings.TrimSpace(os.Getenv("UPLOAD_PUBLIC_BASE_URL")),
		UploadMaxImageMB:           int64(getInt("UPLOAD_MAX_IMAGE_MB", 10)),
		InviteDefaultExpireHours:   getInt("INVITE_DEFAULT_EXPIRE_HOURS", 72),
		InviteDefaultMaxUses:       getInt("INVITE_DEFAULT_MAX_USES", 10),
		RecipeAutoParseEnabled:     getBool("RECIPE_AUTO_PARSE_ENABLED", true),
		RecipeAutoParseInterval:    getInt("RECIPE_AUTO_PARSE_INTERVAL_SECONDS", 30),
		RecipeAutoParseBatchSize:   getInt("RECIPE_AUTO_PARSE_BATCH_SIZE", 3),
		RecipeImageMirrorEnabled:   getBool("RECIPE_IMAGE_MIRROR_ENABLED", true),
		RecipeImageMirrorInterval:  getInt("RECIPE_IMAGE_MIRROR_INTERVAL_SECONDS", 180),
		RecipeImageMirrorBatchSize: getInt("RECIPE_IMAGE_MIRROR_BATCH_SIZE", 2),
	}

	if cfg.SQLiteBusyTimeoutMS <= 0 {
		return Config{}, errors.New("SQLITE_BUSY_TIMEOUT_MS must be positive")
	}

	if cfg.UploadMaxImageMB <= 0 {
		return Config{}, errors.New("UPLOAD_MAX_IMAGE_MB must be positive")
	}

	if cfg.AITimeoutSeconds <= 0 {
		return Config{}, errors.New("AI_TIMEOUT_SECONDS must be positive")
	}

	if cfg.AITitleTimeoutSeconds <= 0 {
		return Config{}, errors.New("AI_TITLE_TIMEOUT_SECONDS must be positive")
	}

	if cfg.XHSSidecarTimeoutSeconds <= 0 {
		return Config{}, errors.New("XHS_SIDECAR_TIMEOUT_SECONDS must be positive")
	}

	if cfg.InviteDefaultExpireHours <= 0 {
		return Config{}, errors.New("INVITE_DEFAULT_EXPIRE_HOURS must be positive")
	}

	if cfg.InviteDefaultMaxUses <= 0 {
		return Config{}, errors.New("INVITE_DEFAULT_MAX_USES must be positive")
	}

	if cfg.RecipeAutoParseInterval <= 0 {
		return Config{}, errors.New("RECIPE_AUTO_PARSE_INTERVAL_SECONDS must be positive")
	}

	if cfg.RecipeAutoParseBatchSize <= 0 {
		return Config{}, errors.New("RECIPE_AUTO_PARSE_BATCH_SIZE must be positive")
	}

	if cfg.RecipeImageMirrorInterval <= 0 {
		return Config{}, errors.New("RECIPE_IMAGE_MIRROR_INTERVAL_SECONDS must be positive")
	}

	if cfg.RecipeImageMirrorBatchSize <= 0 {
		return Config{}, errors.New("RECIPE_IMAGE_MIRROR_BATCH_SIZE must be positive")
	}

	switch cfg.AppSettingsAccessMode {
	case "", "all":
		cfg.AppSettingsAccessMode = "all"
	case "admin", "whitelist":
	default:
		return Config{}, errors.New("APP_SETTINGS_ACCESS_MODE must be one of all, admin, whitelist")
	}

	switch cfg.XHSSidecarProvider {
	case "", "auto":
		cfg.XHSSidecarProvider = "auto"
	case "importer", "rednote":
	default:
		return Config{}, errors.New("XHS_SIDECAR_PROVIDER must be one of auto, importer, rednote")
	}

	if cfg.CredentialsSecret == "" {
		cfg.CredentialsSecret = cfg.JWTSecret
	}

	return cfg, nil
}

func loadEnvFiles() {
	envFiles := []string{
		".env",
		"configs/local.env",
	}

	if extra := os.Getenv("APP_ENV_FILE"); extra != "" {
		envFiles = append(envFiles, extra)
	}

	for _, file := range envFiles {
		if _, err := os.Stat(file); err == nil {
			_ = godotenv.Overload(file)
		}
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

func getBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}

	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		items = append(items, value)
	}

	return items
}
