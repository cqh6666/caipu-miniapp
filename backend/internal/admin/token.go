package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	secret []byte
	expire time.Duration
}

type Claims struct {
	Subject string `json:"sub"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, expire time.Duration) *TokenManager {
	if expire <= 0 {
		expire = 24 * time.Hour
	}
	return &TokenManager{
		secret: []byte(secret),
		expire: expire,
	}
}

func (m *TokenManager) Issue(subject string) (string, error) {
	now := time.Now()
	claims := Claims{
		Subject: subject,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
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
	})
	if err != nil || !token.Valid {
		return "", common.NewAppError(common.CodeUnauthorized, "invalid admin token", http.StatusUnauthorized).WithErr(err)
	}

	if claims.Subject == "" {
		return "", common.NewAppError(common.CodeUnauthorized, "invalid admin token", http.StatusUnauthorized)
	}
	return claims.Subject, nil
}
