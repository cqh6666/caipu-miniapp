package kitchen

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/profile"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByUserID(ctx context.Context, userID int64) ([]Summary, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) ListMembers(ctx context.Context, userID, kitchenID int64) ([]Member, error) {
	if err := s.EnsureMember(ctx, userID, kitchenID); err != nil {
		return nil, err
	}

	items, err := s.repo.ListMembers(ctx, kitchenID, userID)
	if err != nil {
		return nil, err
	}

	for index := range items {
		items[index].Nickname = profile.DisplayName(items[index].Nickname, items[index].UserID, "")
	}

	return items, nil
}

func (s *Service) CountMembers(ctx context.Context, kitchenID int64) (int, error) {
	return s.repo.CountMembers(ctx, kitchenID)
}

func (s *Service) EnsureDefaultKitchen(ctx context.Context, userID int64, nickname, openID string) (Summary, error) {
	items, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return Summary{}, err
	}

	if len(items) > 0 {
		return items[0], nil
	}

	return s.repo.CreateWithOwner(ctx, userID, buildAutoKitchenName(nickname, userID, openID), nameSourceAuto)
}

func (s *Service) SyncOwnedAutoKitchenNames(ctx context.Context, userID int64, nickname, openID string) error {
	return s.repo.UpdateOwnedAutoNames(ctx, userID, buildAutoKitchenName(nickname, userID, openID))
}

func (s *Service) CreateKitchen(ctx context.Context, userID int64, name string) (Summary, error) {
	name, err := validateKitchenName(name)
	if err != nil {
		return Summary{}, err
	}

	return s.repo.CreateWithOwner(ctx, userID, name, nameSourceCustom)
}

func (s *Service) UpdateKitchen(ctx context.Context, userID, kitchenID int64, name string) (Summary, error) {
	name, err := validateKitchenName(name)
	if err != nil {
		return Summary{}, err
	}

	if err := s.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Summary{}, err
	}

	if err := s.repo.UpdateName(ctx, kitchenID, name); err != nil {
		return Summary{}, err
	}

	items, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return Summary{}, err
	}

	for _, item := range items {
		if item.ID == kitchenID {
			return item, nil
		}
	}

	return Summary{}, common.ErrInternal.WithErr(fmt.Errorf("updated kitchen %d not found in user list", kitchenID))
}

func (s *Service) LeaveCurrentMember(ctx context.Context, userID, kitchenID int64) (LeaveResult, error) {
	if err := s.repo.Leave(ctx, userID, kitchenID); err != nil {
		return LeaveResult{}, err
	}

	items, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return LeaveResult{}, fmt.Errorf("list kitchens after leave: %w", err)
	}

	var currentKitchenID int64
	if len(items) > 0 {
		currentKitchenID = items[0].ID
	}

	return LeaveResult{
		Kitchens:         items,
		CurrentKitchenID: currentKitchenID,
	}, nil
}

func (s *Service) EnsureMember(ctx context.Context, userID, kitchenID int64) error {
	ok, err := s.repo.HasMembership(ctx, userID, kitchenID)
	if err != nil {
		return err
	}
	if !ok {
		return common.ErrForbidden
	}

	return nil
}

func validateKitchenName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", common.NewAppError(common.CodeBadRequest, "kitchen name is required", http.StatusBadRequest)
	}
	if len([]rune(name)) > 40 {
		return "", common.NewAppError(common.CodeBadRequest, "kitchen name must be 40 characters or fewer", http.StatusBadRequest)
	}

	return name, nil
}
