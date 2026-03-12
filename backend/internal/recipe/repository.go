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
SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(link, ''), COALESCE(image_url, ''),
       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
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
SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(link, ''), COALESCE(image_url, ''),
       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
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

	ingredientsJSON, stepsJSON, err := marshalParsedContent(item.ParsedContent)
	if err != nil {
		_ = tx.Rollback()
		return Recipe{}, err
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes
SET title = ?, ingredient = ?, link = ?, image_url = ?, meal_type = ?, status = ?, note = ?,
    ingredients_json = ?, steps_json = ?, updated_by = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		item.Title,
		nullableString(item.Ingredient),
		nullableString(item.Link),
		nullableString(item.ImageURL),
		item.MealType,
		item.Status,
		nullableString(item.Note),
		ingredientsJSON,
		stepsJSON,
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

type scanner interface {
	Scan(dest ...any) error
}

func scanRecipe(s scanner) (Recipe, error) {
	var (
		item            Recipe
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
		&item.MealType,
		&item.Status,
		&item.Note,
		&ingredientsJSON,
		&stepsJSON,
		&item.CreatedBy,
		&item.UpdatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return Recipe{}, err
	}

	parsedContent, err := unmarshalParsedContent(ingredientsJSON, stepsJSON)
	if err != nil {
		return Recipe{}, err
	}

	item.ParsedContent = parsedContent
	return item, nil
}

func insertRecipe(ctx context.Context, tx *sql.Tx, item Recipe) error {
	ingredientsJSON, stepsJSON, err := marshalParsedContent(item.ParsedContent)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO recipes (
id, kitchen_id, title, ingredient, link, image_url, meal_type, status, note,
ingredients_json, steps_json, created_by, updated_by, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID,
		item.KitchenID,
		item.Title,
		nullableString(item.Ingredient),
		nullableString(item.Link),
		nullableString(item.ImageURL),
		item.MealType,
		item.Status,
		nullableString(item.Note),
		ingredientsJSON,
		stepsJSON,
		item.CreatedBy,
		item.UpdatedBy,
		item.CreatedAt,
		item.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert recipe: %w", err)
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
