package auth

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type TokenManager struct {
	secret []byte
	expire time.Duration
}

type Claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, expireHours int) *TokenManager {
	if expireHours <= 0 {
		expireHours = 720
	}

	return &TokenManager{
		secret: []byte(secret),
		expire: time.Duration(expireHours) * time.Hour,
	}
}

func (m *TokenManager) Issue(userID int64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expire)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *TokenManager) Parse(tokenString string) (int64, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return 0, common.NewAppError(common.CodeUnauthorized, "invalid token", http.StatusUnauthorized).WithErr(err)
	}

	if claims.UserID != 0 {
		return claims.UserID, nil
	}

	if claims.Subject == "" {
		return 0, common.NewAppError(common.CodeUnauthorized, "invalid token", http.StatusUnauthorized)
	}

	userID, parseErr := strconv.ParseInt(claims.Subject, 10, 64)
	if parseErr != nil {
		return 0, common.NewAppError(common.CodeUnauthorized, "invalid token", http.StatusUnauthorized).WithErr(parseErr)
	}

	return userID, nil
}
