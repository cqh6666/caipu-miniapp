package admin

import (
	"context"
	"net/http"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestServiceLoginIssuesToken(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret-123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}

	tokens := NewTokenManager("unit-test-secret", time.Hour, "admin")
	service := NewService("admin", string(hash), tokens, false)

	token, err := service.Login(context.Background(), "admin", "secret-123")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	subject, err := tokens.Parse(token)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if subject != "admin" {
		t.Fatalf("subject = %q, want %q", subject, "admin")
	}
}

func TestServiceLoginRejectsWrongPassword(t *testing.T) {
	t.Parallel()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret-123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}

	service := NewService("admin", string(hash), NewTokenManager("unit-test-secret", time.Hour, "admin"), false)
	if _, err := service.Login(context.Background(), "admin", "wrong-password"); err == nil {
		t.Fatal("expected login error")
	}
}

func TestServiceBuildsScopedStrictCookies(t *testing.T) {
	t.Parallel()

	service := NewService(
		"admin",
		"unused",
		NewTokenManager("unit-test-secret", time.Hour, "admin"),
		true,
		"/caipu-api/admin",
	)
	for name, cookie := range map[string]*http.Cookie{
		"session": service.BuildSessionCookie("session-token"),
		"logout":  service.BuildLogoutCookie(),
	} {
		if cookie.Path != "/caipu-api/admin" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
			t.Fatalf("%s cookie is not scoped and strict: %#v", name, cookie)
		}
	}
}
