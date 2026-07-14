package recipe

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type scanner interface {
	Scan(dest ...any) error
}

func scanRecipe(s scanner) (Recipe, error) {
	var (
		item                Recipe
		imageURLsJSON       string
		imageMetaJSON       string
		ingredientsJSON     string
		stepsJSON           string
		parsedContentEdited int
	)

	err := s.Scan(
		&item.ID,
		&item.KitchenID,
		&item.Title,
		&item.TitleSource,
		&item.Ingredient,
		&item.Summary,
		&item.Link,
		&item.ImageURL,
		&imageURLsJSON,
		&imageMetaJSON,
		&item.FlowchartImageURL,
		&item.FlowchartProvider,
		&item.FlowchartModel,
		&item.FlowchartUpdatedAt,
		&item.FlowchartSourceHash,
		&item.FlowchartStatus,
		&item.FlowchartError,
		&item.FlowchartRequestedAt,
		&item.FlowchartFinishedAt,
		&item.MealType,
		&item.Status,
		&item.Note,
		&ingredientsJSON,
		&stepsJSON,
		&item.ParseStatus,
		&item.ParseSource,
		&item.ParseError,
		&item.ParseRequestedAt,
		&item.ParseFinishedAt,
		&item.ParseAttempts,
		&item.ParseNextAttemptAt,
		&item.ParseLastErrorType,
		&item.ParseProcessingStartedAt,
		&parsedContentEdited,
		&item.PinnedAt,
		&item.DoneAt,
		&item.CreatedBy,
		&item.UpdatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return Recipe{}, err
	}

	imageURLs, err := unmarshalImageURLs(imageURLsJSON)
	if err != nil {
		return Recipe{}, err
	}
	imageMetas, err := unmarshalImageMetas(imageMetaJSON)
	if err != nil {
		return Recipe{}, err
	}
	parsedContent, err := unmarshalParsedContent(ingredientsJSON, stepsJSON)
	if err != nil {
		return Recipe{}, err
	}

	if len(imageURLs) == 0 && strings.TrimSpace(item.ImageURL) != "" {
		imageURLs = []string{strings.TrimSpace(item.ImageURL)}
	}
	item.ImageMetas = normalizeRecipeImageMetas(imageURLs, imageMetas)
	item.ImageURLs = recipeImageURLsFromMetas(item.ImageMetas)
	item.ImageURL = firstImageURL(item.ImageURLs)
	item.TitleSource = normalizeTitleSource(item.TitleSource)
	item.ParsedContentEdited = parsedContentEdited != 0
	item.ParsedContent = normalizeParsedContent(parsedContent, item.MealType, item.Title, item.Ingredient)
	item.FlowchartStale = strings.TrimSpace(item.FlowchartImageURL) != "" && strings.TrimSpace(item.FlowchartSourceHash) != buildFlowchartSourceHash(item)
	return item, nil
}

func insertRecipe(ctx context.Context, tx *sql.Tx, item Recipe) error {
	imageMetas := normalizeRecipeImageMetas(item.ImageURLs, item.ImageMetas)
	imageURLs := recipeImageURLsFromMetas(imageMetas)
	imageURLsJSON, err := marshalImageURLs(imageURLs)
	if err != nil {
		return err
	}
	imageMetaJSON, err := marshalImageMetas(imageMetas)
	if err != nil {
		return err
	}

	ingredientsJSON, stepsJSON, err := marshalParsedContent(item.ParsedContent)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO recipes (
	id, kitchen_id, title, title_source, ingredient, summary, link, image_url, image_urls_json, image_meta_json, meal_type, status, note,
	ingredients_json, steps_json, flowchart_image_url, flowchart_provider, flowchart_model, flowchart_updated_at, flowchart_source_hash,
	flowchart_status, flowchart_error, flowchart_requested_at, flowchart_finished_at,
	parse_status, parse_source, parse_error, parse_requested_at, parse_finished_at,
	parse_attempts, parse_next_attempt_at, parse_last_error_type, parse_processing_started_at, parsed_content_edited,
	pinned_at, done_at, created_by, updated_by, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID,
		item.KitchenID,
		item.Title,
		normalizeTitleSource(item.TitleSource),
		nullableString(item.Ingredient),
		nonNullableTrimmedString(item.Summary),
		nullableString(item.Link),
		nullableString(firstImageURL(imageURLs)),
		imageURLsJSON,
		imageMetaJSON,
		item.MealType,
		item.Status,
		nullableString(item.Note),
		ingredientsJSON,
		stepsJSON,
		nonNullableTrimmedString(item.FlowchartImageURL),
		strings.TrimSpace(item.FlowchartProvider),
		strings.TrimSpace(item.FlowchartModel),
		nullableString(item.FlowchartUpdatedAt),
		strings.TrimSpace(item.FlowchartSourceHash),
		item.FlowchartStatus,
		strings.TrimSpace(item.FlowchartError),
		nullableString(item.FlowchartRequestedAt),
		nullableString(item.FlowchartFinishedAt),
		item.ParseStatus,
		item.ParseSource,
		strings.TrimSpace(item.ParseError),
		nullableString(item.ParseRequestedAt),
		nullableString(item.ParseFinishedAt),
		item.ParseAttempts,
		nonNullableTrimmedString(item.ParseNextAttemptAt),
		nonNullableTrimmedString(item.ParseLastErrorType),
		nonNullableTrimmedString(item.ParseProcessingStartedAt),
		item.ParsedContentEdited,
		nullableString(item.PinnedAt),
		nonNullableTrimmedString(item.DoneAt),
		item.CreatedBy,
		item.UpdatedBy,
		item.CreatedAt,
		item.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert recipe: %w", err)
	}

	return nil
}

func findRecipeByIDTx(ctx context.Context, tx *sql.Tx, recipeID string) (Recipe, error) {
	const query = `
	SELECT id, kitchen_id, title, COALESCE(title_source, 'manual'), COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parse_attempts, 0), COALESCE(parse_next_attempt_at, ''), COALESCE(parse_last_error_type, ''), COALESCE(parse_processing_started_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''), COALESCE(done_at, ''),
	       created_by, updated_by, created_at, updated_at
	FROM recipes
WHERE id = ? AND deleted_at IS NULL
LIMIT 1`

	row := tx.QueryRowContext(ctx, query, recipeID)
	item, err := scanRecipe(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Recipe{}, err
		}
		return Recipe{}, fmt.Errorf("find recipe by id in tx: %w", err)
	}
	return item, nil
}

func resolveRecipeDoneAt(currentDoneAt string, status string, touchedAt string) string {
	if status != "done" {
		return ""
	}
	if strings.TrimSpace(currentDoneAt) != "" {
		return currentDoneAt
	}
	return strings.TrimSpace(touchedAt)
}

func insertRecipeStatusEvent(ctx context.Context, tx *sql.Tx, kitchenID int64, recipeID string, fromStatus string, toStatus string, changedBy int64, changedAt string, source string) error {
	toStatus = strings.TrimSpace(toStatus)
	if toStatus == "" {
		return nil
	}
	if strings.TrimSpace(source) == "" {
		source = "api"
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO recipe_status_events (
  kitchen_id, recipe_id, from_status, to_status, changed_by, changed_at, source
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		kitchenID,
		recipeID,
		strings.TrimSpace(fromStatus),
		toStatus,
		changedBy,
		strings.TrimSpace(changedAt),
		strings.TrimSpace(source),
	); err != nil {
		return fmt.Errorf("insert recipe status event: %w", err)
	}
	return nil
}

func bumpKitchenUpdatedAt(ctx context.Context, tx *sql.Tx, kitchenID int64, updatedAt string) error {
	if _, err := tx.ExecContext(ctx, `UPDATE kitchens SET updated_at = ? WHERE id = ?`, updatedAt, kitchenID); err != nil {
		return fmt.Errorf("bump kitchen updated_at: %w", err)
	}

	return nil
}

func marshalParsedContent(content ParsedContent) (string, string, error) {
	mainIngredients := cleanLines(content.MainIngredients)
	secondaryIngredients := cleanLines(content.SecondaryIngredients)
	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 {
		mainIngredients, secondaryIngredients = splitIngredientLines(cleanLines(content.legacyIngredients))
	}

	ingredients, err := json.Marshal(struct {
		MainIngredients      []string `json:"mainIngredients,omitempty"`
		SecondaryIngredients []string `json:"secondaryIngredients,omitempty"`
	}{
		MainIngredients:      mainIngredients,
		SecondaryIngredients: secondaryIngredients,
	})
	if err != nil {
		return "", "", fmt.Errorf("marshal ingredients: %w", err)
	}

	stepsValue := cleanParsedSteps(content.Steps)
	if len(stepsValue) == 0 {
		stepsValue = buildParsedSteps(cleanLines(content.legacySteps))
	}

	steps, err := json.Marshal(stepsValue)
	if err != nil {
		return "", "", fmt.Errorf("marshal steps: %w", err)
	}

	return string(ingredients), string(steps), nil
}

func marshalImageURLs(imageURLs []string) (string, error) {
	if len(imageURLs) == 0 {
		return "[]", nil
	}

	encoded, err := json.Marshal(imageURLs)
	if err != nil {
		return "", fmt.Errorf("marshal image urls: %w", err)
	}

	return string(encoded), nil
}

func marshalImageMetas(imageMetas []RecipeImageMeta) (string, error) {
	imageMetas = normalizeRecipeImageMetas(recipeImageURLsFromMetas(imageMetas), imageMetas)
	if len(imageMetas) == 0 {
		return "[]", nil
	}

	encoded, err := json.Marshal(imageMetas)
	if err != nil {
		return "", fmt.Errorf("marshal image metas: %w", err)
	}

	return string(encoded), nil
}

func unmarshalImageURLs(imageURLsJSON string) ([]string, error) {
	if strings.TrimSpace(imageURLsJSON) == "" {
		return []string{}, nil
	}

	var imageURLs []string
	if err := json.Unmarshal([]byte(imageURLsJSON), &imageURLs); err != nil {
		return nil, fmt.Errorf("unmarshal image urls: %w", err)
	}

	return imageURLs, nil
}

func unmarshalImageMetas(imageMetaJSON string) ([]RecipeImageMeta, error) {
	if strings.TrimSpace(imageMetaJSON) == "" {
		return []RecipeImageMeta{}, nil
	}

	var imageMetas []RecipeImageMeta
	if err := json.Unmarshal([]byte(imageMetaJSON), &imageMetas); err != nil {
		return nil, fmt.Errorf("unmarshal image metas: %w", err)
	}

	return imageMetas, nil
}

func unmarshalParsedContent(ingredientsJSON, stepsJSON string) (ParsedContent, error) {
	content := ParsedContent{}
	if strings.TrimSpace(ingredientsJSON) != "" {
		var grouped struct {
			MainIngredients      []string `json:"mainIngredients"`
			SecondaryIngredients []string `json:"secondaryIngredients"`
		}
		if err := json.Unmarshal([]byte(ingredientsJSON), &grouped); err == nil {
			content.MainIngredients = grouped.MainIngredients
			content.SecondaryIngredients = grouped.SecondaryIngredients
		} else {
			if err := json.Unmarshal([]byte(ingredientsJSON), &content.legacyIngredients); err != nil {
				return ParsedContent{}, fmt.Errorf("unmarshal ingredients: %w", err)
			}
		}
	}

	if strings.TrimSpace(stepsJSON) != "" {
		if err := json.Unmarshal([]byte(stepsJSON), &content.Steps); err != nil {
			if err := json.Unmarshal([]byte(stepsJSON), &content.legacySteps); err != nil {
				return ParsedContent{}, fmt.Errorf("unmarshal steps: %w", err)
			}
		}
	}

	return content, nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nonNullableTrimmedString(value string) string {
	return strings.TrimSpace(value)
}

func truncateString(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}

	return string(runes[:maxRunes])
}

func resolveAutoParseImages(current Recipe, draft Recipe) (string, []string, []RecipeImageMeta) {
	currentMetas := recipeImageMetasFromItem(current)
	items := make([]RecipeImageMeta, 0, len(currentMetas)+len(draft.ImageURLs))
	for _, meta := range currentMetas {
		if normalizeRecipeImageSource(meta.SourceType) == RecipeImageSourceParsed {
			continue
		}
		items = append(items, meta)
	}

	sourceLink := strings.TrimSpace(current.Link)
	for _, imageURL := range recipeImageURLsFromItem(draft) {
		items = append(items, RecipeImageMeta{
			URL:        imageURL,
			SourceType: RecipeImageSourceParsed,
			OriginURL:  imageURL,
			SourceLink: sourceLink,
		})
	}

	imageMetas := dedupeRecipeImageMetas(items)
	imageURLs := recipeImageURLsFromMetas(imageMetas)
	return firstImageURL(imageURLs), imageURLs, imageMetas
}

func resolveAutoParseTitle(current Recipe, draft Recipe) string {
	currentTitle := strings.TrimSpace(current.Title)
	if normalizeTitleSource(current.TitleSource) != TitleSourcePlaceholder {
		return currentTitle
	}
	if draftTitle := strings.TrimSpace(draft.Title); draftTitle != "" {
		return draftTitle
	}
	return currentTitle
}

func resolveAutoParseTitleSource(current Recipe, draft Recipe) string {
	if normalizeTitleSource(current.TitleSource) == TitleSourcePlaceholder && strings.TrimSpace(draft.Title) != "" {
		return TitleSourceParsed
	}
	return normalizeTitleSource(current.TitleSource)
}

func mergeRecipeImageURLs(groups ...[]string) []string {
	items := make([]string, 0, maxRecipeImages)
	seen := make(map[string]struct{}, maxRecipeImages)
	for _, group := range groups {
		for _, value := range group {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			items = append(items, value)
			if len(items) >= maxRecipeImages {
				return items
			}
		}
	}

	return items
}

func cleanRecipeImageURLs(values []string) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

func imageSlicesEqual(left, right []string) bool {
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
