package place

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
)

const (
	StatusWant    = "want"
	StatusVisited = "visited"

	TypeFood       = "food"
	TypeAttraction = "attraction"
	TypeOther      = "other"

	SourceManual   = "manual"
	SourceDianping = "dianping"
	SourceMeituan  = "meituan"
	SourceOther    = "other"
)

var (
	allowedStatuses = map[string]struct{}{
		StatusWant:    {},
		StatusVisited: {},
	}
	allowedTypes = map[string]struct{}{
		TypeFood:       {},
		TypeAttraction: {},
		TypeOther:      {},
	}
	allowedSources = map[string]struct{}{
		SourceManual:   {},
		SourceDianping: {},
		SourceMeituan:  {},
		SourceOther:    {},
	}
)

type Service struct {
	repo    *Repository
	kitchen *kitchen.Service
}

func NewService(repo *Repository, kitchenService *kitchen.Service) *Service {
	return &Service{repo: repo, kitchen: kitchenService}
}

func (s *Service) ListByKitchenID(ctx context.Context, userID, kitchenID int64, filter ListFilter) ([]Place, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return nil, err
	}

	filter.Status = strings.TrimSpace(filter.Status)
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	if filter.Status != "" && !isAllowedStatus(filter.Status) {
		return nil, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	return s.repo.ListByKitchenID(ctx, kitchenID, filter)
}

func (s *Service) Create(ctx context.Context, userID, kitchenID int64, req placeRequest) (Place, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Place{}, err
	}

	item, err := normalizePlaceInput(req)
	if err != nil {
		return Place{}, err
	}

	placeID, err := common.NewPrefixedID("pla")
	if err != nil {
		return Place{}, fmt.Errorf("generate place id: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	item.ID = placeID
	item.KitchenID = kitchenID
	item.CreatedBy = userID
	item.UpdatedBy = userID
	item.CreatedAt = now
	item.UpdatedAt = now
	item.VisitedAt = resolveVisitedAt("", item.Status, now)

	return s.repo.Create(ctx, item)
}

func (s *Service) GetByID(ctx context.Context, userID int64, placeID string) (Place, error) {
	item, err := s.repo.FindByID(ctx, placeID)
	if errors.Is(err, sql.ErrNoRows) {
		return Place{}, common.ErrNotFound
	}
	if err != nil {
		return Place{}, err
	}

	if err := s.kitchen.EnsureMember(ctx, userID, item.KitchenID); err != nil {
		return Place{}, err
	}

	return item, nil
}

func (s *Service) Update(ctx context.Context, userID int64, placeID string, req placeRequest) (Place, error) {
	current, err := s.GetByID(ctx, userID, placeID)
	if err != nil {
		return Place{}, err
	}

	next, err := normalizePlaceInput(req)
	if err != nil {
		return Place{}, err
	}

	now := time.Now().Format(time.RFC3339)
	next.ID = current.ID
	next.KitchenID = current.KitchenID
	next.CreatedBy = current.CreatedBy
	next.CreatedAt = current.CreatedAt
	next.UpdatedBy = userID
	next.UpdatedAt = now
	next.VisitedAt = resolveVisitedAt(current.VisitedAt, next.Status, now)

	updated, err := s.repo.Update(ctx, next)
	if errors.Is(err, sql.ErrNoRows) {
		return Place{}, common.ErrNotFound
	}
	if err != nil {
		return Place{}, err
	}

	return updated, nil
}

func (s *Service) UpdateStatus(ctx context.Context, userID int64, placeID string, status string) (Place, error) {
	current, err := s.GetByID(ctx, userID, placeID)
	if err != nil {
		return Place{}, err
	}

	status = strings.TrimSpace(status)
	if !isAllowedStatus(status) {
		return Place{}, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	now := time.Now().Format(time.RFC3339)
	current.Status = status
	current.VisitedAt = resolveVisitedAt(current.VisitedAt, status, now)
	current.UpdatedBy = userID
	current.UpdatedAt = now

	updated, err := s.repo.Update(ctx, current)
	if errors.Is(err, sql.ErrNoRows) {
		return Place{}, common.ErrNotFound
	}
	if err != nil {
		return Place{}, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, userID int64, placeID string) error {
	current, err := s.GetByID(ctx, userID, placeID)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, current.ID, userID, time.Now().Format(time.RFC3339)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return common.ErrNotFound
		}
		return err
	}

	return nil
}

func normalizePlaceInput(req placeRequest) (Place, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Place{}, common.NewAppError(common.CodeBadRequest, "place name is required", http.StatusBadRequest)
	}
	if len([]rune(name)) > 40 {
		return Place{}, common.NewAppError(common.CodeBadRequest, "place name must be 40 characters or fewer", http.StatusBadRequest)
	}

	itemType := strings.TrimSpace(req.Type)
	if itemType == "" {
		itemType = TypeFood
	}
	if !isAllowedType(itemType) {
		return Place{}, common.NewAppError(common.CodeBadRequest, "invalid type", http.StatusBadRequest)
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = StatusWant
	}
	if !isAllowedStatus(status) {
		return Place{}, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = SourceManual
	}
	if !isAllowedSource(source) {
		return Place{}, common.NewAppError(common.CodeBadRequest, "invalid source", http.StatusBadRequest)
	}

	return Place{
		Name:      name,
		Type:      itemType,
		Address:   truncateRunes(strings.TrimSpace(req.Address), 120),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Price:     truncateRunes(strings.TrimSpace(req.Price), 40),
		Source:    source,
		SourceURL: truncateRunes(strings.TrimSpace(req.SourceURL), 300),
		ImageURLs: cleanStringList(req.ImageURLs, 9, 500),
		Status:    status,
		Tags:      cleanStringList(req.Tags, 8, 20),
		Note:      truncateRunes(strings.TrimSpace(req.Note), 300),
	}, nil
}

func resolveVisitedAt(current string, status string, now string) string {
	if status != StatusVisited {
		return ""
	}
	if strings.TrimSpace(current) != "" {
		return current
	}
	return now
}

func cleanStringList(values []string, limit int, maxRunes int) []string {
	if limit <= 0 {
		return []string{}
	}

	seen := map[string]struct{}{}
	items := make([]string, 0, limit)
	for _, value := range values {
		item := truncateRunes(strings.TrimSpace(value), maxRunes)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	return items
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func isAllowedStatus(value string) bool {
	_, ok := allowedStatuses[value]
	return ok
}

func isAllowedType(value string) bool {
	_, ok := allowedTypes[value]
	return ok
}

func isAllowedSource(value string) bool {
	_, ok := allowedSources[value]
	return ok
}
