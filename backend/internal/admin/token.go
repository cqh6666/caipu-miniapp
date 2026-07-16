package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secret          []byte
	expire          time.Duration
	expectedSubject string
}

const (
	adminTokenIssuer   = "caipu-miniapp-admin"
	adminTokenAudience = "caipu-miniapp-admin-api"
	adminTokenUse      = "admin_access"
)

type Claims struct {
	TokenUse string `json:"token_use"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, expire time.Duration, expectedSubject ...string) *TokenManager {
	if expire <= 0 {
		expire = 24 * time.Hour
	}
	subject := ""
	if len(expectedSubject) > 0 {
		subject = strings.TrimSpace(expectedSubject[0])
	}
	return &TokenManager{
		secret:          []byte(secret),
		expire:          expire,
		expectedSubject: subject,
	}
}

func (m *TokenManager) Issue(subject string) (string, error) {
	now := time.Now()
	claims := Claims{
		TokenUse: adminTokenUse,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    adminTokenIssuer,
			Subject:   strings.TrimSpace(subject),
			Audience:  jwt.ClaimStrings{adminTokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expire)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *TokenManager) Parse(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.secret, nil
	}, jwt.WithIssuer(adminTokenIssuer), jwt.WithAudience(adminTokenAudience), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return "", common.NewAppError(common.CodeUnauthorized, "invalid admin token", http.StatusUnauthorized).WithErr(err)
	}

	if claims.TokenUse != adminTokenUse || claims.Subject == "" || m.expectedSubject == "" || claims.Subject != m.expectedSubject {
		return "", common.NewAppError(common.CodeUnauthorized, "invalid admin token", http.StatusUnauthorized)
	}
	return claims.Subject, nil
}

func (m *TokenManager) CSRFToken(tokenString string) string {
	if m == nil || len(m.secret) == 0 || strings.TrimSpace(tokenString) == "" {
		return ""
	}
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte("admin-csrf\x00"))
	_, _ = mac.Write([]byte(tokenString))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (m *TokenManager) ValidateCSRFToken(tokenString, csrfToken string) bool {
	expected := m.CSRFToken(tokenString)
	provided := strings.TrimSpace(csrfToken)
	return expected != "" && provided != "" && hmac.Equal([]byte(expected), []byte(provided))
}
