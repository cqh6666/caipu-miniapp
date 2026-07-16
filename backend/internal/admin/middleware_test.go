package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthMiddlewareOnlyAcceptsAdminSessionCookie(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("admin-session-test-secret", time.Hour, "root-admin")
	token, err := manager.Issue("root-admin")
	if err != nil {
		t.Fatal(err)
	}

	handler := NewAuthMiddleware(manager).Require(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("bearer token is rejected", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/overview", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
		}
	})

	t.Run("session cookie is accepted", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/overview", nil)
		request.AddCookie(&http.Cookie{Name: AdminCookieName, Value: token})
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
		}
	})
}

func TestAuthMiddlewareRequiresSessionBoundCSRFTokenForWrites(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("admin-csrf-test-secret", time.Hour, "root-admin")
	token, err := manager.Issue("root-admin")
	if err != nil {
		t.Fatal(err)
	}
	csrfToken := manager.CSRFToken(token)
	handler := NewAuthMiddleware(manager).Require(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name      string
		csrfToken string
		fetchSite string
		want      int
	}{
		{name: "missing token", want: http.StatusForbidden},
		{name: "invalid token", csrfToken: "invalid", want: http.StatusForbidden},
		{name: "cross site", csrfToken: csrfToken, fetchSite: "cross-site", want: http.StatusForbidden},
		{name: "same site sibling origin", csrfToken: csrfToken, fetchSite: "same-site", want: http.StatusForbidden},
		{name: "same origin", csrfToken: csrfToken, fetchSite: "same-origin", want: http.StatusNoContent},
		{name: "legacy browser without fetch metadata", csrfToken: csrfToken, want: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/admin/auth/logout", nil)
			request.AddCookie(&http.Cookie{Name: AdminCookieName, Value: token})
			if test.csrfToken != "" {
				request.Header.Set(AdminCSRFHeader, test.csrfToken)
			}
			if test.fetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", test.fetchSite)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.want {
				t.Fatalf("status=%d body=%s, want=%d", response.Code, response.Body.String(), test.want)
			}
		})
	}
}
