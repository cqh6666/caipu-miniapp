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
	AdminUsername              string
	AdminPasswordHash          string
	AdminJWTSecret             string
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
	AIFlowchartBaseURL         string
	AIFlowchartAPIKey          string
	AIFlowchartModel           string
	AIFlowchartTimeoutSeconds  int
	AITitleEnabled             bool
	AITitleBaseURL             string
	AITitleAPIKey              string
	AITitleModel               string
	AITitleStream              bool
	AITitleTemperature         float64
	AITitleMaxTokens           int
	AITitleTimeoutSeconds      int
	LinkparseSidecarEnabled    bool
	LinkparseSidecarBaseURL    string
	LinkparseSidecarTimeoutSec int
	LinkparseSidecarAPIKey     string
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
	InviteShareFontPath        string
	InviteShareFontBoldPath    string
	RecipeAutoParseEnabled     bool
	RecipeAutoParseInterval    int
	RecipeAutoParseBatchSize   int
	RecipeFlowchartEnabled     bool
	RecipeFlowchartAutoEnqueue bool
	RecipeFlowchartInterval    int
	RecipeFlowchartBatchSize   int
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
		AdminUsername:              strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		AdminPasswordHash:          strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_HASH")),
		AdminJWTSecret:             strings.TrimSpace(os.Getenv("ADMIN_JWT_SECRET")),
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
		AIFlowchartBaseURL:         strings.TrimSpace(os.Getenv("AI_FLOWCHART_BASE_URL")),
		AIFlowchartAPIKey:          strings.TrimSpace(os.Getenv("AI_FLOWCHART_API_KEY")),
		AIFlowchartModel:           strings.TrimSpace(os.Getenv("AI_FLOWCHART_MODEL")),
		AIFlowchartTimeoutSeconds:  getInt("AI_FLOWCHART_TIMEOUT_SECONDS", 45),
		AITitleEnabled:             getBool("AI_TITLE_ENABLED", false),
		AITitleBaseURL:             strings.TrimSpace(os.Getenv("AI_TITLE_BASE_URL")),
		AITitleAPIKey:              strings.TrimSpace(os.Getenv("AI_TITLE_API_KEY")),
		AITitleModel:               strings.TrimSpace(os.Getenv("AI_TITLE_MODEL")),
		AITitleStream:              getBool("AI_TITLE_STREAM", false),
		AITitleTemperature:         getFloat("AI_TITLE_TEMPERATURE", 0),
		AITitleMaxTokens:           getInt("AI_TITLE_MAX_TOKENS", 64),
		AITitleTimeoutSeconds:      getInt("AI_TITLE_TIMEOUT_SECONDS", 3),
		LinkparseSidecarEnabled:    getBool("LINKPARSE_SIDECAR_ENABLED", false),
		LinkparseSidecarBaseURL:    strings.TrimSpace(os.Getenv("LINKPARSE_SIDECAR_BASE_URL")),
		LinkparseSidecarTimeoutSec: getInt("LINKPARSE_SIDECAR_TIMEOUT_SECONDS", 150),
		LinkparseSidecarAPIKey:     strings.TrimSpace(os.Getenv("LINKPARSE_SIDECAR_API_KEY")),
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
		InviteShareFontPath:        strings.TrimSpace(os.Getenv("INVITE_SHARE_FONT_PATH")),
		InviteShareFontBoldPath:    strings.TrimSpace(os.Getenv("INVITE_SHARE_FONT_BOLD_PATH")),
		RecipeAutoParseEnabled:     getBool("RECIPE_AUTO_PARSE_ENABLED", true),
		RecipeAutoParseInterval:    getInt("RECIPE_AUTO_PARSE_INTERVAL_SECONDS", 30),
		RecipeAutoParseBatchSize:   getInt("RECIPE_AUTO_PARSE_BATCH_SIZE", 3),
		RecipeFlowchartEnabled:     getBool("RECIPE_FLOWCHART_ENABLED", true),
		RecipeFlowchartAutoEnqueue: getBool("RECIPE_FLOWCHART_AUTO_ENQUEUE_ENABLED", false),
		RecipeFlowchartInterval:    getInt("RECIPE_FLOWCHART_INTERVAL_SECONDS", 5),
		RecipeFlowchartBatchSize:   getInt("RECIPE_FLOWCHART_BATCH_SIZE", 1),
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

	if cfg.AIFlowchartTimeoutSeconds <= 0 {
		return Config{}, errors.New("AI_FLOWCHART_TIMEOUT_SECONDS must be positive")
	}

	if cfg.AITitleTimeoutSeconds <= 0 {
		return Config{}, errors.New("AI_TITLE_TIMEOUT_SECONDS must be positive")
	}

	if cfg.AITitleMaxTokens <= 0 {
		return Config{}, errors.New("AI_TITLE_MAX_TOKENS must be positive")
	}

	if cfg.LinkparseSidecarTimeoutSec <= 0 {
		return Config{}, errors.New("LINKPARSE_SIDECAR_TIMEOUT_SECONDS must be positive")
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

	if cfg.RecipeFlowchartInterval <= 0 {
		return Config{}, errors.New("RECIPE_FLOWCHART_INTERVAL_SECONDS must be positive")
	}

	if cfg.RecipeFlowchartBatchSize <= 0 {
		return Config{}, errors.New("RECIPE_FLOWCHART_BATCH_SIZE must be positive")
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

	if cfg.CredentialsSecret == "" {
		cfg.CredentialsSecret = cfg.JWTSecret
	}

	if cfg.AdminJWTSecret == "" {
		cfg.AdminJWTSecret = cfg.JWTSecret
	}

	if cfg.AIFlowchartBaseURL == "" {
		cfg.AIFlowchartBaseURL = cfg.AIBaseURL
	}
	if cfg.AIFlowchartAPIKey == "" {
		cfg.AIFlowchartAPIKey = cfg.AIAPIKey
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

func getFloat(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseFloat(raw, 64)
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
