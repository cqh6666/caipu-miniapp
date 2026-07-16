package airouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/logging"
)

var (
	markdownImageURLPattern = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)\)`)
	plainURLPattern         = regexp.MustCompile(`https?://[^\s)]+`)
	dataImageURLPattern     = regexp.MustCompile(`data:image/[a-zA-Z0-9.+-]+;base64,[A-Za-z0-9+/=]+`)
)

func classifyRequestError(err error, timeout time.Duration) error {
	if audit.IsTimeoutError(err) {
		message := "upstream request timed out"
		if timeout > 0 {
			message = fmt.Sprintf("%s after %s", message, timeout.Round(time.Second))
		}
		return &typedError{
			errorType:  ErrorTypeTimeout,
			message:    message,
			httpStatus: http.StatusBadGateway,
			cause:      err,
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return &typedError{
			errorType:  ErrorTypeNetwork,
			message:    "network error while calling upstream",
			httpStatus: http.StatusBadGateway,
			cause:      err,
		}
	}

	return &typedError{
		errorType:  ErrorTypeUnknown,
		message:    "request to upstream failed",
		httpStatus: http.StatusBadGateway,
		cause:      err,
	}
}

func classifyHTTPError(status int, message string) error {
	var cause error
	if sanitized := logging.SanitizeText(message); sanitized != "" {
		cause = errors.New(sanitized)
	}

	switch {
	case status == http.StatusTooManyRequests:
		return &typedError{errorType: ErrorTypeRateLimit, message: "upstream rate limited request", httpStatus: http.StatusBadGateway, cause: cause}
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return &typedError{errorType: ErrorTypeAuth, message: "upstream authentication failed", httpStatus: http.StatusBadGateway, cause: cause}
	case status >= 500:
		return &typedError{errorType: ErrorTypeUpstream, message: fmt.Sprintf("upstream service returned status %d", status), httpStatus: http.StatusBadGateway, cause: cause}
	case status >= 400:
		return &typedError{errorType: ErrorTypeBadRequest, message: fmt.Sprintf("upstream rejected request with status %d", status), httpStatus: http.StatusBadGateway, cause: cause}
	default:
		return &typedError{errorType: ErrorTypeUnknown, message: "upstream returned an error response", httpStatus: http.StatusBadGateway, cause: cause}
	}
}

func normalizeValidationError(err error) error {
	if err == nil {
		return nil
	}

	type typed interface {
		AuditErrorType() string
	}
	var typedErr typed
	if errors.As(err, &typedErr) && strings.TrimSpace(typedErr.AuditErrorType()) != "" {
		return err
	}

	var appErr *common.AppError
	if errors.As(err, &appErr) {
		return &typedError{
			errorType:  ErrorTypeBusiness,
			message:    strings.TrimSpace(appErr.Message),
			httpStatus: appErr.HTTPStatus,
			cause:      err,
		}
	}

	return &typedError{
		errorType:  ErrorTypeInvalidResponse,
		message:    truncateText(strings.TrimSpace(err.Error()), 180),
		httpStatus: http.StatusBadGateway,
		cause:      err,
	}
}

func validateSummaryTestContent(content string) error {
	content = trimCodeFenceContent(content)
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("summary response was empty")
	}
	var payload struct {
		Title string `json:"title"`
		Steps []struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Title) == "" {
		return fmt.Errorf("summary title is required")
	}
	if len(payload.Steps) == 0 {
		return fmt.Errorf("summary steps are required")
	}
	for _, step := range payload.Steps {
		if strings.TrimSpace(step.Title) == "" || strings.TrimSpace(step.Detail) == "" {
			return fmt.Errorf("summary steps must contain title and detail")
		}
	}
	return nil
}

func validateTitleTestContent(content string) error {
	content = trimCodeFenceContent(content)
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("title response was empty")
	}
	var payload struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Title) == "" {
		return fmt.Errorf("title is required")
	}
	return nil
}

func validateFlowchartTestContent(content string) error {
	content = trimCodeFenceContent(content)
	if extractImageURL(content) == "" {
		return fmt.Errorf("flowchart response did not contain an image url")
	}
	return nil
}

func trimCodeFenceContent(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}
	start := 0
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
		start = 1
	}
	end := len(lines)
	if end > start && strings.HasPrefix(strings.TrimSpace(lines[end-1]), "```") {
		end--
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func extractImageURL(content string) string {
	if matches := markdownImageURLPattern.FindStringSubmatch(content); len(matches) == 2 {
		if value := normalizeImageReference(matches[1]); value != "" {
			return value
		}
	}
	if dataURL := dataImageURLPattern.FindString(content); dataURL != "" {
		return normalizeImageReference(dataURL)
	}
	for _, candidate := range plainURLPattern.FindAllString(content, -1) {
		if value := normalizeImageReference(candidate); value != "" {
			return value
		}
	}
	return ""
}

func normalizeImageReference(value string) string {
	value = strings.TrimSpace(strings.TrimRight(value, "])}>.,;!\"'"))
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "data:image/"):
		return value
	default:
		return ""
	}
}
