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

const (
	userTokenIssuer   = "caipu-miniapp-auth"
	userTokenAudience = "caipu-miniapp-api"
	userTokenUse      = "user_access"
)

type Claims struct {
	UserID       int64  `json:"uid"`
	TokenVersion int64  `json:"ver"`
	TokenUse     string `json:"token_use"`
	jwt.RegisteredClaims
}

type TokenIdentity struct {
	UserID       int64
	TokenVersion int64
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

func (m *TokenManager) Issue(userID, tokenVersion int64) (string, error) {
	if userID <= 0 || tokenVersion <= 0 {
		return "", fmt.Errorf("user id and token version must be positive")
	}
	now := time.Now()
	tokenID, err := common.NewPrefixedID("usr")
	if err != nil {
		return "", fmt.Errorf("generate user token id: %w", err)
	}
	claims := Claims{
		UserID:       userID,
		TokenVersion: tokenVersion,
		TokenUse:     userTokenUse,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    userTokenIssuer,
			Subject:   strconv.FormatInt(userID, 10),
			Audience:  jwt.ClaimStrings{userTokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expire)),
			ID:        tokenID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *TokenManager) Parse(tokenString string) (TokenIdentity, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.secret, nil
	}, jwt.WithIssuer(userTokenIssuer), jwt.WithAudience(userTokenAudience), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return TokenIdentity{}, common.NewAppError(common.CodeUnauthorized, "invalid token", http.StatusUnauthorized).WithErr(err)
	}

	if claims.TokenUse != userTokenUse || claims.UserID <= 0 || claims.TokenVersion <= 0 || claims.Subject == "" || claims.ID == "" {
		return TokenIdentity{}, common.NewAppError(common.CodeUnauthorized, "invalid token", http.StatusUnauthorized)
	}

	userID, parseErr := strconv.ParseInt(claims.Subject, 10, 64)
	if parseErr != nil || userID != claims.UserID {
		return TokenIdentity{}, common.NewAppError(common.CodeUnauthorized, "invalid token", http.StatusUnauthorized).WithErr(parseErr)
	}

	return TokenIdentity{UserID: userID, TokenVersion: claims.TokenVersion}, nil
}
