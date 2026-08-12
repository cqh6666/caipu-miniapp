package recipe

import (
	"errors"
	"regexp"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

var (
	flowchartMarkdownImagePattern = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)\)`)
	flowchartPlainURLPattern      = regexp.MustCompile(`https?://[^\s)]+`)
	flowchartDataImageURLPattern  = regexp.MustCompile(`data:image/[a-zA-Z0-9.+-]+;base64,[A-Za-z0-9+/=]+`)
)

func flowchartErrorCause(err error) string {
	if err == nil {
		return ""
	}

	var appErr *common.AppError
	if errors.As(err, &appErr) {
		parts := make([]string, 0, 2)
		message := strings.TrimSpace(appErr.Message)
		if message != "" {
			parts = append(parts, message)
		}
		if appErr.Err != nil {
			cause := deepestError(appErr.Err)
			if cause != "" && (message == "" || !strings.Contains(message, cause)) {
				parts = append(parts, cause)
			}
		}
		return strings.Join(parts, ": ")
	}

	return deepestError(err)
}

func deepestError(err error) string {
	if err == nil {
		return ""
	}

	current := err
	for {
		next := errors.Unwrap(current)
		if next == nil {
			break
		}
		current = next
	}
	return strings.TrimSpace(current.Error())
}

func extractFlowchartImageURL(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	if matches := flowchartMarkdownImagePattern.FindStringSubmatch(content); len(matches) == 2 {
		if value := normalizeFlowchartImageReference(matches[1]); value != "" {
			return value
		}
	}
	if dataURL := flowchartDataImageURLPattern.FindString(content); dataURL != "" {
		return normalizeFlowchartImageReference(dataURL)
	}

	for _, candidate := range flowchartPlainURLPattern.FindAllString(content, -1) {
		if value := normalizeFlowchartImageReference(candidate); value != "" {
			return value
		}
	}
	return ""
}

func normalizeFlowchartImageReference(value string) string {
	value = strings.TrimSpace(strings.TrimRight(value, "])}>.,;!\"'"))
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "data:image/"):
		return value
	default:
		return ""
	}
}
