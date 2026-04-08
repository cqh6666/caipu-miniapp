package admin

import (
	"context"
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

	tokens := NewTokenManager("unit-test-secret", time.Hour)
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

	service := NewService("admin", string(hash), NewTokenManager("unit-test-secret", time.Hour), false)
	if _, err := service.Login(context.Background(), "admin", "wrong-password"); err == nil {
		t.Fatal("expected login error")
	}
}
