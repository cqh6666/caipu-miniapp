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

func (s *Service) ListByKitchenID(ctx context.Context, userID, kitchenID int64, filter ListFilter) ([]Recipe, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return nil, err
	}

	filter.MealType = strings.TrimSpace(filter.MealType)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Keyword = strings.TrimSpace(filter.Keyword)

	if filter.MealType != "" && !isAllowedMealType(filter.MealType) {
		return nil, common.NewAppError(common.CodeBadRequest, "invalid mealType", http.StatusBadRequest)
	}

	if filter.Status != "" && !isAllowedStatus(filter.Status) {
		return nil, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	return s.repo.ListByKitchenID(ctx, kitchenID, filter)
}

func (s *Service) Create(ctx context.Context, userID, kitchenID int64, req createRecipeRequest) (Recipe, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Recipe{}, err
	}

	item, err := normalizeRecipeInput(req.Title, req.Ingredient, req.Link, req.ImageURL, req.MealType, req.Status, req.Note, req.ParsedContent)
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

	return s.repo.Create(ctx, item)
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

	return item, nil
}

func (s *Service) Update(ctx context.Context, userID int64, recipeID string, req updateRecipeRequest) (Recipe, error) {
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return Recipe{}, err
	}

	next, err := normalizeRecipeInput(req.Title, req.Ingredient, req.Link, req.ImageURL, req.MealType, req.Status, req.Note, req.ParsedContent)
	if err != nil {
		return Recipe{}, err
	}

	next.ID = current.ID
	next.KitchenID = current.KitchenID
	next.CreatedBy = current.CreatedBy
	next.CreatedAt = current.CreatedAt
	next.UpdatedBy = userID
	next.UpdatedAt = time.Now().Format(time.RFC3339)

	updated, err := s.repo.Update(ctx, next)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	}
	if err != nil {
		return Recipe{}, err
	}

	return updated, nil
}

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
	current.UpdatedBy = userID
	current.UpdatedAt = now
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

func normalizeRecipeInput(
	title string,
	ingredient string,
	link string,
	imageURL string,
	mealType string,
	status string,
	note string,
	parsedContent ParsedContent,
) (Recipe, error) {
	title = strings.TrimSpace(title)
	ingredient = strings.TrimSpace(ingredient)
	link = strings.TrimSpace(link)
	imageURL = strings.TrimSpace(imageURL)
	mealType = strings.TrimSpace(mealType)
	status = strings.TrimSpace(status)
	note = strings.TrimSpace(note)

	if title == "" {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "title is required", http.StatusBadRequest)
	}
	if len([]rune(title)) > 40 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "title must be 40 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(ingredient)) > 60 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "ingredient must be 60 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(link)) > 300 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "link must be 300 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(note)) > 300 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "note must be 300 characters or fewer", http.StatusBadRequest)
	}

	if mealType == "" {
		mealType = "breakfast"
	}
	if status == "" {
		status = "wishlist"
	}

	if !isAllowedMealType(mealType) {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "invalid mealType", http.StatusBadRequest)
	}
	if !isAllowedStatus(status) {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	normalizedContent := normalizeParsedContent(parsedContent, mealType, title, ingredient)

	return Recipe{
		Title:         title,
		Ingredient:    ingredient,
		Link:          link,
		ImageURL:      imageURL,
		MealType:      mealType,
		Status:        status,
		Note:          note,
		ParsedContent: normalizedContent,
	}, nil
}

func normalizeParsedContent(content ParsedContent, mealType, title, ingredient string) ParsedContent {
	ingredients := cleanLines(content.Ingredients)
	steps := cleanLines(content.Steps)

	if len(ingredients) > 0 || len(steps) > 0 {
		return ParsedContent{
			Ingredients: ingredients,
			Steps:       steps,
		}
	}

	mainIngredient := ingredient
	if mainIngredient == "" {
		mainIngredient = title
	}
	if mainIngredient == "" {
		mainIngredient = "主食材"
	}

	mealLabel := "早餐"
	if mealType == "main" {
		mealLabel = "正餐"
	}

	return ParsedContent{
		Ingredients: []string{
			mainIngredient + " 1份",
			mealLabel + "常用配菜 适量",
			"基础调味 适量",
		},
		Steps: []string{
			"先整理这道菜的核心做法。",
			"按自己的口味调整成容易复刻的版本。",
			"做完以后补充口感和火候记录。",
		},
	}
}

func cleanLines(lines []string) []string {
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		items = append(items, line)
	}
	return items
}

func isAllowedMealType(value string) bool {
	_, ok := allowedMealTypes[value]
	return ok
}

func isAllowedStatus(value string) bool {
	_, ok := allowedStatuses[value]
	return ok
}
