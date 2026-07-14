package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/dietassistant"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
)

type dietAssistantRecipeService interface {
	ListByKitchenID(context.Context, int64, int64, recipe.ListFilter) ([]recipe.Recipe, error)
	CreateFromInput(context.Context, int64, int64, recipe.CreateInput) (recipe.Recipe, error)
	GetByID(context.Context, int64, string) (recipe.Recipe, error)
}

type dietAssistantLinkParser interface {
	ParseRecipeLink(context.Context, string) (linkparse.RecipeParseOutcome, error)
}

type dietAssistantRecipeTools struct {
	countRecipes  dietassistant.CountRecipesFunc
	createFromURL dietassistant.CreateFromURLFunc
	searchRecipes dietassistant.SearchRecipesFunc
	getRecipeByID dietassistant.GetRecipeByIDFunc
}

func newDietAssistantService(
	cfg config.Config,
	repo *dietassistant.Repository,
	ensureMember dietassistant.EnsureMemberFunc,
	recipeService dietAssistantRecipeService,
	linkParser dietAssistantLinkParser,
) *dietassistant.Service {
	tools := buildDietAssistantRecipeTools(recipeService, linkParser)
	return dietassistant.NewService(dietassistant.Options{
		BaseURL:         cfg.DietAssistantAIBaseURL,
		APIKey:          cfg.DietAssistantAIAPIKey,
		Model:           cfg.DietAssistantAIModel,
		ThinkingType:    cfg.DietAssistantAIThinkingType,
		ReasoningEffort: cfg.DietAssistantAIReasoningEffort,
		Timeout:         time.Duration(cfg.DietAssistantAITimeoutSec) * time.Second,
		Repo:            repo,
		EnsureMember:    ensureMember,
		CountRecipes:    tools.countRecipes,
		CreateFromURL:   tools.createFromURL,
		SearchRecipes:   tools.searchRecipes,
		GetRecipeByID:   tools.getRecipeByID,
	})
}

func buildDietAssistantRecipeTools(
	recipeService dietAssistantRecipeService,
	linkParser dietAssistantLinkParser,
) dietAssistantRecipeTools {
	return dietAssistantRecipeTools{
		countRecipes: func(ctx context.Context, input dietassistant.RecipeCountInput) (int, error) {
			items, err := recipeService.ListByKitchenID(ctx, input.UserID, input.KitchenID, recipe.ListFilter{
				MealType: input.MealType,
				Status:   input.Status,
			})
			if err != nil {
				return 0, err
			}
			return len(items), nil
		},
		createFromURL: func(ctx context.Context, input dietassistant.RecipeFromURLInput) (dietassistant.RecipeFromURLResult, error) {
			outcome, err := linkParser.ParseRecipeLink(ctx, input.URL)
			if err != nil {
				return dietassistant.RecipeFromURLResult{}, err
			}
			draft := outcome.RecipeDraft
			parsedContent := buildRecipeParsedContentFromLinkDraft(draft.ParsedContent)
			parsedContentEdited := false
			imageURLs := cleanDietAssistantRecipeImageURLs(append(draft.ImageURLs, strings.TrimSpace(draft.ImageURL)))
			item, err := recipeService.CreateFromInput(ctx, input.UserID, input.KitchenID, recipe.CreateInput{
				Title:               truncateDietAssistantRecipeText(firstNonEmptyDietAssistantText(draft.Title, "链接菜谱"), 40),
				Ingredient:          truncateDietAssistantRecipeText(draft.Ingredient, 60),
				Summary:             truncateDietAssistantRecipeText(draft.Summary, 24),
				Link:                truncateDietAssistantRecipeText(firstNonEmptyDietAssistantText(draft.Link, input.URL), 300),
				ImageURL:            truncateDietAssistantRecipeText(draft.ImageURL, 500),
				ImageURLs:           imageURLs,
				MealType:            input.MealType,
				Status:              input.Status,
				Note:                truncateDietAssistantRecipeText(draft.Note, 300),
				ParsedContent:       parsedContent,
				ParsedContentEdited: &parsedContentEdited,
			})
			if err != nil {
				return dietassistant.RecipeFromURLResult{}, err
			}
			return dietassistant.RecipeFromURLResult{
				Recipe:               buildDietAssistantRecipeToolItem(item),
				Source:               firstNonEmptyDietAssistantText(outcome.Source, linkparse.DetectParsePlatform(input.URL)),
				SourceDetail:         strings.TrimSpace(outcome.SourceDetail),
				SummaryMode:          strings.TrimSpace(outcome.SummaryMode),
				MainIngredients:      cleanDietAssistantRecipeLines(parsedContent.MainIngredients, 8),
				SecondaryIngredients: cleanDietAssistantRecipeLines(parsedContent.SecondaryIngredients, 12),
				StepsCount:           len(parsedContent.Steps),
				Warnings:             cleanDietAssistantRecipeLines(outcome.Warnings, 5),
			}, nil
		},
		searchRecipes: func(ctx context.Context, input dietassistant.RecipeSearchInput) ([]dietassistant.RecipeToolItem, error) {
			filter := recipe.ListFilter{
				MealType: input.MealType,
				Status:   input.Status,
			}
			keyword := strings.TrimSpace(input.Keyword)
			switch input.SearchScope {
			case "title":
				filter.TitleKeyword = keyword
			case "ingredient":
				filter.IngredientKeyword = keyword
			default:
				filter.TitleOrIngredientKeyword = keyword
			}
			items, err := recipeService.ListByKitchenID(ctx, input.UserID, input.KitchenID, filter)
			if err != nil {
				return nil, err
			}
			limit := input.Limit
			if limit <= 0 || limit > 10 {
				limit = 10
			}
			result := make([]dietassistant.RecipeToolItem, 0, min(len(items), limit))
			for index, item := range items {
				if index >= limit {
					break
				}
				result = append(result, buildDietAssistantRecipeToolItem(item))
			}
			return result, nil
		},
		getRecipeByID: func(ctx context.Context, input dietassistant.RecipeGetInput) (dietassistant.RecipeDetailToolItem, error) {
			item, err := recipeService.GetByID(ctx, input.UserID, input.RecipeID)
			if err != nil {
				return dietassistant.RecipeDetailToolItem{}, err
			}
			if input.KitchenID > 0 && item.KitchenID != input.KitchenID {
				return dietassistant.RecipeDetailToolItem{}, errors.New("recipe not found in current kitchen")
			}
			return buildDietAssistantRecipeDetailToolItem(item), nil
		},
	}
}

func buildDietAssistantRecipeToolItem(item recipe.Recipe) dietassistant.RecipeToolItem {
	return dietassistant.RecipeToolItem{
		ID:         item.ID,
		Title:      item.Title,
		MealType:   item.MealType,
		Status:     item.Status,
		Ingredient: item.Ingredient,
		Summary:    item.Summary,
		Note:       item.Note,
		Link:       item.Link,
	}
}

func buildDietAssistantRecipeDetailToolItem(item recipe.Recipe) dietassistant.RecipeDetailToolItem {
	return dietassistant.RecipeDetailToolItem{
		RecipeToolItem:       buildDietAssistantRecipeToolItem(item),
		ImageURL:             item.ImageURL,
		ImageURLs:            cleanDietAssistantRecipeImageURLs(item.ImageURLs),
		MainIngredients:      cleanDietAssistantRecipeLines(item.ParsedContent.MainIngredients, 20),
		SecondaryIngredients: cleanDietAssistantRecipeLines(item.ParsedContent.SecondaryIngredients, 20),
		Steps:                buildDietAssistantRecipeStepToolItems(item.ParsedContent.Steps, 20),
		StepsCount:           len(item.ParsedContent.Steps),
		ParseStatus:          item.ParseStatus,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func buildDietAssistantRecipeStepToolItems(steps []recipe.ParsedStep, limit int) []dietassistant.RecipeStepToolItem {
	items := make([]dietassistant.RecipeStepToolItem, 0, len(steps))
	for _, step := range steps {
		title := truncateDietAssistantRecipeText(step.Title, 80)
		detail := truncateDietAssistantRecipeText(step.Detail, 500)
		if title == "" && detail == "" {
			continue
		}
		items = append(items, dietassistant.RecipeStepToolItem{
			Title:  title,
			Detail: detail,
		})
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items
}

func buildRecipeParsedContentFromLinkDraft(content linkparse.ParsedContent) recipe.ParsedContent {
	steps := make([]recipe.ParsedStep, 0, len(content.Steps))
	for _, step := range content.Steps {
		title := strings.TrimSpace(step.Title)
		detail := strings.TrimSpace(step.Detail)
		if title == "" && detail == "" {
			continue
		}
		steps = append(steps, recipe.ParsedStep{
			Title:  title,
			Detail: detail,
		})
	}
	return recipe.ParsedContent{
		MainIngredients:      cleanDietAssistantRecipeLines(content.MainIngredients, 0),
		SecondaryIngredients: cleanDietAssistantRecipeLines(content.SecondaryIngredients, 0),
		Steps:                steps,
	}
}

func cleanDietAssistantRecipeLines(values []string, limit int) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items
}

func cleanDietAssistantRecipeImageURLs(values []string) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		value := truncateDietAssistantRecipeText(item, 500)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
		if len(items) >= 9 {
			break
		}
	}
	return items
}

func firstNonEmptyDietAssistantText(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}

func truncateDietAssistantRecipeText(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	if maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
