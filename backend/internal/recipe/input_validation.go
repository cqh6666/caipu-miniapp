package recipe

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func normalizeRecipeInput(
	title string,
	ingredient string,
	summary string,
	link string,
	imageURL string,
	imageURLs []string,
	mealType string,
	status string,
	note string,
	parsedContent ParsedContent,
) (Recipe, error) {
	title = strings.TrimSpace(title)
	ingredient = strings.TrimSpace(ingredient)
	summary = strings.TrimSpace(summary)
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
	if len([]rune(summary)) > maxRecipeSummaryRunes {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "summary must be 24 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(link)) > 300 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "link must be 300 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(imageURL)) > 500 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "imageUrl must be 500 characters or fewer", http.StatusBadRequest)
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

	normalizedImages, err := normalizeImageURLs(imageURL, imageURLs)
	if err != nil {
		return Recipe{}, err
	}
	normalizedContent := normalizeParsedContent(parsedContent, mealType, title, ingredient)

	return Recipe{
		Title:         title,
		Ingredient:    ingredient,
		Summary:       summary,
		Link:          link,
		ImageURL:      firstImageURL(normalizedImages),
		ImageURLs:     normalizedImages,
		MealType:      mealType,
		Status:        status,
		Note:          note,
		ParsedContent: normalizedContent,
	}, nil
}

func normalizeTitleSource(value string) string {
	switch strings.TrimSpace(value) {
	case TitleSourcePlaceholder:
		return TitleSourcePlaceholder
	case TitleSourceParsed:
		return TitleSourceParsed
	default:
		return TitleSourceManual
	}
}

func resolveUpdateTitleSource(current Recipe, next Recipe, req updateRecipeRequest) string {
	if strings.TrimSpace(next.Title) != strings.TrimSpace(current.Title) {
		return TitleSourceManual
	}
	requested := strings.TrimSpace(req.TitleSource)
	if requested != "" {
		return normalizeTitleSource(requested)
	}
	return normalizeTitleSource(current.TitleSource)
}

func normalizeImageURLs(imageURL string, imageURLs []string) ([]string, error) {
	candidates := make([]string, 0, len(imageURLs)+1)
	candidates = append(candidates, imageURLs...)
	if strings.TrimSpace(imageURL) != "" {
		candidates = append(candidates, imageURL)
	}

	normalized := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, raw := range candidates {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if len([]rune(value)) > 500 {
			return nil, common.NewAppError(common.CodeBadRequest, "each imageUrl must be 500 characters or fewer", http.StatusBadRequest)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	if len(normalized) > maxRecipeImages {
		return nil, common.NewAppError(common.CodeBadRequest, "at most 9 images are allowed", http.StatusBadRequest)
	}

	return normalized, nil
}

func firstImageURL(imageURLs []string) string {
	if len(imageURLs) == 0 {
		return ""
	}
	return strings.TrimSpace(imageURLs[0])
}
