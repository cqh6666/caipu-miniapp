package admin

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenManagerRejectsWrongPurposeAndConfiguredSubject(t *testing.T) {
	manager := NewTokenManager("shared-test-secret", time.Hour, "root-admin")
	now := time.Now()

	tests := []Claims{
		{
			TokenUse:         "user_access",
			RegisteredClaims: validAdminRegisteredClaims(now, "root-admin"),
		},
		{
			TokenUse:         adminTokenUse,
			RegisteredClaims: validAdminRegisteredClaims(now, "another-admin"),
		},
	}
	for _, claims := range tests {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		raw, err := token.SignedString([]byte("shared-test-secret"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := manager.Parse(raw); err == nil {
			t.Fatal("expected admin token rejection")
		}
	}
}

func TestTokenManagerBindsCSRFTokenToSessionToken(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("csrf-test-secret", time.Hour, "root-admin")
	sessionToken, err := manager.Issue("root-admin")
	if err != nil {
		t.Fatal(err)
	}
	csrfToken := manager.CSRFToken(sessionToken)
	if csrfToken == "" || !manager.ValidateCSRFToken(sessionToken, csrfToken) {
		t.Fatal("expected valid CSRF token for session")
	}
	if manager.ValidateCSRFToken(sessionToken+"tampered", csrfToken) {
		t.Fatal("CSRF token must be bound to the exact session token")
	}
	if manager.ValidateCSRFToken(sessionToken, csrfToken+"tampered") {
		t.Fatal("tampered CSRF token must be rejected")
	}
}

func validAdminRegisteredClaims(now time.Time, subject string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Issuer:    adminTokenIssuer,
		Subject:   subject,
		Audience:  jwt.ClaimStrings{adminTokenAudience},
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now.Add(-time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
}
