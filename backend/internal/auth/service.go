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
	repo                      *Repository
	kitchen                   *kitchen.Service
	tokenManager              *TokenManager
	wechatClient              wechat.Client
	wechatAppID               string
	adminOpenIDs              map[string]struct{}
	appSettingsAccessMode     string
	appSettingsAllowedOpenIDs map[string]struct{}
}

func NewService(
	repo *Repository,
	kitchenService *kitchen.Service,
	tokenManager *TokenManager,
	wechatClient wechat.Client,
	wechatAppID string,
	adminOpenIDs []string,
	appSettingsAccessMode string,
	appSettingsAllowedOpenIDs []string,
) *Service {
	adminSet := make(map[string]struct{}, len(adminOpenIDs))
	for _, openID := range adminOpenIDs {
		openID = strings.TrimSpace(openID)
		if openID == "" {
			continue
		}
		adminSet[openID] = struct{}{}
	}

	allowedSet := make(map[string]struct{}, len(appSettingsAllowedOpenIDs))
	for _, openID := range appSettingsAllowedOpenIDs {
		openID = strings.TrimSpace(openID)
		if openID == "" {
			continue
		}
		allowedSet[openID] = struct{}{}
	}

	return &Service{
		repo:                      repo,
		kitchen:                   kitchenService,
		tokenManager:              tokenManager,
		wechatClient:              wechatClient,
		wechatAppID:               strings.TrimSpace(wechatAppID),
		adminOpenIDs:              adminSet,
		appSettingsAccessMode:     strings.TrimSpace(strings.ToLower(appSettingsAccessMode)),
		appSettingsAllowedOpenIDs: allowedSet,
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
	if err := s.syncAutoKitchenNames(ctx, user); err != nil {
		return User{}, fmt.Errorf("sync auto kitchen names: %w", err)
	}
	user = s.enrichUser(user)

	return user, nil
}

func (s *Service) EnsureCanManageAppSettings(ctx context.Context, userID int64) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !s.canManageAppSettings(user) {
		return common.ErrForbidden
	}

	return nil
}

func (s *Service) buildSession(ctx context.Context, user User, includeToken bool) (SessionResponse, error) {
	if s.kitchen == nil {
		return SessionResponse{}, common.ErrInternal.WithErr(fmt.Errorf("kitchen service is required"))
	}

	if err := s.ensureCurrentKitchen(ctx, user); err != nil {
		return SessionResponse{}, err
	}

	kitchens, err := s.kitchen.ListByUserID(ctx, user.ID)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("list kitchens: %w", err)
	}
	if len(kitchens) == 0 {
		return SessionResponse{}, common.ErrInternal.WithErr(fmt.Errorf("user %d has no kitchens after bootstrap", user.ID))
	}

	user = s.enrichUser(user)

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

func (s *Service) ensureCurrentKitchen(ctx context.Context, user User) error {
	if s.kitchen == nil {
		return nil
	}

	if _, err := s.kitchen.EnsureDefaultKitchen(ctx, user.ID, user.Nickname, user.OpenID); err != nil {
		return fmt.Errorf("ensure default kitchen: %w", err)
	}
	if err := s.syncAutoKitchenNames(ctx, user); err != nil {
		return fmt.Errorf("sync auto kitchen names: %w", err)
	}

	return nil
}

func (s *Service) syncAutoKitchenNames(ctx context.Context, user User) error {
	if s.kitchen == nil {
		return nil
	}

	return s.kitchen.SyncOwnedAutoKitchenNames(ctx, user.ID, user.Nickname, user.OpenID)
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

func (s *Service) isAdminUser(user User) bool {
	if len(s.adminOpenIDs) == 0 {
		return false
	}

	_, ok := s.adminOpenIDs[strings.TrimSpace(user.OpenID)]
	return ok
}

func (s *Service) canManageAppSettings(user User) bool {
	if s.isAdminUser(user) {
		return true
	}

	switch s.appSettingsAccessMode {
	case "all", "":
		return true
	case "admin":
		return false
	case "whitelist":
		_, ok := s.appSettingsAllowedOpenIDs[strings.TrimSpace(user.OpenID)]
		return ok
	default:
		return false
	}
}

func (s *Service) enrichUser(user User) User {
	user.IsAdmin = s.isAdminUser(user)
	user.CanManageAppSettings = s.canManageAppSettings(user)
	return user
}
