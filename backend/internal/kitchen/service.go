package kitchen

import (
	"context"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const defaultKitchenName = "我们的厨房"

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

	return s.repo.ListMembers(ctx, kitchenID, userID)
}

func (s *Service) EnsureDefaultKitchen(ctx context.Context, userID int64) (Summary, error) {
	items, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return Summary{}, err
	}

	if len(items) > 0 {
		return items[0], nil
	}

	return s.repo.CreateWithOwner(ctx, userID, defaultKitchenName)
}

func (s *Service) CreateKitchen(ctx context.Context, userID int64, name string) (Summary, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Summary{}, common.NewAppError(common.CodeBadRequest, "kitchen name is required", http.StatusBadRequest)
	}
	if len([]rune(name)) > 40 {
		return Summary{}, common.NewAppError(common.CodeBadRequest, "kitchen name must be 40 characters or fewer", http.StatusBadRequest)
	}

	return s.repo.CreateWithOwner(ctx, userID, name)
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
