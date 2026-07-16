package config

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	configSourceProcessEnv      = "process_env"
	configSourceExplicitEnvFile = "explicit_env_file"
	configSourceLocalDotenv     = "local_dotenv"
)

func loadEnvFiles() (string, error) {
	sources := []string{configSourceProcessEnv}
	if explicitFile := strings.TrimSpace(os.Getenv("APP_ENV_FILE")); explicitFile != "" {
		if err := loadDotenvFile(explicitFile, "APP_ENV_FILE", false); err != nil {
			return "", err
		}
		return strings.Join(append(sources, configSourceExplicitEnvFile), "+"), nil
	}

	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "local") {
		return strings.Join(sources, "+"), nil
	}

	loaded := false
	for _, file := range []string{"configs/local.env", ".env"} {
		fileLoaded, err := loadOptionalLocalDotenv(file)
		if err != nil {
			return "", err
		}
		loaded = loaded || fileLoaded
	}
	if loaded {
		sources = append(sources, configSourceLocalDotenv)
	}
	return strings.Join(sources, "+"), nil
}

func loadOptionalLocalDotenv(file string) (bool, error) {
	info, err := os.Stat(file)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, errors.New("local dotenv is not readable")
	}
	if err := validateDotenvPermissions(info, "local dotenv"); err != nil {
		return false, err
	}
	if err := loadDotenvValues(file, "local dotenv"); err != nil {
		return false, errors.New("local dotenv is invalid")
	}
	return true, nil
}

func loadDotenvFile(file, label string, optional bool) error {
	info, err := os.Stat(file)
	if optional && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s is not readable", label)
	}
	if err := validateDotenvPermissions(info, label); err != nil {
		return err
	}
	if err := loadDotenvValues(file, label); err != nil {
		return fmt.Errorf("%s is invalid", label)
	}
	return nil
}

func loadDotenvValues(file, label string) error {
	values, err := godotenv.Read(file)
	if err != nil {
		return err
	}
	for key, value := range values {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set value from %s: %w", label, err)
		}
	}
	return nil
}

func validateDotenvPermissions(info os.FileInfo, label string) error {
	if info == nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s must be a regular file", label)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%s permissions must be 0600 or stricter", label)
	}
	return nil
}

type typedEnvParser struct {
	errors []error
}

func (parser *typedEnvParser) Int(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		parser.addError(key, "integer")
		return fallback
	}
	return value
}

func (parser *typedEnvParser) Float(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		parser.addError(key, "finite number")
		return fallback
	}
	return value
}

func (parser *typedEnvParser) Bool(key string, fallback bool) bool {
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
		parser.addError(key, "boolean")
		return fallback
	}
}

func (parser *typedEnvParser) addError(key, expected string) {
	parser.errors = append(parser.errors, fmt.Errorf("%s must be a valid %s", key, expected))
}

func (parser *typedEnvParser) Err() error {
	if parser == nil || len(parser.errors) == 0 {
		return nil
	}
	return errors.Join(parser.errors...)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
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
