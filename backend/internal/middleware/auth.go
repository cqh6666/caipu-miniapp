package middleware

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func Authenticate(tokens *auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if header == "" {
				common.WriteError(w, common.NewAppError(common.CodeUnauthorized, "authorization header is required", http.StatusUnauthorized))
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				common.WriteError(w, common.NewAppError(common.CodeUnauthorized, "authorization header must use bearer token", http.StatusUnauthorized))
				return
			}

			userID, err := tokens.Parse(strings.TrimSpace(parts[1]))
			if err != nil {
				common.WriteError(w, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(common.WithCurrentUserID(r.Context(), userID)))
		})
	}
}
