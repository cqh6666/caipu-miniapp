package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/wechat"
)

type Service struct {
	repo         *Repository
	kitchen      *kitchen.Service
	tokenManager *TokenManager
	wechatClient wechat.Client
	wechatAppID  string
}

func NewService(
	repo *Repository,
	kitchenService *kitchen.Service,
	tokenManager *TokenManager,
	wechatClient wechat.Client,
	wechatAppID string,
) *Service {
	return &Service{
		repo:         repo,
		kitchen:      kitchenService,
		tokenManager: tokenManager,
		wechatClient: wechatClient,
		wechatAppID:  strings.TrimSpace(wechatAppID),
	}
}

func (s *Service) LoginWithWechatCode(ctx context.Context, code, appID, nickname, avatarURL string) (SessionResponse, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return SessionResponse{}, common.NewAppError(common.CodeBadRequest, "code is required", http.StatusBadRequest)
	}

	appID = strings.TrimSpace(appID)
	if appID != "" && s.wechatAppID != "" && appID != s.wechatAppID {
		return SessionResponse{}, common.NewAppError(common.CodeBadRequest, "mini program appId does not match backend wechat config", http.StatusBadRequest)
	}

	session, err := s.wechatClient.Code2Session(ctx, code)
	if err != nil {
		return SessionResponse{}, err
	}

	user, err := s.repo.FindOrCreateByOpenID(ctx, session.OpenID, nickname, avatarURL)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("find or create user: %w", err)
	}

	return s.buildSession(ctx, user, true)
}

func (s *Service) LoginForDev(ctx context.Context, identity string) (SessionResponse, error) {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		identity = "demo"
	}

	openID := fmt.Sprintf("dev:%s", normalizeIdentity(identity))
	user, err := s.repo.FindOrCreateByOpenID(ctx, openID, identity, "")
	if err != nil {
		return SessionResponse{}, fmt.Errorf("find or create dev user: %w", err)
	}

	return s.buildSession(ctx, user, true)
}

func (s *Service) CurrentSession(ctx context.Context, userID int64) (SessionResponse, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return SessionResponse{}, err
	}

	user, err = s.repo.EnsureProfile(ctx, user, "", "")
	if err != nil {
		return SessionResponse{}, fmt.Errorf("ensure current user profile: %w", err)
	}

	return s.buildSession(ctx, user, false)
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, nickname, avatarURL string) (User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return User{}, err
	}

	user, err = s.repo.EnsureProfile(ctx, user, nickname, avatarURL)
	if err != nil {
		return User{}, fmt.Errorf("update user profile: %w", err)
	}

	return user, nil
}

func (s *Service) buildSession(ctx context.Context, user User, includeToken bool) (SessionResponse, error) {
	if _, err := s.kitchen.EnsureDefaultKitchen(ctx, user.ID); err != nil {
		return SessionResponse{}, fmt.Errorf("ensure default kitchen: %w", err)
	}

	kitchens, err := s.kitchen.ListByUserID(ctx, user.ID)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("list kitchens: %w", err)
	}
	if len(kitchens) == 0 {
		return SessionResponse{}, common.ErrInternal.WithErr(fmt.Errorf("user %d has no kitchens after bootstrap", user.ID))
	}

	response := SessionResponse{
		User:             user,
		CurrentKitchenID: kitchens[0].ID,
		Kitchens:         kitchens,
	}

	if includeToken {
		token, err := s.tokenManager.Issue(user.ID)
		if err != nil {
			return SessionResponse{}, fmt.Errorf("issue token: %w", err)
		}
		response.Token = token
	}

	return response, nil
}

func normalizeIdentity(identity string) string {
	identity = strings.ToLower(strings.TrimSpace(identity))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-", "@", "-", "#", "-")
	identity = replacer.Replace(identity)
	identity = strings.Trim(identity, "-")
	if identity == "" {
		return "demo"
	}
	return identity
}
