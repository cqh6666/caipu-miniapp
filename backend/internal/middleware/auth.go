package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type TokenVersionStore interface {
	TokenVersion(ctx context.Context, userID int64) (int64, error)
}

func Authenticate(tokens *auth.TokenManager, versions TokenVersionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tokens == nil || versions == nil {
				common.WriteError(w, common.NewAppError(common.CodeInternalServer, "user auth is not configured", http.StatusServiceUnavailable))
				return
			}
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

			identity, err := tokens.Parse(strings.TrimSpace(parts[1]))
			if err != nil {
				common.WriteError(w, err)
				return
			}
			currentVersion, err := versions.TokenVersion(r.Context(), identity.UserID)
			if err != nil {
				common.WriteError(w, err)
				return
			}
			if currentVersion != identity.TokenVersion {
				common.WriteError(w, common.NewAppError(common.CodeUnauthorized, "token has been revoked", http.StatusUnauthorized))
				return
			}

			next.ServeHTTP(w, r.WithContext(common.WithCurrentUserID(r.Context(), identity.UserID)))
		})
	}
}
