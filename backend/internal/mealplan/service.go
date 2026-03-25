package mealplan

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
)

const (
	maxPlanItems = 20
	maxPlanNote  = 120
)

var allowedMealTypes = map[string]struct{}{
	"breakfast": {},
	"main":      {},
}

type Service struct {
	repo    *Repository
	kitchen *kitchen.Service
}

func NewService(repo *Repository, kitchenService *kitchen.Service) *Service {
	return &Service{
		repo:    repo,
		kitchen: kitchenService,
	}
}

func (s *Service) ListStoreByKitchenID(ctx context.Context, userID, kitchenID int64) (Store, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Store{}, err
	}

	plans, err := s.repo.ListByKitchenID(ctx, kitchenID)
	if err != nil {
		return Store{}, err
	}

	return groupPlansAsStore(plans), nil
}

func (s *Service) SaveDraft(ctx context.Context, userID, kitchenID int64, planDate string, req savePlanRequest) (Store, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Store{}, err
	}

	normalizedDate, err := normalizePlanDate(planDate)
	if err != nil {
		return Store{}, err
	}

	normalizedPlan, err := normalizePlanInput(req, false)
	if err != nil {
		return Store{}, err
	}
	if err := s.ensureRecipesBelongToKitchen(ctx, kitchenID, normalizedPlan.Items); err != nil {
		return Store{}, err
	}

	now := time.Now().Format(time.RFC3339)
	normalizedPlan.KitchenID = kitchenID
	normalizedPlan.PlanDate = normalizedDate
	normalizedPlan.Status = StatusDraft
	normalizedPlan.CreatedBy = userID
	normalizedPlan.UpdatedBy = userID
	normalizedPlan.CreatedAt = now
	normalizedPlan.UpdatedAt = now

	if err := s.repo.ReplaceDraft(ctx, normalizedPlan, now); err != nil {
		return Store{}, err
	}

	return s.ListStoreByKitchenID(ctx, userID, kitchenID)
}

func (s *Service) Submit(ctx context.Context, userID, kitchenID int64, planDate string, req savePlanRequest) (Store, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Store{}, err
	}

	normalizedDate, err := normalizePlanDate(planDate)
	if err != nil {
		return Store{}, err
	}

	normalizedPlan, err := normalizePlanInput(req, true)
	if err != nil {
		return Store{}, err
	}
	if err := s.ensureRecipesBelongToKitchen(ctx, kitchenID, normalizedPlan.Items); err != nil {
		return Store{}, err
	}

	now := time.Now().Format(time.RFC3339)
	normalizedPlan.KitchenID = kitchenID
	normalizedPlan.PlanDate = normalizedDate
	normalizedPlan.Status = StatusSubmitted
	normalizedPlan.CreatedBy = userID
	normalizedPlan.UpdatedBy = userID
	normalizedPlan.SubmittedBy = userID
	normalizedPlan.CreatedAt = now
	normalizedPlan.UpdatedAt = now
	normalizedPlan.SubmittedAt = now

	if err := s.repo.ReplaceSubmitted(ctx, normalizedPlan, now); err != nil {
		return Store{}, err
	}

	return s.ListStoreByKitchenID(ctx, userID, kitchenID)
}

func (s *Service) ensureRecipesBelongToKitchen(ctx context.Context, kitchenID int64, items []Item) error {
	if len(items) == 0 {
		return nil
	}

	recipeIDs := make([]string, 0, len(items))
	for _, item := range items {
		recipeIDs = append(recipeIDs, item.RecipeID)
	}

	count, err := s.repo.CountRecipesByKitchenID(ctx, kitchenID, recipeIDs)
	if err != nil {
		return err
	}
	if count != len(recipeIDs) {
		return common.NewAppError(common.CodeBadRequest, "recipeId must belong to the current kitchen", http.StatusBadRequest)
	}

	return nil
}

func normalizePlanDate(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", common.NewAppError(common.CodeBadRequest, "planDate is required", http.StatusBadRequest)
	}
	if _, err := time.Parse("2006-01-02", trimmed); err != nil {
		return "", common.NewAppError(common.CodeBadRequest, "invalid planDate", http.StatusBadRequest)
	}
	return trimmed, nil
}

func normalizePlanInput(req savePlanRequest, requireItems bool) (Plan, error) {
	note := strings.TrimSpace(req.Note)
	if len([]rune(note)) > maxPlanNote {
		return Plan{}, common.NewAppError(common.CodeBadRequest, fmt.Sprintf("note must be %d characters or fewer", maxPlanNote), http.StatusBadRequest)
	}

	sourceItems := req.Items
	if len(sourceItems) > maxPlanItems {
		return Plan{}, common.NewAppError(common.CodeBadRequest, fmt.Sprintf("items must be %d or fewer", maxPlanItems), http.StatusBadRequest)
	}

	normalizedItems := make([]Item, 0, len(sourceItems))
	seenRecipeIDs := make(map[string]struct{}, len(sourceItems))
	for index, raw := range sourceItems {
		recipeID := strings.TrimSpace(raw.RecipeID)
		if recipeID == "" {
			return Plan{}, common.NewAppError(common.CodeBadRequest, "recipeId is required", http.StatusBadRequest)
		}
		if _, exists := seenRecipeIDs[recipeID]; exists {
			return Plan{}, common.NewAppError(common.CodeBadRequest, "duplicate recipeId is not allowed", http.StatusBadRequest)
		}
		seenRecipeIDs[recipeID] = struct{}{}

		quantity := raw.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		if quantity > 9 {
			return Plan{}, common.NewAppError(common.CodeBadRequest, "quantity must be between 1 and 9", http.StatusBadRequest)
		}

		mealType := strings.TrimSpace(raw.MealTypeSnapshot)
		if mealType == "" {
			mealType = "main"
		}
		if _, ok := allowedMealTypes[mealType]; !ok {
			return Plan{}, common.NewAppError(common.CodeBadRequest, "invalid mealTypeSnapshot", http.StatusBadRequest)
		}

		title := strings.TrimSpace(raw.TitleSnapshot)
		if title == "" {
			return Plan{}, common.NewAppError(common.CodeBadRequest, "titleSnapshot is required", http.StatusBadRequest)
		}

		normalizedItems = append(normalizedItems, Item{
			RecipeID:         recipeID,
			Quantity:         quantity,
			MealTypeSnapshot: mealType,
			TitleSnapshot:    title,
			ImageSnapshot:    strings.TrimSpace(raw.ImageSnapshot),
			Sort:             index,
		})
	}

	if requireItems && len(normalizedItems) == 0 {
		return Plan{}, common.NewAppError(common.CodeBadRequest, "at least one dish is required", http.StatusBadRequest)
	}

	return Plan{
		Note:  note,
		Items: normalizedItems,
	}, nil
}

func groupPlansAsStore(plans []Plan) Store {
	store := Store{
		Drafts:    map[string]Plan{},
		Submitted: make([]Plan, 0),
	}
	for _, plan := range plans {
		switch plan.Status {
		case StatusDraft:
			store.Drafts[plan.PlanDate] = plan
		case StatusSubmitted:
			store.Submitted = append(store.Submitted, plan)
		}
	}

	sort.Slice(store.Submitted, func(left, right int) bool {
		leftSubmittedAt := strings.TrimSpace(store.Submitted[left].SubmittedAt)
		rightSubmittedAt := strings.TrimSpace(store.Submitted[right].SubmittedAt)
		if leftSubmittedAt != rightSubmittedAt {
			return rightSubmittedAt < leftSubmittedAt
		}
		if store.Submitted[left].PlanDate != store.Submitted[right].PlanDate {
			return store.Submitted[right].PlanDate < store.Submitted[left].PlanDate
		}
		return store.Submitted[left].ID > store.Submitted[right].ID
	})

	return store
}
