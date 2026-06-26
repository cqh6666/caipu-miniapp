package spacestats

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
)

type Service struct {
	repo    *Repository
	kitchen *kitchen.Service
	now     func() time.Time
}

func NewService(repo *Repository, kitchenService *kitchen.Service) *Service {
	return &Service{
		repo:    repo,
		kitchen: kitchenService,
		now:     time.Now,
	}
}

func (s *Service) GetStats(ctx context.Context, userID, kitchenID int64, rawWindow string) (Stats, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Stats{}, err
	}

	window, windowStart, err := normalizeWindow(rawWindow, s.now())
	if err != nil {
		return Stats{}, err
	}

	now := s.now()
	stats, err := s.repo.GetStats(ctx, kitchenID, windowStart, now.Format("2006-01-02"))
	if err != nil {
		return Stats{}, err
	}

	stats.UpdatedAt = now.Format(time.RFC3339)
	stats.Window = window
	stats.WindowStart = windowStart
	stats.Source = "remote"
	return stats, nil
}

func normalizeWindow(raw string, now time.Time) (string, string, error) {
	window := strings.TrimSpace(strings.ToLower(raw))
	if window == "" {
		window = "30d"
	}

	switch window {
	case "7d":
		return window, now.AddDate(0, 0, -7).Format(time.RFC3339), nil
	case "30d":
		return window, now.AddDate(0, 0, -30).Format(time.RFC3339), nil
	case "90d":
		return window, now.AddDate(0, 0, -90).Format(time.RFC3339), nil
	case "all":
		return window, "", nil
	default:
		return "", "", common.NewAppError(common.CodeBadRequest, "invalid window", http.StatusBadRequest)
	}
}
