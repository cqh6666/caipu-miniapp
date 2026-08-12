package recipe

import (
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipecontent"
)

func normalizeParsedContent(content ParsedContent, mealType, title, ingredient string) ParsedContent {
	normalized := recipecontent.Normalize(content)
	mainIngredients := normalized.MainIngredients
	secondaryIngredients := normalized.SecondaryIngredients
	steps := normalized.Steps

	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 && len(steps) == 0 {
		return defaultParsedContent(mealType, title, ingredient)
	}

	fallback := defaultParsedContent(mealType, title, ingredient)
	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 {
		mainIngredients = append([]string{}, fallback.MainIngredients...)
		secondaryIngredients = append([]string{}, fallback.SecondaryIngredients...)
	}
	if len(steps) == 0 {
		steps = append([]ParsedStep{}, fallback.Steps...)
	}

	return ParsedContent{
		MainIngredients:      mainIngredients,
		SecondaryIngredients: secondaryIngredients,
		Steps:                steps,
	}
}

func defaultParsedContent(mealType, title, ingredient string) ParsedContent {
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
		MainIngredients: []string{
			mainIngredient + " 1份",
		},
		SecondaryIngredients: []string{
			mealLabel + "常用配菜 适量",
			"基础调味 适量",
		},
		Steps: []ParsedStep{
			{Title: "整理做法", Detail: "先整理这道菜的核心做法。"},
			{Title: "调整口味", Detail: "按自己的口味调整成容易复刻的版本。"},
			{Title: "补充记录", Detail: "做完以后补充口感和火候记录。"},
		},
	}
}

func legacyFrontendFallbackParsedContent(mealType, title, ingredient string) ParsedContent {
	return normalizeParsedContent(recipecontent.FromLegacy(
		legacyFrontendFallbackIngredientLines(mealType, title, ingredient),
		legacyFrontendFallbackStepLines(title),
	), mealType, title, ingredient)
}

func cleanLines(lines []string) []string {
	return recipecontent.CleanLines(lines)
}

func shouldQueueAutoParse(link string, content ParsedContent, mealType, title, ingredient string) bool {
	if !linkparse.SupportsAutoParseURL(link) {
		return false
	}

	return !hasUserProvidedParsedContent(content, mealType, title, ingredient)
}

func canGenerateFlowchartForRecipe(item Recipe) bool {
	if strings.TrimSpace(item.Title) == "" {
		return false
	}
	if !hasUserProvidedParsedContent(item.ParsedContent, item.MealType, item.Title, item.Ingredient) {
		return false
	}
	return len(cleanParsedSteps(item.ParsedContent.Steps)) >= 3
}

func hasMeaningfulParsedContent(content ParsedContent) bool {
	return len(parsedContentIngredientLines(content)) > 0 || len(parsedContentStepLines(content)) > 0
}

func hasUserProvidedParsedContent(content ParsedContent, mealType, title, ingredient string) bool {
	if !hasMeaningfulParsedContent(content) {
		return false
	}

	requestedIngredients := parsedContentIngredientLines(content)
	requestedSteps := parsedContentStepLines(content)

	for _, fallback := range []struct {
		ingredients []string
		steps       []string
	}{
		{
			ingredients: parsedContentIngredientLines(defaultParsedContent(mealType, title, ingredient)),
			steps:       parsedContentStepLines(defaultParsedContent(mealType, title, ingredient)),
		},
		{
			ingredients: legacyFrontendFallbackIngredientLines(mealType, title, ingredient),
			steps: recipecontent.StepLines(recipecontent.FromLegacy(
				nil,
				legacyFrontendFallbackStepLines(title),
			)),
		},
	} {
		if stringSlicesEqual(requestedIngredients, fallback.ingredients) && stringSlicesEqual(requestedSteps, fallback.steps) {
			return false
		}
	}

	return true
}

func legacyFrontendFallbackIngredientLines(mealType, title, ingredient string) []string {
	return parsedContentIngredientLines(defaultParsedContent(mealType, title, ingredient))
}

func legacyFrontendFallbackStepLines(title string) []string {
	titleLabel := strings.TrimSpace(title)
	if titleLabel == "" {
		titleLabel = "这道菜"
	}

	return []string{
		"先从链接里抓出 " + titleLabel + " 的核心做法。",
		"按自己的口味整理成容易复刻的家常版本。",
		"做完以后回来补充口感、火候和踩坑点。",
	}
}

func parsedContentIngredientLines(content ParsedContent) []string {
	return recipecontent.IngredientLines(content)
}

func parsedContentStepLines(content ParsedContent) []string {
	return recipecontent.StepLines(content)
}

func splitIngredientLines(lines []string) ([]string, []string) {
	return recipecontent.SplitIngredientLines(lines)
}

func ingredientLabelFromLine(line string) string {
	return recipecontent.IngredientLabel(line)
}

func cleanParsedSteps(steps []ParsedStep) []ParsedStep {
	return recipecontent.CleanSteps(steps)
}

func buildParsedSteps(lines []string) []ParsedStep {
	return recipecontent.BuildSteps(lines)
}

func compactParsedSteps(steps []ParsedStep) []ParsedStep {
	return recipecontent.CompactSteps(steps)
}

func inferParsedStepTitle(detail string, index int) string {
	return recipecontent.InferStepTitle(detail, index)
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}

func parsedContentSlicesEqual(left, right ParsedContent) bool {
	return recipecontent.Equal(left, right)
}

func isAllowedMealType(value string) bool {
	_, ok := allowedMealTypes[value]
	return ok
}

func isAllowedStatus(value string) bool {
	_, ok := allowedStatuses[value]
	return ok
}
