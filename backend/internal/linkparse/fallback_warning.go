package linkparse

import (
	"encoding/json"
	"regexp"
	"strings"
)

var requestIDHintPattern = regexp.MustCompile(`\s*\(request id:[^)]+\)`)

func buildAISummaryFallbackWarning(err error) string {
	message := extractAIErrorMessage(err)
	if message == "" {
		return "AI 总结暂时不可用，已回退到规则整理并生成一句话重点。"
	}
	return "AI 总结失败：" + message + "；已回退到规则整理。"
}

func extractAIErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		return ""
	}

	var payload struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal([]byte(message), &payload) == nil && payload.Error != nil {
		message = strings.TrimSpace(payload.Error.Message)
	}

	message = requestIDHintPattern.ReplaceAllString(message, "")
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	return truncateRunes(message, 120)
}
