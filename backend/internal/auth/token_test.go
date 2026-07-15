package auth

import (
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenManagerRejectsWrongTokenPurposeAndSubject(t *testing.T) {
	manager := NewTokenManager("shared-test-secret", 1)
	now := time.Now()

	tests := []struct {
		name   string
		claims Claims
	}{
		{
			name:   "wrong token use",
			claims: Claims{UserID: 42, TokenUse: "admin_access", RegisteredClaims: validUserRegisteredClaims(now, "42")},
		},
		{
			name:   "subject mismatch",
			claims: Claims{UserID: 42, TokenUse: userTokenUse, RegisteredClaims: validUserRegisteredClaims(now, "43")},
		},
		{
			name:   "wrong issuer",
			claims: Claims{UserID: 42, TokenUse: userTokenUse, RegisteredClaims: validUserRegisteredClaims(now, "42")},
		},
	}
	tests[2].claims.Issuer = "caipu-miniapp-admin"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, tt.claims)
			raw, err := token.SignedString([]byte("shared-test-secret"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := manager.Parse(raw); err == nil {
				t.Fatal("expected token rejection")
			}
		})
	}
}

func validUserRegisteredClaims(now time.Time, subject string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Issuer:    userTokenIssuer,
		Subject:   subject,
		Audience:  jwt.ClaimStrings{userTokenAudience},
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now.Add(-time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		ID:        strconv.FormatInt(now.UnixNano(), 10),
	}
}
