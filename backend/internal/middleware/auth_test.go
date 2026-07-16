package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func TestAuthenticateChecksCurrentTokenVersion(t *testing.T) {
	t.Parallel()

	manager := auth.NewTokenManager("middleware-user-token-secret", 1)
	token, err := manager.Issue(42, 7)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		store   TokenVersionStore
		want    int
		wantHit bool
	}{
		{name: "matching version", store: staticTokenVersionStore{version: 7}, want: http.StatusNoContent, wantHit: true},
		{name: "revoked version", store: staticTokenVersionStore{version: 8}, want: http.StatusUnauthorized},
		{name: "lookup failure", store: staticTokenVersionStore{err: errors.New("database unavailable")}, want: http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hit := false
			handler := Authenticate(manager, test.store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hit = true
				userID, ok := common.CurrentUserID(r.Context())
				if !ok || userID != 42 {
					t.Fatalf("current user=%d ok=%t", userID, ok)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.want || hit != test.wantHit {
				t.Fatalf("status=%d hit=%t body=%s, want status=%d hit=%t", response.Code, hit, response.Body.String(), test.want, test.wantHit)
			}
		})
	}
}

type staticTokenVersionStore struct {
	version int64
	err     error
}

func (s staticTokenVersionStore) TokenVersion(context.Context, int64) (int64, error) {
	return s.version, s.err
}
