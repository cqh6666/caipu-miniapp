package addpreview

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var (
	bracketPattern       = regexp.MustCompile(`【([^】]+)】`)
	addressPattern       = regexp.MustCompile(`(?i)【?\s*地址[:：]\s*([^】\n\r]+)`)
	phonePattern         = regexp.MustCompile(`(?i)【?\s*(?:电话|手机|联系电话)[:：]\s*([0-9+\-\s]{6,})`)
	firstURLPattern      = regexp.MustCompile(`https?://[^\s，,。；;）)\]】>]+`)
	poiIDPattern         = regexp.MustCompile(`(?i)(?:poiId|poiid|poi_id)=([0-9]+)`)
	poiIDEncryptPattern  = regexp.MustCompile(`(?i)(?:poiIdEncrypt|poiidEncrypt|poi_id_encrypt)=([^&\s]+)`)
	controlCharsReplacer = strings.NewReplacer("\u0000", "", "\u0008", "", "\u001b", "")
)

func sanitizePreviewText(value string) string {
	value = controlCharsReplacer.Replace(value)
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, value)
	value = strings.TrimSpace(value)

	runes := []rune(value)
	if len(runes) > 2000 {
		return string(runes[:2000])
	}
	return value
}

func parseShareText(text string) ExtractedPlace {
	sourceURL := extractFirstURL(text)
	extracted := ExtractedPlace{
		Name:      extractShareName(text),
		Address:   extractPatternValue(addressPattern, text),
		Phone:     normalizePhone(extractPatternValue(phonePattern, text)),
		SourceURL: sourceURL,
	}

	extracted.POIID, extracted.POIIDEncrypt = extractPOIIDs(text)
	if sourceURL != "" {
		poiID, poiIDEncrypt := extractPOIIDs(sourceURL)
		if extracted.POIID == "" {
			extracted.POIID = poiID
		}
		if extracted.POIIDEncrypt == "" {
			extracted.POIIDEncrypt = poiIDEncrypt
		}
	}

	return extracted
}

func extractShareName(text string) string {
	matches := bracketPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		value := strings.TrimSpace(match[1])
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "地址") || strings.HasPrefix(value, "电话") || strings.Contains(value, "地址：") || strings.Contains(value, "电话：") {
			continue
		}
		return value
	}
	return ""
}

func extractPatternValue(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func extractFirstURL(text string) string {
	value := firstURLPattern.FindString(text)
	return strings.TrimRight(strings.TrimSpace(value), "。；;，,）)]】>")
}

func extractPOIIDs(text string) (string, string) {
	poiID := ""
	if match := poiIDPattern.FindStringSubmatch(text); len(match) >= 2 {
		poiID = strings.TrimSpace(match[1])
	}
	poiIDEncrypt := ""
	if match := poiIDEncryptPattern.FindStringSubmatch(text); len(match) >= 2 {
		if decoded, err := url.QueryUnescape(strings.TrimSpace(match[1])); err == nil {
			poiIDEncrypt = decoded
		} else {
			poiIDEncrypt = strings.TrimSpace(match[1])
		}
	}
	return poiID, poiIDEncrypt
}

func normalizePhone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) || r == '+' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func detectSource(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(text, "美团") || strings.Contains(lower, "meituan") || strings.Contains(lower, "dpurl.cn"):
		return SourceMeituan
	case strings.Contains(text, "大众点评") || strings.Contains(text, "点评") || strings.Contains(lower, "dianping"):
		return SourceDianping
	default:
		return SourceOther
	}
}

func looksLikePlaceShare(text string, extracted ExtractedPlace) bool {
	if extracted.Name != "" && (extracted.Address != "" || extracted.Phone != "" || extracted.SourceURL != "") {
		return true
	}
	source := detectSource(text)
	return source == SourceMeituan || source == SourceDianping
}

func cleanStoreName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer("·", "", "•", "", "（", "(", "）", ")")
	value = replacer.Replace(value)
	if index := strings.Index(value, "("); index > 0 {
		value = strings.TrimSpace(value[:index])
	}
	return strings.Trim(value, " -_")
}

func normalizeMatchText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		" ", "", "\t", "", "\n", "", "\r", "",
		"·", "", "•", "", "-", "", "_", "",
		"（", "", "）", "", "(", "", ")", "",
		"【", "", "】", "", "[", "", "]", "",
		"。", "", "，", "", ",", "", ".", "",
	)
	return replacer.Replace(value)
}
