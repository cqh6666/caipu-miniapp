package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"golang.org/x/crypto/bcrypt"
)

const AdminCookieName = "caipu_admin_token"

type AuthService interface {
	Login(ctx context.Context, username, password string) (string, error)
	CurrentSubject(ctx context.Context) (string, error)
	BuildSessionCookie(token string) *http.Cookie
	BuildLogoutCookie() *http.Cookie
}

type Service struct {
	username     string
	passwordHash string
	tokens       *TokenManager
	cookieSecure bool
}

func NewService(username, passwordHash string, tokens *TokenManager, cookieSecure bool) *Service {
	return &Service{
		username:     strings.TrimSpace(username),
		passwordHash: strings.TrimSpace(passwordHash),
		tokens:       tokens,
		cookieSecure: cookieSecure,
	}
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	if strings.TrimSpace(s.username) == "" || strings.TrimSpace(s.passwordHash) == "" || s.tokens == nil {
		return "", common.NewAppError(common.CodeInternalServer, "admin login is not configured", http.StatusServiceUnavailable)
	}

	if strings.TrimSpace(username) != s.username {
		return "", common.NewAppError(common.CodeUnauthorized, "invalid username or password", http.StatusUnauthorized)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(s.passwordHash), []byte(password)); err != nil {
		return "", common.NewAppError(common.CodeUnauthorized, "invalid username or password", http.StatusUnauthorized).WithErr(err)
	}
	return s.tokens.Issue(s.username)
}

func (s *Service) CurrentSubject(ctx context.Context) (string, error) {
	subject, ok := common.CurrentAdminSubject(ctx)
	if !ok || strings.TrimSpace(subject) == "" {
		return "", common.ErrUnauthorized
	}
	return strings.TrimSpace(subject), nil
}

func (s *Service) BuildSessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     AdminCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cookieSecure,
		Expires:  time.Now().Add(24 * time.Hour),
		MaxAge:   int((24 * time.Hour).Seconds()),
	}
}

func (s *Service) BuildLogoutCookie() *http.Cookie {
	return &http.Cookie{
		Name:     AdminCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cookieSecure,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
}
