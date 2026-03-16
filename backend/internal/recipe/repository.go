package recipe

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByKitchenID(ctx context.Context, kitchenID int64, filter ListFilter) ([]Recipe, error) {
	query := `
SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'),
       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''),
       created_by, updated_by, created_at, updated_at
FROM recipes
WHERE kitchen_id = ? AND deleted_at IS NULL
`

	args := []any{kitchenID}

	if filter.MealType != "" {
		query += " AND meal_type = ?"
		args = append(args, filter.MealType)
	}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	if filter.Keyword != "" {
		query += " AND (title LIKE ? OR ingredient LIKE ? OR note LIKE ? OR link LIKE ?)"
		keyword := "%" + filter.Keyword + "%"
		args = append(args, keyword, keyword, keyword, keyword)
	}

	query += " ORDER BY updated_at DESC, id DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list recipes by kitchen: %w", err)
	}
	defer rows.Close()

	items := make([]Recipe, 0)
	for rows.Next() {
		item, err := scanRecipe(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipes: %w", err)
	}

	return items, nil
}

func (r *Repository) FindByID(ctx context.Context, recipeID string) (Recipe, error) {
	const query = `
SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'),
       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''),
       created_by, updated_by, created_at, updated_at
FROM recipes
WHERE id = ? AND deleted_at IS NULL
LIMIT 1
`

	row := r.db.QueryRowContext(ctx, query, recipeID)
	item, err := scanRecipe(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, err
	}
	if err != nil {
		return Recipe{}, fmt.Errorf("find recipe by id: %w", err)
	}

	return item, nil
}

func (r *Repository) Create(ctx context.Context, item Recipe) (Recipe, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Recipe{}, fmt.Errorf("begin create recipe tx: %w", err)
	}

	if err := insertRecipe(ctx, tx, item); err != nil {
		_ = tx.Rollback()
		return Recipe{}, err
	}

	if err := bumpKitchenUpdatedAt(ctx, tx, item.KitchenID, item.UpdatedAt); err != nil {
		_ = tx.Rollback()
		return Recipe{}, err
	}

	if err := tx.Commit(); err != nil {
		return Recipe{}, fmt.Errorf("commit create recipe: %w", err)
	}

	return item, nil
}

func (r *Repository) Update(ctx context.Context, item Recipe) (Recipe, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Recipe{}, fmt.Errorf("begin update recipe tx: %w", err)
	}

	imageURLsJSON, err := marshalImageURLs(item.ImageURLs)
	if err != nil {
		_ = tx.Rollback()
		return Recipe{}, err
	}

	ingredientsJSON, stepsJSON, err := marshalParsedContent(item.ParsedContent)
	if err != nil {
		_ = tx.Rollback()
		return Recipe{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes
SET title = ?, ingredient = ?, link = ?, image_url = ?, image_urls_json = ?, meal_type = ?, status = ?, note = ?,
    ingredients_json = ?, steps_json = ?, parse_status = ?, parse_source = ?, parse_error = ?,
    parse_requested_at = ?, parse_finished_at = ?, updated_by = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		item.Title,
		nullableString(item.Ingredient),
		nullableString(item.Link),
		nullableString(item.ImageURL),
		imageURLsJSON,
		item.MealType,
		item.Status,
		nullableString(item.Note),
		ingredientsJSON,
		stepsJSON,
		item.ParseStatus,
		item.ParseSource,
		strings.TrimSpace(item.ParseError),
		nullableString(item.ParseRequestedAt),
		nullableString(item.ParseFinishedAt),
		item.UpdatedBy,
		item.UpdatedAt,
		item.ID,
	)
	if err != nil {
		_ = tx.Rollback()
		return Recipe{}, fmt.Errorf("update recipe: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return Recipe{}, fmt.Errorf("read update rows: %w", err)
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return Recipe{}, sql.ErrNoRows
	}

	if err := bumpKitchenUpdatedAt(ctx, tx, item.KitchenID, item.UpdatedAt); err != nil {
		_ = tx.Rollback()
		return Recipe{}, err
	}

	if err := tx.Commit(); err != nil {
		return Recipe{}, fmt.Errorf("commit update recipe: %w", err)
	}

	return item, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, recipeID string, kitchenID int64, status string, updatedBy int64, updatedAt string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update status tx: %w", err)
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes SET status = ?, updated_by = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		status,
		updatedBy,
		updatedAt,
		recipeID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("update recipe status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("read status rows: %w", err)
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return sql.ErrNoRows
	}

	if err := bumpKitchenUpdatedAt(ctx, tx, kitchenID, updatedAt); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update recipe status: %w", err)
	}

	return nil
}

func (r *Repository) SoftDelete(ctx context.Context, recipeID string, kitchenID int64, deletedBy int64, deletedAt string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete recipe tx: %w", err)
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes
SET deleted_at = ?, updated_by = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		deletedAt,
		deletedBy,
		deletedAt,
		recipeID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("soft delete recipe: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("read delete rows: %w", err)
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return sql.ErrNoRows
	}

	if err := bumpKitchenUpdatedAt(ctx, tx, kitchenID, deletedAt); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete recipe: %w", err)
	}

	return nil
}

func (r *Repository) ListPendingAutoParse(ctx context.Context, limit int) ([]Recipe, error) {
	const query = `
SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'),
       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''),
       created_by, updated_by, created_at, updated_at
FROM recipes
WHERE deleted_at IS NULL AND parse_status = ?
ORDER BY COALESCE(parse_requested_at, created_at) ASC, id ASC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, ParseStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending auto parse recipes: %w", err)
	}
	defer rows.Close()

	items := make([]Recipe, 0, limit)
	for rows.Next() {
		item, err := scanRecipe(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending auto parse recipes: %w", err)
	}

	return items, nil
}

func (r *Repository) ListLegacyAutoParseCandidates(ctx context.Context, limit int) ([]Recipe, error) {
	const query = `
SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'),
       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''),
       created_by, updated_by, created_at, updated_at
FROM recipes
WHERE deleted_at IS NULL
  AND COALESCE(parse_status, '') = ''
  AND (
    instr(lower(COALESCE(link, '')), 'bilibili.com') > 0
    OR instr(lower(COALESCE(link, '')), 'b23.tv') > 0
    OR instr(lower(COALESCE(link, '')), 'bili2233.cn') > 0
    OR instr(lower(COALESCE(link, '')), 'xiaohongshu.com') > 0
    OR instr(lower(COALESCE(link, '')), 'xhslink.com') > 0
  )
ORDER BY created_at ASC, id ASC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list legacy auto parse recipes: %w", err)
	}
	defer rows.Close()

	items := make([]Recipe, 0, limit)
	for rows.Next() {
		item, err := scanRecipe(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy auto parse recipes: %w", err)
	}

	return items, nil
}

func (r *Repository) ListImageMirrorCandidates(ctx context.Context, limit int) ([]Recipe, error) {
	const query = `
SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'),
       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''),
       created_by, updated_by, created_at, updated_at
FROM recipes
WHERE deleted_at IS NULL
  AND (
    COALESCE(image_urls_json, '[]') <> '[]'
    OR COALESCE(TRIM(image_url), '') <> ''
  )
ORDER BY updated_at ASC, id ASC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list image mirror candidates: %w", err)
	}
	defer rows.Close()

	items := make([]Recipe, 0, limit)
	for rows.Next() {
		item, err := scanRecipe(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate image mirror candidates: %w", err)
	}

	return items, nil
}

func (r *Repository) MarkAutoParseProcessing(ctx context.Context, recipeID, parseSource string) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
SET parse_status = ?, parse_source = ?, parse_error = ''
WHERE id = ? AND deleted_at IS NULL AND parse_status = ?`,
		ParseStatusProcessing,
		parseSource,
		recipeID,
		ParseStatusPending,
	)
	if err != nil {
		return false, fmt.Errorf("mark recipe auto parse processing: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read auto parse processing rows: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) MarkAutoParsePending(ctx context.Context, recipeID, parseSource, requestedAt string) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
SET parse_status = ?, parse_source = ?, parse_error = '', parse_requested_at = ?, parse_finished_at = NULL, updated_at = ?
WHERE id = ? AND deleted_at IS NULL AND COALESCE(parse_status, '') = ''`,
		ParseStatusPending,
		parseSource,
		requestedAt,
		requestedAt,
		recipeID,
	)
	if err != nil {
		return false, fmt.Errorf("mark recipe auto parse pending: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read auto parse pending rows: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) MarkAutoParseFailed(ctx context.Context, recipeID, parseSource, parseError, finishedAt string) error {
	if _, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
SET parse_status = ?, parse_source = ?, parse_error = ?, parse_finished_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		ParseStatusFailed,
		parseSource,
		truncateString(strings.TrimSpace(parseError), 300),
		finishedAt,
		recipeID,
	); err != nil {
		return fmt.Errorf("mark recipe auto parse failed: %w", err)
	}

	return nil
}

func (r *Repository) ApplyAutoParseResult(ctx context.Context, recipeID, parseSource, finishedAt string, draft Recipe) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin apply auto parse tx: %w", err)
	}

	current, err := findRecipeByIDTx(ctx, tx, recipeID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	ingredientsJSON, stepsJSON, err := marshalParsedContent(draft.ParsedContent)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	imageURLValue, imageURLsValue := resolveAutoParseImages(current, draft)

	imageURLsJSON, err := marshalImageURLs(imageURLsValue)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	ingredientValue := current.Ingredient
	if strings.TrimSpace(ingredientValue) == "" {
		ingredientValue = draft.Ingredient
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes
SET ingredient = ?, image_url = ?, image_urls_json = ?, ingredients_json = ?, steps_json = ?,
    parse_status = ?, parse_source = ?, parse_error = '', parse_finished_at = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		nullableString(ingredientValue),
		nullableString(imageURLValue),
		imageURLsJSON,
		ingredientsJSON,
		stepsJSON,
		ParseStatusDone,
		parseSource,
		finishedAt,
		finishedAt,
		recipeID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply recipe auto parse result: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("read auto parse result rows: %w", err)
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return sql.ErrNoRows
	}

	if err := bumpKitchenUpdatedAt(ctx, tx, current.KitchenID, finishedAt); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit apply auto parse result: %w", err)
	}

	return nil
}

func (r *Repository) RequeueAutoParse(ctx context.Context, recipeID string, userID int64, parseSource, requestedAt string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin requeue auto parse tx: %w", err)
	}

	current, err := findRecipeByIDTx(ctx, tx, recipeID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes
SET parse_status = ?, parse_source = ?, parse_error = '', parse_requested_at = ?, parse_finished_at = NULL,
    updated_by = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		ParseStatusPending,
		parseSource,
		requestedAt,
		userID,
		requestedAt,
		recipeID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("requeue recipe auto parse: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("read requeue auto parse rows: %w", err)
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return sql.ErrNoRows
	}

	if err := bumpKitchenUpdatedAt(ctx, tx, current.KitchenID, requestedAt); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit requeue auto parse: %w", err)
	}

	return nil
}

func (r *Repository) ApplyMirroredImages(ctx context.Context, recipeID string, oldImages, newImages []string, updatedAt string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin apply mirrored images tx: %w", err)
	}

	current, err := findRecipeByIDTx(ctx, tx, recipeID)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}

	expectedOld := cleanRecipeImageURLs(oldImages)
	currentImages := cleanRecipeImageURLs(current.ImageURLs)
	if len(currentImages) == 0 && strings.TrimSpace(current.ImageURL) != "" {
		currentImages = []string{strings.TrimSpace(current.ImageURL)}
	}
	if !imageSlicesEqual(currentImages, expectedOld) {
		_ = tx.Rollback()
		return false, nil
	}

	nextImages := cleanRecipeImageURLs(newImages)
	nextImageURL := firstImageURL(nextImages)
	imageURLsJSON, err := marshalImageURLs(nextImages)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes
SET image_url = ?, image_urls_json = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		nullableString(nextImageURL),
		imageURLsJSON,
		updatedAt,
		recipeID,
	)
	if err != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("apply mirrored recipe images: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("read mirrored image rows: %w", err)
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return false, sql.ErrNoRows
	}

	if err := bumpKitchenUpdatedAt(ctx, tx, current.KitchenID, updatedAt); err != nil {
		_ = tx.Rollback()
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit mirrored recipe images: %w", err)
	}

	return true, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRecipe(s scanner) (Recipe, error) {
	var (
		item            Recipe
		imageURLsJSON   string
		ingredientsJSON string
		stepsJSON       string
	)

	err := s.Scan(
		&item.ID,
		&item.KitchenID,
		&item.Title,
		&item.Ingredient,
		&item.Link,
		&item.ImageURL,
		&imageURLsJSON,
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
	parsedContent, err := unmarshalParsedContent(ingredientsJSON, stepsJSON)
	if err != nil {
		return Recipe{}, err
	}

	item.ImageURLs = imageURLs
	if len(item.ImageURLs) == 0 && strings.TrimSpace(item.ImageURL) != "" {
		item.ImageURLs = []string{strings.TrimSpace(item.ImageURL)}
	}
	if strings.TrimSpace(item.ImageURL) == "" {
		item.ImageURL = firstImageURL(imageURLs)
	}
	item.ParsedContent = parsedContent
	return item, nil
}

func insertRecipe(ctx context.Context, tx *sql.Tx, item Recipe) error {
	imageURLsJSON, err := marshalImageURLs(item.ImageURLs)
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
id, kitchen_id, title, ingredient, link, image_url, image_urls_json, meal_type, status, note,
ingredients_json, steps_json, parse_status, parse_source, parse_error, parse_requested_at, parse_finished_at,
created_by, updated_by, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID,
		item.KitchenID,
		item.Title,
		nullableString(item.Ingredient),
		nullableString(item.Link),
		nullableString(item.ImageURL),
		imageURLsJSON,
		item.MealType,
		item.Status,
		nullableString(item.Note),
		ingredientsJSON,
		stepsJSON,
		item.ParseStatus,
		item.ParseSource,
		strings.TrimSpace(item.ParseError),
		nullableString(item.ParseRequestedAt),
		nullableString(item.ParseFinishedAt),
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
SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'),
       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''),
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

func bumpKitchenUpdatedAt(ctx context.Context, tx *sql.Tx, kitchenID int64, updatedAt string) error {
	if _, err := tx.ExecContext(ctx, `UPDATE kitchens SET updated_at = ? WHERE id = ?`, updatedAt, kitchenID); err != nil {
		return fmt.Errorf("bump kitchen updated_at: %w", err)
	}

	return nil
}

func marshalParsedContent(content ParsedContent) (string, string, error) {
	ingredients, err := json.Marshal(content.Ingredients)
	if err != nil {
		return "", "", fmt.Errorf("marshal ingredients: %w", err)
	}

	steps, err := json.Marshal(content.Steps)
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

func unmarshalParsedContent(ingredientsJSON, stepsJSON string) (ParsedContent, error) {
	content := ParsedContent{
		Ingredients: []string{},
		Steps:       []string{},
	}

	if strings.TrimSpace(ingredientsJSON) != "" {
		if err := json.Unmarshal([]byte(ingredientsJSON), &content.Ingredients); err != nil {
			return ParsedContent{}, fmt.Errorf("unmarshal ingredients: %w", err)
		}
	}

	if strings.TrimSpace(stepsJSON) != "" {
		if err := json.Unmarshal([]byte(stepsJSON), &content.Steps); err != nil {
			return ParsedContent{}, fmt.Errorf("unmarshal steps: %w", err)
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

func resolveAutoParseImages(current Recipe, draft Recipe) (string, []string) {
	currentImageURL := strings.TrimSpace(current.ImageURL)
	currentImageURLs := current.ImageURLs
	if len(currentImageURLs) > 0 || currentImageURL != "" {
		if len(currentImageURLs) == 0 && currentImageURL != "" {
			return currentImageURL, []string{currentImageURL}
		}
		return currentImageURL, currentImageURLs
	}

	draftImageURLs := cleanRecipeImageURLs(append(draft.ImageURLs, strings.TrimSpace(draft.ImageURL)))
	return firstImageURL(draftImageURLs), draftImageURLs
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
