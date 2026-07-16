package middleware

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type BodyLimitOverride struct {
	Method   string
	Path     string
	MaxBytes int64
}

func RequestBodyLimit(defaultMaxBytes int64, overrides []BodyLimitOverride) func(http.Handler) http.Handler {
	entries := make([]bodyLimitOverrideEntry, 0, len(overrides))
	for _, override := range overrides {
		if override.MaxBytes <= 0 {
			continue
		}
		entries = append(entries, bodyLimitOverrideEntry{
			method:   strings.ToUpper(strings.TrimSpace(override.Method)),
			path:     strings.TrimSpace(override.Path),
			maxBytes: override.MaxBytes,
		})
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := defaultMaxBytes
			method := strings.ToUpper(strings.TrimSpace(r.Method))
			path := strings.TrimSpace(r.URL.Path)
			for _, entry := range entries {
				if entry.matches(method, path) {
					limit = entry.maxBytes
					break
				}
			}
			if limit <= 0 || r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}
			if r.ContentLength > limit {
				common.WriteError(w, common.ErrPayloadTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

type bodyLimitOverrideEntry struct {
	method   string
	path     string
	maxBytes int64
}

func (entry bodyLimitOverrideEntry) matches(method, path string) bool {
	if entry.method != "" && entry.method != method {
		return false
	}
	return entry.path == "" || entry.path == path
}
