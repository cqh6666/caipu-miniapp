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
	AppName                  string
	AppEnv                   string
	AppAddr                  string
	AppEnvFile               string
	LogLevel                 string
	JWTSecret                string
	JWTExpireHours           int
	WechatAppID              string
	WechatAppSecret          string
	SQLitePath               string
	SQLiteBusyTimeoutMS      int
	MigrationDir             string
	UploadDir                string
	UploadPublicBaseURL      string
	UploadMaxImageMB         int64
	InviteDefaultExpireHours int
	InviteDefaultMaxUses     int
}

func Load() (Config, error) {
	loadEnvFiles()

	cfg := Config{
		AppName:                  getEnv("APP_NAME", "caipu-miniapp-backend"),
		AppEnv:                   getEnv("APP_ENV", "local"),
		AppAddr:                  getEnv("APP_ADDR", ":8080"),
		AppEnvFile:               os.Getenv("APP_ENV_FILE"),
		LogLevel:                 getEnv("LOG_LEVEL", "info"),
		JWTSecret:                getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpireHours:           getInt("JWT_EXPIRE_HOURS", 720),
		WechatAppID:              os.Getenv("WECHAT_APP_ID"),
		WechatAppSecret:          os.Getenv("WECHAT_APP_SECRET"),
		SQLitePath:               filepath.Clean(getEnv("SQLITE_PATH", "./data/app.db")),
		SQLiteBusyTimeoutMS:      getInt("SQLITE_BUSY_TIMEOUT_MS", 5000),
		MigrationDir:             filepath.Clean(getEnv("MIGRATION_DIR", "./migrations")),
		UploadDir:                filepath.Clean(getEnv("UPLOAD_DIR", "./data/uploads")),
		UploadPublicBaseURL:      strings.TrimSpace(os.Getenv("UPLOAD_PUBLIC_BASE_URL")),
		UploadMaxImageMB:         int64(getInt("UPLOAD_MAX_IMAGE_MB", 10)),
		InviteDefaultExpireHours: getInt("INVITE_DEFAULT_EXPIRE_HOURS", 72),
		InviteDefaultMaxUses:     getInt("INVITE_DEFAULT_MAX_USES", 10),
	}

	if cfg.SQLiteBusyTimeoutMS <= 0 {
		return Config{}, errors.New("SQLITE_BUSY_TIMEOUT_MS must be positive")
	}

	if cfg.UploadMaxImageMB <= 0 {
		return Config{}, errors.New("UPLOAD_MAX_IMAGE_MB must be positive")
	}

	if cfg.InviteDefaultExpireHours <= 0 {
		return Config{}, errors.New("INVITE_DEFAULT_EXPIRE_HOURS must be positive")
	}

	if cfg.InviteDefaultMaxUses <= 0 {
		return Config{}, errors.New("INVITE_DEFAULT_MAX_USES must be positive")
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
