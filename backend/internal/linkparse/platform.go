package linkparse

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func extractSupportedURL(rawInput string) (string, error) {
	value := strings.TrimSpace(rawInput)
	if value == "" {
		return "", common.NewAppError(common.CodeBadRequest, "url is required", http.StatusBadRequest)
	}

	if match := firstURLPattern.FindString(value); match != "" {
		value = strings.TrimRight(match, "。；;，,）)]】>")
	}

	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "https://" + value
	}

	u, err := url.Parse(value)
	if err != nil || strings.TrimSpace(u.Host) == "" {
		return "", common.NewAppError(common.CodeBadRequest, "invalid url", http.StatusBadRequest)
	}

	return u.String(), nil
}

func SupportsBilibiliURL(rawInput string) bool {
	normalized, err := extractSupportedURL(rawInput)
	if err != nil {
		return false
	}

	u, err := url.Parse(normalized)
	if err != nil {
		return false
	}

	return isResolvableBilibiliHost(u.Host)
}

func SupportsXiaohongshuURL(rawInput string) bool {
	normalized, err := extractSupportedURL(rawInput)
	if err != nil {
		return false
	}

	u, err := url.Parse(normalized)
	if err != nil {
		return false
	}

	return isResolvableXiaohongshuHost(u.Host)
}

func SupportsAutoParseURL(rawInput string) bool {
	return DetectParsePlatform(rawInput) != ""
}

func DetectParsePlatform(rawInput string) string {
	switch {
	case SupportsBilibiliURL(rawInput):
		return "bilibili"
	case SupportsXiaohongshuURL(rawInput):
		return "xiaohongshu"
	default:
		return ""
	}
}

func isResolvableXiaohongshuHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return strings.Contains(host, "xiaohongshu.com") || strings.Contains(host, "xhslink.com")
}
