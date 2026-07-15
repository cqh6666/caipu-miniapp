package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	localUserJWTSecret  = "local-user-jwt-secret-change-me"
	localAdminJWTSecret = "local-admin-jwt-secret-change-me"
	localCredentialsKey = "local-credentials-secret-change-me"
)

type Config struct {
	AppName                        string
	AppEnv                         string
	AppAddr                        string
	AppEnvFile                     string
	LogLevel                       string
	AdminUsername                  string
	AdminPasswordHash              string
	AdminJWTSecret                 string
	AdminOpenIDs                   []string
	AppSettingsAccessMode          string
	AppSettingsAllowedOpenIDs      []string
	CredentialsSecret              string
	CredentialsKeyVersion          string
	CredentialsPreviousKeys        string
	JWTSecret                      string
	JWTExpireHours                 int
	AIBaseURL                      string
	AIAPIKey                       string
	AIModel                        string
	AITimeoutSeconds               int
	AIFlowchartBaseURL             string
	AIFlowchartAPIKey              string
	AIFlowchartModel               string
	AIFlowchartEndpointMode        string
	AIFlowchartResponseFormat      string
	AIFlowchartTimeoutSeconds      int
	AITitleEnabled                 bool
	AITitleBaseURL                 string
	AITitleAPIKey                  string
	AITitleModel                   string
	AITitleStream                  bool
	AITitleTemperature             float64
	AITitleMaxTokens               int
	AITitleTimeoutSeconds          int
	AIAlertEnabled                 bool
	AIAlertFailureThreshold        int
	AIAlertSMTPHost                string
	AIAlertSMTPPort                int
	AIAlertSMTPUsername            string
	AIAlertSMTPPassword            string
	AIAlertFromEmail               string
	AIAlertToEmails                []string
	DietAssistantAIBaseURL         string
	DietAssistantAIAPIKey          string
	DietAssistantAIModel           string
	DietAssistantAIThinkingType    string
	DietAssistantAIReasoningEffort string
	DietAssistantAITimeoutSec      int
	LinkparseSidecarEnabled        bool
	LinkparseSidecarBaseURL        string
	LinkparseSidecarTimeoutSec     int
	LinkparseSidecarAPIKey         string
	AMapPlacePreviewEnabled        bool
	AMapWebServiceKey              string
	AMapPlacePreviewDefaultCity    string
	AMapPlacePreviewTimeoutSeconds int
	AMapPlacePreviewMaxAttempts    int
	AMapPlacePreviewQPSDelayMS     int
	WechatAppID                    string
	WechatAppSecret                string
	SQLitePath                     string
	SQLiteBusyTimeoutMS            int
	MigrationDir                   string
	UploadDir                      string
	UploadPublicBaseURL            string
	UploadMaxImageMB               int64
	InviteDefaultExpireHours       int
	InviteDefaultMaxUses           int
	InviteShareFontPath            string
	InviteShareFontBoldPath        string
	RecipeAutoParseEnabled         bool
	RecipeAutoParseInterval        int
	RecipeAutoParseBatchSize       int
	RecipeAutoParseMaxAttempts     int
	RecipeAutoParseRetryBaseSec    int
	RecipeAutoParseStaleSec        int
	RecipeFlowchartEnabled         bool
	RecipeFlowchartAutoEnqueue     bool
	RecipeFlowchartInterval        int
	RecipeFlowchartBatchSize       int
	RecipeImageMirrorEnabled       bool
	RecipeImageMirrorInterval      int
	RecipeImageMirrorBatchSize     int
}

func Load() (Config, error) {
	loadEnvFiles()

	cfg := Config{
		AppName:                        getEnv("APP_NAME", "caipu-miniapp-backend"),
		AppEnv:                         getEnv("APP_ENV", "local"),
		AppAddr:                        getEnv("APP_ADDR", ":8080"),
		AppEnvFile:                     os.Getenv("APP_ENV_FILE"),
		LogLevel:                       getEnv("LOG_LEVEL", "info"),
		AdminUsername:                  strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		AdminPasswordHash:              strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_HASH")),
		AdminJWTSecret:                 strings.TrimSpace(getEnv("ADMIN_JWT_SECRET", localAdminJWTSecret)),
		AdminOpenIDs:                   splitCSV(os.Getenv("APP_ADMIN_OPENIDS")),
		AppSettingsAccessMode:          strings.TrimSpace(strings.ToLower(getEnv("APP_SETTINGS_ACCESS_MODE", "admin"))),
		AppSettingsAllowedOpenIDs:      splitCSV(os.Getenv("APP_SETTINGS_ALLOWED_OPENIDS")),
		CredentialsSecret:              strings.TrimSpace(getEnv("CREDENTIALS_SECRET", localCredentialsKey)),
		CredentialsKeyVersion:          strings.TrimSpace(getEnv("CREDENTIALS_KEY_VERSION", "local-v1")),
		CredentialsPreviousKeys:        strings.TrimSpace(os.Getenv("CREDENTIALS_PREVIOUS_KEYS")),
		JWTSecret:                      getEnv("JWT_SECRET", localUserJWTSecret),
		JWTExpireHours:                 getInt("JWT_EXPIRE_HOURS", 720),
		AIBaseURL:                      strings.TrimSpace(getEnv("AI_BASE_URL", "https://api.openai.com/v1")),
		AIAPIKey:                       strings.TrimSpace(os.Getenv("AI_API_KEY")),
		AIModel:                        strings.TrimSpace(os.Getenv("AI_MODEL")),
		AITimeoutSeconds:               getInt("AI_TIMEOUT_SECONDS", 30),
		AIFlowchartBaseURL:             strings.TrimSpace(os.Getenv("AI_FLOWCHART_BASE_URL")),
		AIFlowchartAPIKey:              strings.TrimSpace(os.Getenv("AI_FLOWCHART_API_KEY")),
		AIFlowchartModel:               strings.TrimSpace(os.Getenv("AI_FLOWCHART_MODEL")),
		AIFlowchartEndpointMode:        strings.TrimSpace(os.Getenv("AI_FLOWCHART_ENDPOINT_MODE")),
		AIFlowchartResponseFormat:      strings.TrimSpace(os.Getenv("AI_FLOWCHART_RESPONSE_FORMAT")),
		AIFlowchartTimeoutSeconds:      getInt("AI_FLOWCHART_TIMEOUT_SECONDS", 45),
		AITitleEnabled:                 getBool("AI_TITLE_ENABLED", false),
		AITitleBaseURL:                 strings.TrimSpace(os.Getenv("AI_TITLE_BASE_URL")),
		AITitleAPIKey:                  strings.TrimSpace(os.Getenv("AI_TITLE_API_KEY")),
		AITitleModel:                   strings.TrimSpace(os.Getenv("AI_TITLE_MODEL")),
		AITitleStream:                  getBool("AI_TITLE_STREAM", false),
		AITitleTemperature:             getFloat("AI_TITLE_TEMPERATURE", 0),
		AITitleMaxTokens:               getInt("AI_TITLE_MAX_TOKENS", 64),
		AITitleTimeoutSeconds:          getInt("AI_TITLE_TIMEOUT_SECONDS", 3),
		AIAlertEnabled:                 getBool("AI_ALERT_ENABLED", false),
		AIAlertFailureThreshold:        getInt("AI_ALERT_FAILURE_THRESHOLD", 3),
		AIAlertSMTPHost:                strings.TrimSpace(getEnv("AI_ALERT_SMTP_HOST", "smtp.qq.com")),
		AIAlertSMTPPort:                getInt("AI_ALERT_SMTP_PORT", 587),
		AIAlertSMTPUsername:            strings.TrimSpace(os.Getenv("AI_ALERT_SMTP_USERNAME")),
		AIAlertSMTPPassword:            strings.TrimSpace(os.Getenv("AI_ALERT_SMTP_PASSWORD")),
		AIAlertFromEmail:               strings.TrimSpace(os.Getenv("AI_ALERT_FROM_EMAIL")),
		AIAlertToEmails:                splitCSV(os.Getenv("AI_ALERT_TO_EMAILS")),
		DietAssistantAIBaseURL:         strings.TrimSpace(getEnv("DIET_ASSISTANT_AI_BASE_URL", "https://api.longcat.chat/openai/v1")),
		DietAssistantAIAPIKey:          strings.TrimSpace(os.Getenv("DIET_ASSISTANT_AI_API_KEY")),
		DietAssistantAIModel:           strings.TrimSpace(getEnv("DIET_ASSISTANT_AI_MODEL", "LongCat-2.0-Preview")),
		DietAssistantAIThinkingType:    strings.TrimSpace(strings.ToLower(os.Getenv("DIET_ASSISTANT_AI_THINKING_TYPE"))),
		DietAssistantAIReasoningEffort: strings.TrimSpace(strings.ToLower(os.Getenv("DIET_ASSISTANT_AI_REASONING_EFFORT"))),
		DietAssistantAITimeoutSec:      getInt("DIET_ASSISTANT_AI_TIMEOUT_SECONDS", 90),
		LinkparseSidecarEnabled:        getBool("LINKPARSE_SIDECAR_ENABLED", false),
		LinkparseSidecarBaseURL:        strings.TrimSpace(os.Getenv("LINKPARSE_SIDECAR_BASE_URL")),
		LinkparseSidecarTimeoutSec:     getInt("LINKPARSE_SIDECAR_TIMEOUT_SECONDS", 150),
		LinkparseSidecarAPIKey:         strings.TrimSpace(os.Getenv("LINKPARSE_SIDECAR_API_KEY")),
		AMapPlacePreviewEnabled:        getBool("AMAP_PLACE_PREVIEW_ENABLED", false),
		AMapWebServiceKey:              strings.TrimSpace(os.Getenv("AMAP_WEB_SERVICE_KEY")),
		AMapPlacePreviewDefaultCity:    strings.TrimSpace(getEnv("AMAP_PLACE_PREVIEW_DEFAULT_CITY", "佛山")),
		AMapPlacePreviewTimeoutSeconds: getInt("AMAP_PLACE_PREVIEW_TIMEOUT_SECONDS", 8),
		AMapPlacePreviewMaxAttempts:    getInt("AMAP_PLACE_PREVIEW_MAX_ATTEMPTS", 4),
		AMapPlacePreviewQPSDelayMS:     getInt("AMAP_PLACE_PREVIEW_QPS_DELAY_MS", 400),
		WechatAppID:                    os.Getenv("WECHAT_APP_ID"),
		WechatAppSecret:                os.Getenv("WECHAT_APP_SECRET"),
		SQLitePath:                     filepath.Clean(getEnv("SQLITE_PATH", "./data/app.db")),
		SQLiteBusyTimeoutMS:            getInt("SQLITE_BUSY_TIMEOUT_MS", 5000),
		MigrationDir:                   filepath.Clean(getEnv("MIGRATION_DIR", "./migrations")),
		UploadDir:                      filepath.Clean(getEnv("UPLOAD_DIR", "./data/uploads")),
		UploadPublicBaseURL:            strings.TrimSpace(os.Getenv("UPLOAD_PUBLIC_BASE_URL")),
		UploadMaxImageMB:               int64(getInt("UPLOAD_MAX_IMAGE_MB", 10)),
		InviteDefaultExpireHours:       getInt("INVITE_DEFAULT_EXPIRE_HOURS", 72),
		InviteDefaultMaxUses:           getInt("INVITE_DEFAULT_MAX_USES", 10),
		InviteShareFontPath:            strings.TrimSpace(os.Getenv("INVITE_SHARE_FONT_PATH")),
		InviteShareFontBoldPath:        strings.TrimSpace(os.Getenv("INVITE_SHARE_FONT_BOLD_PATH")),
		RecipeAutoParseEnabled:         getBool("RECIPE_AUTO_PARSE_ENABLED", true),
		RecipeAutoParseInterval:        getInt("RECIPE_AUTO_PARSE_INTERVAL_SECONDS", 30),
		RecipeAutoParseBatchSize:       getInt("RECIPE_AUTO_PARSE_BATCH_SIZE", 3),
		RecipeAutoParseMaxAttempts:     getInt("RECIPE_AUTO_PARSE_MAX_ATTEMPTS", 3),
		RecipeAutoParseRetryBaseSec:    getInt("RECIPE_AUTO_PARSE_RETRY_BASE_SECONDS", 60),
		RecipeAutoParseStaleSec:        getInt("RECIPE_AUTO_PARSE_STALE_SECONDS", 600),
		RecipeFlowchartEnabled:         getBool("RECIPE_FLOWCHART_ENABLED", true),
		RecipeFlowchartAutoEnqueue:     getBool("RECIPE_FLOWCHART_AUTO_ENQUEUE_ENABLED", false),
		RecipeFlowchartInterval:        getInt("RECIPE_FLOWCHART_INTERVAL_SECONDS", 5),
		RecipeFlowchartBatchSize:       getInt("RECIPE_FLOWCHART_BATCH_SIZE", 1),
		RecipeImageMirrorEnabled:       getBool("RECIPE_IMAGE_MIRROR_ENABLED", true),
		RecipeImageMirrorInterval:      getInt("RECIPE_IMAGE_MIRROR_INTERVAL_SECONDS", 180),
		RecipeImageMirrorBatchSize:     getInt("RECIPE_IMAGE_MIRROR_BATCH_SIZE", 2),
	}

	return validateAndFinalize(cfg)
}
