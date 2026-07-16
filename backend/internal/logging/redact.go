package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

const (
	redactedValue       = "[REDACTED]"
	redactedPayload     = "[structured payload redacted]"
	maxSafeLogTextRunes = 1024
)

var (
	bearerPattern              = regexp.MustCompile(`(?i)\bBearer\s+[^\s,;]+`)
	jwtPattern                 = regexp.MustCompile(`\b[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`)
	urlPattern                 = regexp.MustCompile(`https?://[^\s"'<>]+`)
	sensitiveAssignmentPattern = regexp.MustCompile(
		`(?i)(["']?(?:authorization|cookie|set-cookie|api[_-]?key|token|secret|password|sessdata|credential)["']?\s*[:=]\s*)(["'][^"']*["']|[^,;\s}\]]+)`,
	)
	sensitivePayloadPattern = regexp.MustCompile(`(?i)["']?(?:messages|prompt|input|request[_-]?body|response[_-]?body)["']?\s*:`)
)

type RedactingHandler struct {
	next slog.Handler
}

func NewRedactingHandler(next slog.Handler) slog.Handler {
	return &RedactingHandler{next: next}
}

func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *RedactingHandler) Handle(ctx context.Context, record slog.Record) error {
	sanitized := slog.NewRecord(record.Time, record.Level, SanitizeText(record.Message), record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		sanitized.AddAttrs(sanitizeAttr(attr))
		return true
	})
	return h.next.Handle(ctx, sanitized)
}

func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	sanitized := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		sanitized = append(sanitized, sanitizeAttr(attr))
	}
	return &RedactingHandler{next: h.next.WithAttrs(sanitized)}
}

func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{next: h.next.WithGroup(SanitizeText(name))}
}

func sanitizeAttr(attr slog.Attr) slog.Attr {
	attr.Value = attr.Value.Resolve()
	if isSensitiveKey(attr.Key) {
		return slog.String(attr.Key, redactedValue)
	}
	if attr.Value.Kind() == slog.KindGroup {
		group := attr.Value.Group()
		sanitized := make([]slog.Attr, 0, len(group))
		for _, child := range group {
			sanitized = append(sanitized, sanitizeAttr(child))
		}
		return slog.Group(attr.Key, attrsToAny(sanitized)...)
	}
	if attr.Value.Kind() == slog.KindString {
		return slog.String(attr.Key, SanitizeText(attr.Value.String()))
	}
	if attr.Value.Kind() == slog.KindAny {
		switch value := attr.Value.Any().(type) {
		case error:
			return slog.String(attr.Key, SafeErrorSummary(value))
		case string:
			return slog.String(attr.Key, SanitizeText(value))
		case []string:
			items := make([]string, 0, len(value))
			for _, item := range value {
				items = append(items, SanitizeText(item))
			}
			return slog.Any(attr.Key, items)
		case []byte:
			return slog.String(attr.Key, redactedPayload)
		}
		value := reflect.ValueOf(attr.Value.Any())
		if value.IsValid() {
			switch value.Kind() {
			case reflect.Map, reflect.Struct, reflect.Slice, reflect.Array, reflect.Pointer:
				return slog.String(attr.Key, redactedPayload)
			}
		}
	}
	return attr
}

func attrsToAny(attrs []slog.Attr) []any {
	values := make([]any, len(attrs))
	for index := range attrs {
		values[index] = attrs[index]
	}
	return values
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.TrimSpace(key)))
	for _, marker := range []string{"password", "secret", "token", "cookie", "authorization", "sessdata", "apikey", "credential"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	switch normalized {
	case "body", "payload", "prompt", "messages", "input", "requestbody", "responsebody":
		return true
	default:
		return false
	}
}

func SanitizeText(value string) string {
	value = strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value))
	if value == "" {
		return ""
	}
	if location := sensitivePayloadPattern.FindStringIndex(value); location != nil {
		prefix := strings.TrimSpace(value[:location[0]])
		if brace := strings.LastIndexAny(prefix, "{["); brace >= 0 {
			prefix = strings.TrimSpace(prefix[:brace])
		}
		if prefix == "" {
			value = redactedPayload
		} else {
			value = prefix + " " + redactedPayload
		}
	}
	value = urlPattern.ReplaceAllStringFunc(value, sanitizeURL)
	value = bearerPattern.ReplaceAllString(value, "Bearer "+redactedValue)
	value = jwtPattern.ReplaceAllString(value, redactedValue)
	value = sensitiveAssignmentPattern.ReplaceAllString(value, "${1}"+redactedValue)
	return truncateRunes(value, maxSafeLogTextRunes)
}

func sanitizeURL(raw string) string {
	trailing := ""
	for len(raw) > 0 && strings.ContainsRune(".,);]}", rune(raw[len(raw)-1])) {
		trailing = raw[len(raw)-1:] + trailing
		raw = raw[:len(raw)-1]
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return "[URL REDACTED]" + trailing
	}
	parsed.User = nil
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	return parsed.String() + trailing
}

func SafeErrorSummary(err error) string {
	if err == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	for current, depth := err, 0; current != nil && depth < 8; current, depth = errors.Unwrap(current), depth+1 {
		message := SanitizeText(current.Error())
		if message == "" || (len(parts) > 0 && parts[len(parts)-1] == message) {
			continue
		}
		parts = append(parts, message)
	}
	return truncateRunes(strings.Join(parts, " <- "), maxSafeLogTextRunes)
}

func ErrorTypeChain(err error) []string {
	if err == nil {
		return nil
	}
	result := make([]string, 0, 4)
	for current, depth := err, 0; current != nil && depth < 8; current, depth = errors.Unwrap(current), depth+1 {
		result = append(result, fmt.Sprintf("%T", current))
	}
	return result
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
