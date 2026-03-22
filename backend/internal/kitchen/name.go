package kitchen

import (
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/profile"
)

const (
	fallbackKitchenName = "我的厨房"
	nameSourceAuto      = "auto"
	nameSourceCustom    = "custom"
)

func buildAutoKitchenName(ownerNickname string, ownerUserID int64, ownerOpenID string) string {
	displayName := strings.TrimSpace(profile.DisplayName(ownerNickname, ownerUserID, ownerOpenID))
	if displayName == "" {
		return fallbackKitchenName
	}

	return displayName + "的厨房"
}

func normalizeNameSource(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case nameSourceAuto:
		return nameSourceAuto
	default:
		return nameSourceCustom
	}
}
