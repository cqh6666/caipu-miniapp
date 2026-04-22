package middleware

import (
	"net/http"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type TimeoutOverride struct {
	Method  string
	Prefix  string
	Suffix  string
	Timeout time.Duration
}

func ConditionalTimeout(defaultTimeout time.Duration, overrides []TimeoutOverride) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		defaultHandler := chimiddleware.Timeout(defaultTimeout)(next)
		entries := make([]timeoutOverrideEntry, 0, len(overrides))
		for _, override := range overrides {
			timeout := override.Timeout
			if timeout <= 0 {
				continue
			}
			entries = append(entries, timeoutOverrideEntry{
				method:  strings.ToUpper(strings.TrimSpace(override.Method)),
				prefix:  strings.TrimSpace(override.Prefix),
				suffix:  strings.TrimSpace(override.Suffix),
				handler: chimiddleware.Timeout(timeout)(next),
			})
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method := strings.ToUpper(strings.TrimSpace(r.Method))
			path := strings.TrimSpace(r.URL.Path)
			for _, entry := range entries {
				if entry.matches(method, path) {
					entry.handler.ServeHTTP(w, r)
					return
				}
			}
			defaultHandler.ServeHTTP(w, r)
		})
	}
}

type timeoutOverrideEntry struct {
	method  string
	prefix  string
	suffix  string
	handler http.Handler
}

func (e timeoutOverrideEntry) matches(method, path string) bool {
	if e.method != "" && e.method != method {
		return false
	}
	if e.prefix != "" && !strings.HasPrefix(path, e.prefix) {
		return false
	}
	if e.suffix != "" && !strings.HasSuffix(path, e.suffix) {
		return false
	}
	return true
}
