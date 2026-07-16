package dietassistant

import (
	"errors"
	"net/http"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const (
	maxDietAssistantJSONResponseBytes  int64 = 4 << 20
	maxDietAssistantStreamEventBytes         = 256 << 10
	maxDietAssistantVisibleBytes             = 256 << 10
	maxDietAssistantToolBlockBytes           = 64 << 10
	maxDietAssistantToolArgumentsBytes       = 64 << 10
)

var (
	errStreamEventTooLarge = errors.New("diet assistant stream event exceeds size limit")
	errVisibleTextTooLarge = errors.New("diet assistant visible content exceeds size limit")
	errToolPayloadTooLarge = errors.New("diet assistant tool payload exceeds size limit")
)

func dietAssistantLimitError(message string, cause error) error {
	return common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway).WithErr(cause)
}
