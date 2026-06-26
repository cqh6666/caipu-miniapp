package place

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
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

	ExternalProviderAMap = "amap"
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
	upload  *upload.Service
}

type statusUpdateInput struct {
	Status           string
	VisitedAt        *string
	RevisitRating    *int
	RecommendedItems []string
}

func NewService(repo *Repository, kitchenService *kitchen.Service) *Service {
	return &Service{repo: repo, kitchen: kitchenService}
}

func (s *Service) SetUploadService(uploadService *upload.Service) {
	s.upload = uploadService
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

	item, err := normalizePlaceCreateInput(req)
	if err != nil {
		return Place{}, err
	}
	item.ImageURLs = s.mirrorExternalImages(ctx, item.ImageURLs)

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
	applyPlacePriceStatsFields(&item)

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

	next, err := normalizePlaceUpdateInput(current, req)
	if err != nil {
		return Place{}, err
	}
	next.ImageURLs = s.mirrorExternalImages(ctx, next.ImageURLs)

	now := time.Now().Format(time.RFC3339)
	next.ID = current.ID
	next.KitchenID = current.KitchenID
	next.CreatedBy = current.CreatedBy
	next.CreatedAt = current.CreatedAt
	next.UpdatedBy = userID
	next.UpdatedAt = now
	next.VisitedAt = resolveVisitedAt(firstNonEmpty(next.VisitedAt, current.VisitedAt), next.Status, now)
	applyPlacePriceStatsFields(&next)

	updated, err := s.repo.Update(ctx, next)
	if errors.Is(err, sql.ErrNoRows) {
		return Place{}, common.ErrNotFound
	}
	if err != nil {
		return Place{}, err
	}

	return updated, nil
}

func (s *Service) UpdateStatus(ctx context.Context, userID int64, placeID string, input statusUpdateInput) (Place, error) {
	current, err := s.GetByID(ctx, userID, placeID)
	if err != nil {
		return Place{}, err
	}

	status := strings.TrimSpace(input.Status)
	if !isAllowedStatus(status) {
		return Place{}, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	now := time.Now().Format(time.RFC3339)
	current.Status = status
	if input.VisitedAt != nil {
		current.VisitedAt = truncateRunes(strings.TrimSpace(*input.VisitedAt), 40)
	}
	current.VisitedAt = resolveVisitedAt(current.VisitedAt, status, now)
	if input.RevisitRating != nil {
		rating, err := normalizeRevisitRating(*input.RevisitRating)
		if err != nil {
			return Place{}, err
		}
		current.RevisitRating = rating
	}
	if input.RecommendedItems != nil {
		current.RecommendedItems = cleanStringList(input.RecommendedItems, 12, 24)
	}
	if status != StatusVisited {
		current.RevisitRating = 0
		current.RecommendedItems = []string{}
	}
	current.UpdatedBy = userID
	current.UpdatedAt = now
	applyPlacePriceStatsFields(&current)

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

func (s *Service) mirrorExternalImages(ctx context.Context, imageURLs []string) []string {
	if s.upload == nil || len(imageURLs) == 0 {
		return imageURLs
	}

	next := make([]string, 0, len(imageURLs))
	seen := map[string]struct{}{}
	for _, raw := range imageURLs {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}

		resolved := value
		if isRemoteImageURL(value) && !s.upload.IsManagedImageURL(value) {
			if image, err := s.upload.SaveRemoteImage(ctx, value); err == nil && strings.TrimSpace(image.URL) != "" {
				resolved = strings.TrimSpace(image.URL)
			} else {
				continue
			}
		}

		key := strings.ToLower(resolved)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		next = append(next, resolved)
		if len(next) >= 9 {
			break
		}
	}
	return next
}

func isRemoteImageURL(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func normalizePlaceCreateInput(req placeRequest) (Place, error) {
	return normalizePlaceUpdateInput(Place{}, req)
}

func normalizePlaceUpdateInput(base Place, req placeRequest) (Place, error) {
	next := base
	if next.Type == "" {
		next.Type = TypeFood
	}
	if next.Status == "" {
		next.Status = StatusWant
	}
	if next.Source == "" {
		next.Source = SourceManual
	}

	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	name := strings.TrimSpace(next.Name)
	if name == "" {
		return Place{}, common.NewAppError(common.CodeBadRequest, "place name is required", http.StatusBadRequest)
	}
	if len([]rune(name)) > 40 {
		return Place{}, common.NewAppError(common.CodeBadRequest, "place name must be 40 characters or fewer", http.StatusBadRequest)
	}
	next.Name = name

	if req.Type != nil {
		next.Type = strings.TrimSpace(*req.Type)
	}
	if strings.TrimSpace(next.Type) == "" {
		next.Type = TypeFood
	}
	if !isAllowedType(next.Type) {
		return Place{}, common.NewAppError(common.CodeBadRequest, "invalid type", http.StatusBadRequest)
	}

	if req.Status != nil {
		next.Status = strings.TrimSpace(*req.Status)
	}
	if strings.TrimSpace(next.Status) == "" {
		next.Status = StatusWant
	}
	if !isAllowedStatus(next.Status) {
		return Place{}, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	if req.Source != nil {
		next.Source = strings.TrimSpace(*req.Source)
	}
	if strings.TrimSpace(next.Source) == "" {
		next.Source = SourceManual
	}
	if !isAllowedSource(next.Source) {
		return Place{}, common.NewAppError(common.CodeBadRequest, "invalid source", http.StatusBadRequest)
	}

	if req.Address != nil {
		next.Address = truncateRunes(strings.TrimSpace(*req.Address), 120)
	}
	if req.Latitude != nil {
		next.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		next.Longitude = *req.Longitude
	}
	if req.Price != nil {
		next.Price = truncateRunes(strings.TrimSpace(*req.Price), 40)
	}
	if req.SourceURL != nil {
		next.SourceURL = truncateRunes(strings.TrimSpace(*req.SourceURL), 300)
	}
	if req.ImageURLs != nil {
		next.ImageURLs = cleanStringList(req.ImageURLs, 9, 500)
	}
	if req.Tags != nil {
		next.Tags = cleanStringList(req.Tags, 8, 20)
	}
	if req.Note != nil {
		next.Note = truncateRunes(strings.TrimSpace(*req.Note), 300)
	}
	if req.VisitedAt != nil {
		next.VisitedAt = truncateRunes(strings.TrimSpace(*req.VisitedAt), 40)
	}
	if req.RevisitRating != nil {
		rating, err := normalizeRevisitRating(*req.RevisitRating)
		if err != nil {
			return Place{}, err
		}
		next.RevisitRating = rating
	}
	if req.RecommendedItems != nil {
		next.RecommendedItems = cleanStringList(req.RecommendedItems, 12, 24)
	}
	if req.Phone != nil {
		next.Phone = truncateRunes(strings.TrimSpace(*req.Phone), 40)
	}
	if req.ExternalProvider != nil {
		next.ExternalProvider = normalizeExternalProvider(*req.ExternalProvider)
	}
	if req.ExternalPOIID != nil {
		next.ExternalPOIID = truncateRunes(strings.TrimSpace(*req.ExternalPOIID), 80)
	}
	if req.Rating != nil {
		next.Rating = truncateRunes(strings.TrimSpace(*req.Rating), 20)
	}
	if req.DiningTips != nil {
		next.DiningTips = truncateRunes(strings.TrimSpace(*req.DiningTips), 160)
	}
	if req.Scenes != nil {
		next.Scenes = cleanStringList(req.Scenes, 8, 12)
	}
	if req.BestTime != nil {
		next.BestTime = truncateRunes(strings.TrimSpace(*req.BestTime), 60)
	}
	if req.Duration != nil {
		next.Duration = truncateRunes(strings.TrimSpace(*req.Duration), 20)
	}
	if req.CompanionTags != nil {
		next.CompanionTags = cleanStringList(req.CompanionTags, 6, 12)
	}
	if req.ParkingNote != nil {
		next.ParkingNote = truncateRunes(strings.TrimSpace(*req.ParkingNote), 160)
	}
	if next.Status != StatusVisited {
		next.VisitedAt = ""
		next.RevisitRating = 0
		next.RecommendedItems = []string{}
	}
	return next, nil
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

func normalizeRevisitRating(value int) (int, error) {
	if value < 0 || value > 5 {
		return 0, common.NewAppError(common.CodeBadRequest, "revisitRating must be between 0 and 5", http.StatusBadRequest)
	}
	return value, nil
}

func normalizeExternalProvider(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case ExternalProviderAMap:
		return value
	default:
		return truncateRunes(value, 40)
	}
}

var priceAmountPattern = regexp.MustCompile(`\d+(?:\.\d+)?`)

func applyPlacePriceStatsFields(item *Place) {
	if item == nil {
		return
	}
	amountCents, priceType := parsePlacePrice(item.Price)
	item.PriceAmountCents = amountCents
	item.PriceCurrency = "CNY"
	item.PriceType = priceType
}

func parsePlacePrice(price string) (int64, string) {
	value := strings.TrimSpace(price)
	if value == "" {
		return 0, ""
	}
	match := priceAmountPattern.FindString(value)
	if match == "" {
		return 0, ""
	}
	amount, err := strconv.ParseFloat(match, 64)
	if err != nil || amount <= 0 {
		return 0, ""
	}
	priceType := "amount"
	lower := strings.ToLower(value)
	if strings.Contains(value, "人均") || strings.Contains(value, "/人") || strings.Contains(value, "每人") || strings.Contains(lower, "per") {
		priceType = "per_person"
	}
	return int64(amount*100 + 0.5), priceType
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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
