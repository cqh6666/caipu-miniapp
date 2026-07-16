package linkparse

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/logging"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upstream"
)

const (
	maxLinkparseAIResponseBytes int64 = 2 << 20
	maxSidecarResponseBytes     int64 = 4 << 20
	maxBilibiliResponseBytes    int64 = 4 << 20
)

func decodeBoundedUpstreamJSON(reader io.Reader, maxBytes int64, label string, dst any) error {
	err := upstream.DecodeJSON(reader, maxBytes, dst)
	if err == nil {
		return nil
	}
	message := "invalid " + strings.TrimSpace(label) + " response"
	if upstream.IsResponseTooLarge(err) {
		message = strings.TrimSpace(label) + " response exceeded size limit"
	}
	return common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway).WithErr(err)
}

func sanitizedUpstreamError(code int, message string, status int, raw string) error {
	appErr := common.NewAppError(code, message, status)
	sanitized := logging.SanitizeText(raw)
	if sanitized == "" {
		return appErr
	}
	return appErr.WithErr(errors.New(sanitized))
}
