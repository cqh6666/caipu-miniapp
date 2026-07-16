package admin

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const AdminCSRFHeader = "X-CSRF-Token"

type Middleware interface {
	Require(next http.Handler) http.Handler
}

type AuthMiddleware struct {
	tokens *TokenManager
}

func NewAuthMiddleware(tokens *TokenManager) *AuthMiddleware {
	return &AuthMiddleware{tokens: tokens}
}

func (m *AuthMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.tokens == nil {
			common.WriteError(w, common.NewAppError(common.CodeInternalServer, "admin auth is not configured", http.StatusServiceUnavailable))
			return
		}

		token := strings.TrimSpace(readAdminToken(r))
		if token == "" {
			common.WriteError(w, common.NewAppError(common.CodeUnauthorized, "admin login is required", http.StatusUnauthorized))
			return
		}

		subject, err := m.tokens.Parse(token)
		if err != nil {
			common.WriteError(w, err)
			return
		}
		if adminCSRFRequired(r.Method) {
			fetchSite := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")))
			if fetchSite != "" && fetchSite != "same-origin" {
				common.WriteError(w, common.NewAppError(common.CodeForbidden, "cross-site admin request is forbidden", http.StatusForbidden))
				return
			}
			if !m.tokens.ValidateCSRFToken(token, r.Header.Get(AdminCSRFHeader)) {
				common.WriteError(w, common.NewAppError(common.CodeForbidden, "invalid admin CSRF token", http.StatusForbidden))
				return
			}
		}

		next.ServeHTTP(w, r.WithContext(common.WithCurrentAdminSubject(r.Context(), subject)))
	})
}

func adminCSRFRequired(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func readAdminToken(r *http.Request) string {
	if r == nil {
		return ""
	}

	if cookie, err := r.Cookie(AdminCookieName); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}
