package admin

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

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

		next.ServeHTTP(w, r.WithContext(common.WithCurrentAdminSubject(r.Context(), subject)))
	})
}

func readAdminToken(r *http.Request) string {
	if r == nil {
		return ""
	}

	if cookie, err := r.Cookie(AdminCookieName); err == nil {
		return strings.TrimSpace(cookie.Value)
	}

	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
