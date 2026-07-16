package recipe

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
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

var (
	allowedMealTypes = map[string]struct{}{
		"breakfast": {},
		"main":      {},
	}
	allowedStatuses = map[string]struct{}{
		"wishlist": {},
		"done":     {},
	}
)

const (
	maxRecipeImages       = 9
	maxRecipeSummaryRunes = 24
	maxParsedSteps        = 6
)

type Service struct {
	repo              *Repository
	kitchen           *kitchen.Service
	upload            *upload.Service
	flowchart         *FlowchartGenerator
	flowchartEnabled  bool
	autoParseEstimate queueEstimateConfig
	flowchartEstimate queueEstimateConfig
}

type ServiceOptions struct {
	Repo               *Repository
	KitchenService     *kitchen.Service
	UploadService      *upload.Service
	Flowchart          *FlowchartGenerator
	FlowchartEnabled   bool
	AutoParseEnabled   bool
	AutoParseInterval  time.Duration
	AutoParseBatchSize int
	FlowchartInterval  time.Duration
	FlowchartBatchSize int
}

func NewService(opts ServiceOptions) *Service {
	return &Service{
		repo:             opts.Repo,
		kitchen:          opts.KitchenService,
		upload:           opts.UploadService,
		flowchart:        opts.Flowchart,
		flowchartEnabled: opts.FlowchartEnabled,
		autoParseEstimate: queueEstimateConfig{
			enabled:         opts.AutoParseEnabled,
			interval:        opts.AutoParseInterval,
			batchSize:       opts.AutoParseBatchSize,
			averageDuration: defaultAutoParseEstimatedDuration,
		},
		flowchartEstimate: queueEstimateConfig{
			enabled:         opts.FlowchartEnabled && opts.Flowchart != nil && opts.Flowchart.IsConfigured(),
			interval:        opts.FlowchartInterval,
			batchSize:       opts.FlowchartBatchSize,
			averageDuration: defaultFlowchartEstimatedDuration,
		},
	}
}

func (s *Service) ListByKitchenID(ctx context.Context, userID, kitchenID int64, filter ListFilter) ([]Recipe, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return nil, err
	}

	filter.MealType = strings.TrimSpace(filter.MealType)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	filter.TitleKeyword = strings.TrimSpace(filter.TitleKeyword)
	filter.IngredientKeyword = strings.TrimSpace(filter.IngredientKeyword)
	filter.TitleOrIngredientKeyword = strings.TrimSpace(filter.TitleOrIngredientKeyword)

	if filter.MealType != "" && !isAllowedMealType(filter.MealType) {
		return nil, common.NewAppError(common.CodeBadRequest, "invalid mealType", http.StatusBadRequest)
	}

	if filter.Status != "" && !isAllowedStatus(filter.Status) {
		return nil, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	return s.repo.ListByKitchenID(ctx, kitchenID, filter)
}

func (s *Service) CreateFromInput(ctx context.Context, userID, kitchenID int64, input CreateInput) (Recipe, error) {
	return s.Create(ctx, userID, kitchenID, createRecipeRequest{
		Title:               input.Title,
		TitleSource:         input.TitleSource,
		Ingredient:          input.Ingredient,
		Summary:             input.Summary,
		Link:                input.Link,
		ImageURL:            input.ImageURL,
		ImageURLs:           input.ImageURLs,
		MealType:            input.MealType,
		Status:              input.Status,
		Note:                input.Note,
		ParsedContent:       input.ParsedContent,
		ParsedContentEdited: input.ParsedContentEdited,
	})
}

func (s *Service) Create(ctx context.Context, userID, kitchenID int64, req createRecipeRequest) (Recipe, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Recipe{}, err
	}

	item, err := normalizeRecipeInput(req.Title, req.Ingredient, req.Summary, req.Link, req.ImageURL, req.ImageURLs, req.MealType, req.Status, req.Note, req.ParsedContent)
	if err != nil {
		return Recipe{}, err
	}

	recipeID, err := common.NewPrefixedID("rec")
	if err != nil {
		return Recipe{}, fmt.Errorf("generate recipe id: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	item.ID = recipeID
	item.KitchenID = kitchenID
	item.CreatedBy = userID
	item.UpdatedBy = userID
	item.CreatedAt = now
	item.UpdatedAt = now
	item.Version = 1
	item.TitleSource = normalizeTitleSource(req.TitleSource)
	item.ImageMetas = buildSubmittedImageMetas(item.ImageURLs, Recipe{}, s.upload)
	item.ImageURLs = recipeImageURLsFromMetas(item.ImageMetas)
	item.ImageURL = firstImageURL(item.ImageURLs)
	applyCreateParseState(&item, req, now)
	item.ParsedContentEdited = resolveCreateParsedContentEditedState(item, req)

	created, err := s.repo.Create(ctx, item)
	if err != nil {
		return Recipe{}, err
	}
	return s.decorateRecipeRuntimeState(ctx, created), nil
}

func (s *Service) GetByID(ctx context.Context, userID int64, recipeID string) (Recipe, error) {
	item, err := s.repo.FindByID(ctx, recipeID)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	}
	if err != nil {
		return Recipe{}, err
	}

	if err := s.kitchen.EnsureMember(ctx, userID, item.KitchenID); err != nil {
		return Recipe{}, err
	}

	return s.decorateRecipeRuntimeState(ctx, item), nil
}

func (s *Service) Update(ctx context.Context, userID int64, recipeID string, req updateRecipeRequest) (Recipe, error) {
	expectedVersion, err := requireRecipeVersion(req.Version)
	if err != nil {
		return Recipe{}, err
	}
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return Recipe{}, err
	}
	if current.Version != expectedVersion {
		return Recipe{}, recipeVersionConflictError()
	}

	summary := req.Summary
	if strings.TrimSpace(summary) == "" {
		summary = current.Summary
	}

	next, err := normalizeRecipeInput(req.Title, req.Ingredient, summary, req.Link, req.ImageURL, req.ImageURLs, req.MealType, req.Status, req.Note, req.ParsedContent)
	if err != nil {
		return Recipe{}, err
	}

	next.ID = current.ID
	next.KitchenID = current.KitchenID
	next.TitleSource = resolveUpdateTitleSource(current, next, req)
	next.PinnedAt = current.PinnedAt
	next.FlowchartImageURL = current.FlowchartImageURL
	next.FlowchartProvider = current.FlowchartProvider
	next.FlowchartModel = current.FlowchartModel
	next.FlowchartStatus = current.FlowchartStatus
	next.FlowchartError = current.FlowchartError
	next.FlowchartRequestedAt = current.FlowchartRequestedAt
	next.FlowchartFinishedAt = current.FlowchartFinishedAt
	next.FlowchartUpdatedAt = current.FlowchartUpdatedAt
	next.FlowchartSourceHash = current.FlowchartSourceHash
	next.CreatedBy = current.CreatedBy
	next.CreatedAt = current.CreatedAt
	next.UpdatedBy = userID
	next.UpdatedAt = time.Now().Format(time.RFC3339)
	next.Version = expectedVersion
	next.ImageMetas = buildSubmittedImageMetas(next.ImageURLs, current, s.upload)
	next.ImageURLs = recipeImageURLsFromMetas(next.ImageMetas)
	next.ImageURL = firstImageURL(next.ImageURLs)
	applyUpdateParseState(&next, current, req, next.UpdatedAt)
	next.ParsedContentEdited = resolveUpdateParsedContentEditedState(current, next, req)

	updated, err := s.repo.Update(ctx, next)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	}
	if errors.Is(err, errRecipeVersionConflict) {
		return Recipe{}, recipeVersionConflictError()
	}
	if err != nil {
		return Recipe{}, err
	}

	return s.decorateRecipeRuntimeState(ctx, updated), nil
}

func requireRecipeVersion(version *int64) (int64, error) {
	if version == nil || *version < 1 {
		return 0, common.NewAppError(common.CodeBadRequest, "version is required", http.StatusBadRequest)
	}
	return *version, nil
}

func recipeVersionConflictError() error {
	return common.NewAppError(
		common.CodeConflict,
		"recipe has been updated; reload and try again",
		http.StatusConflict,
	)
}
