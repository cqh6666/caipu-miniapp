package recipe

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func (s *Service) UpdateStatus(ctx context.Context, userID int64, recipeID string, status string) (Recipe, error) {
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return Recipe{}, err
	}

	status = strings.TrimSpace(status)
	if !isAllowedStatus(status) {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	now := time.Now().Format(time.RFC3339)
	if err := s.repo.UpdateStatus(ctx, recipeID, current.KitchenID, status, userID, now); errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	} else if err != nil {
		return Recipe{}, err
	}

	current.Status = status
	current.DoneAt = resolveRecipeStatusDoneAt(current.DoneAt, status, now)
	current.UpdatedBy = userID
	return current, nil
}

func resolveRecipeStatusDoneAt(currentDoneAt string, status string, touchedAt string) string {
	if status != "done" {
		return ""
	}
	if strings.TrimSpace(currentDoneAt) != "" {
		return currentDoneAt
	}
	return touchedAt
}

func (s *Service) UpdatePinned(ctx context.Context, userID int64, recipeID string, pinned bool) (Recipe, error) {
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return Recipe{}, err
	}

	currentPinned := strings.TrimSpace(current.PinnedAt) != ""
	if currentPinned == pinned {
		return current, nil
	}

	now := time.Now().Format(time.RFC3339)
	if err := s.repo.UpdatePinned(ctx, recipeID, current.KitchenID, pinned, userID, now); errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	} else if err != nil {
		return Recipe{}, err
	}

	if pinned {
		current.PinnedAt = now
	} else {
		current.PinnedAt = ""
	}
	current.UpdatedBy = userID
	return current, nil
}

func (s *Service) Delete(ctx context.Context, userID int64, recipeID string) error {
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return err
	}

	if err := s.repo.SoftDelete(ctx, recipeID, current.KitchenID, userID, time.Now().Format(time.RFC3339)); errors.Is(err, sql.ErrNoRows) {
		return common.ErrNotFound
	} else if err != nil {
		return err
	}

	return nil
}
