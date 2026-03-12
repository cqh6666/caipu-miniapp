package profile

import (
	"fmt"
	"hash/fnv"
	"strings"
)

const fallbackPrefix = "厨友"

var fallbackWords = []string{
	"青柠",
	"海盐",
	"南瓜",
	"糯米",
	"山茶",
	"奶油",
	"可可",
	"栗子",
	"芋圆",
	"桂花",
	"松子",
	"薄荷",
	"米酒",
	"芝麻",
	"番茄",
	"豆乳",
}

var placeholderNames = map[string]struct{}{
	"微信用户":    {},
	"wechat user": {},
}

func DisplayName(current string, userID int64, openID string) string {
	current = strings.TrimSpace(current)
	if current != "" && !IsPlaceholderNickname(current) {
		return current
	}

	return FallbackNickname(userID, openID)
}

func FallbackNickname(userID int64, openID string) string {
	seed := seedValue(userID, openID)
	word := fallbackWords[int(seed)%len(fallbackWords)]
	number := int(seed%90) + 10
	return fmt.Sprintf("%s%s%d", fallbackPrefix, word, number)
}

func IsFallbackNickname(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), fallbackPrefix)
}

func IsPlaceholderNickname(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	if IsFallbackNickname(trimmed) {
		return true
	}

	_, ok := placeholderNames[strings.ToLower(trimmed)]
	return ok
}

func seedValue(userID int64, openID string) uint32 {
	if userID > 0 {
		return uint32(userID)
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(strings.TrimSpace(openID)))
	return hasher.Sum32()
}
