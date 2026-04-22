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
	SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''),
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

	query += " ORDER BY CASE WHEN COALESCE(pinned_at, '') = '' THEN 1 ELSE 0 END ASC, pinned_at DESC, updated_at DESC, id DESC"

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
	SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''),
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

	imageMetas := normalizeRecipeImageMetas(item.ImageURLs, item.ImageMetas)
	imageURLs := recipeImageURLsFromMetas(imageMetas)
	imageURLsJSON, err := marshalImageURLs(imageURLs)
	if err != nil {
		_ = tx.Rollback()
		return Recipe{}, err
	}
	imageMetaJSON, err := marshalImageMetas(imageMetas)
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
	SET title = ?, ingredient = ?, summary = ?, link = ?, image_url = ?, image_urls_json = ?, image_meta_json = ?, meal_type = ?, status = ?, note = ?,
	    ingredients_json = ?, steps_json = ?, flowchart_image_url = ?, flowchart_provider = ?, flowchart_model = ?, flowchart_updated_at = ?, flowchart_source_hash = ?,
	    flowchart_status = ?, flowchart_error = ?, flowchart_requested_at = ?, flowchart_finished_at = ?,
	    parse_status = ?, parse_source = ?, parse_error = ?,
	    parse_requested_at = ?, parse_finished_at = ?, parsed_content_edited = ?, pinned_at = ?, updated_by = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		item.Title,
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
		item.ParsedContentEdited,
		nullableString(item.PinnedAt),
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

func (r *Repository) QueueFlowchart(ctx context.Context, recipeID, requestedAt string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
SET flowchart_status = ?, flowchart_error = '', flowchart_requested_at = ?, flowchart_finished_at = NULL
WHERE id = ? AND deleted_at IS NULL`,
		FlowchartStatusPending,
		nullableString(requestedAt),
		recipeID,
	)
	if err != nil {
		return fmt.Errorf("queue recipe flowchart: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read queue flowchart rows: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, recipeID string, kitchenID int64, status string, updatedBy int64, touchedAt string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update status tx: %w", err)
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes SET status = ?, updated_by = ? WHERE id = ? AND deleted_at IS NULL`,
		status,
		updatedBy,
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

	if err := bumpKitchenUpdatedAt(ctx, tx, kitchenID, touchedAt); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update recipe status: %w", err)
	}

	return nil
}

func (r *Repository) UpdatePinned(ctx context.Context, recipeID string, kitchenID int64, pinned bool, updatedBy int64, touchedAt string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update pinned tx: %w", err)
	}

	var pinnedAtValue any
	if pinned {
		pinnedAtValue = touchedAt
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes SET pinned_at = ?, updated_by = ? WHERE id = ? AND deleted_at IS NULL`,
		pinnedAtValue,
		updatedBy,
		recipeID,
	)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("update recipe pinned state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("read pinned rows: %w", err)
	}
	if rowsAffected == 0 {
		_ = tx.Rollback()
		return sql.ErrNoRows
	}

	if err := bumpKitchenUpdatedAt(ctx, tx, kitchenID, touchedAt); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update pinned state: %w", err)
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
	SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''),
	       created_by, updated_by, created_at, updated_at
	FROM recipes
	WHERE deleted_at IS NULL AND parse_status = ?
ORDER BY COALESCE(NULLIF(parse_requested_at, ''), created_at) ASC, id ASC
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

func (r *Repository) CountPendingAutoParseAhead(ctx context.Context, item Recipe) (int, error) {
	cursor := strings.TrimSpace(item.ParseRequestedAt)
	if cursor == "" {
		cursor = strings.TrimSpace(item.CreatedAt)
	}

	const query = `
	SELECT COUNT(1)
	FROM recipes
	WHERE deleted_at IS NULL
	  AND parse_status = ?
	  AND (
	    COALESCE(NULLIF(parse_requested_at, ''), created_at) < ?
	    OR (COALESCE(NULLIF(parse_requested_at, ''), created_at) = ? AND id < ?)
	  )`

	var count int
	if err := r.db.QueryRowContext(ctx, query, ParseStatusPending, cursor, cursor, item.ID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count pending auto parse ahead: %w", err)
	}
	return count, nil
}

func (r *Repository) CountProcessingAutoParse(ctx context.Context) (int, error) {
	const query = `
	SELECT COUNT(1)
	FROM recipes
	WHERE deleted_at IS NULL AND parse_status = ?`

	var count int
	if err := r.db.QueryRowContext(ctx, query, ParseStatusProcessing).Scan(&count); err != nil {
		return 0, fmt.Errorf("count processing auto parse jobs: %w", err)
	}
	return count, nil
}

func (r *Repository) ListLegacyAutoParseCandidates(ctx context.Context, limit int) ([]Recipe, error) {
	const query = `
	SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''),
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
	SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''),
	       created_by, updated_by, created_at, updated_at
	FROM recipes
WHERE deleted_at IS NULL
  AND (
    (
      LOWER(TRIM(COALESCE(image_url, ''))) LIKE 'http%'
      AND LOWER(TRIM(COALESCE(image_url, ''))) NOT LIKE '%/uploads/%'
    )
    OR EXISTS (
      SELECT 1
      FROM json_each(COALESCE(NULLIF(image_urls_json, ''), '[]'))
      WHERE LOWER(TRIM(COALESCE(json_each.value, ''))) LIKE 'http%'
        AND LOWER(TRIM(COALESCE(json_each.value, ''))) NOT LIKE '%/uploads/%'
    )
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

func (r *Repository) ListPendingFlowcharts(ctx context.Context, limit int) ([]Recipe, error) {
	const query = `
	SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''),
	       created_by, updated_by, created_at, updated_at
	FROM recipes
	WHERE deleted_at IS NULL AND flowchart_status = ?
ORDER BY COALESCE(NULLIF(flowchart_requested_at, ''), updated_at, created_at) ASC, id ASC
LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, FlowchartStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending flowchart jobs: %w", err)
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
		return nil, fmt.Errorf("iterate pending flowchart jobs: %w", err)
	}

	return items, nil
}

func (r *Repository) CountPendingFlowchartAhead(ctx context.Context, item Recipe) (int, error) {
	cursor := strings.TrimSpace(item.FlowchartRequestedAt)
	if cursor == "" {
		cursor = strings.TrimSpace(item.UpdatedAt)
	}
	if cursor == "" {
		cursor = strings.TrimSpace(item.CreatedAt)
	}

	const query = `
	SELECT COUNT(1)
	FROM recipes
	WHERE deleted_at IS NULL
	  AND flowchart_status = ?
	  AND (
	    COALESCE(NULLIF(flowchart_requested_at, ''), updated_at, created_at) < ?
	    OR (COALESCE(NULLIF(flowchart_requested_at, ''), updated_at, created_at) = ? AND id < ?)
	  )`

	var count int
	if err := r.db.QueryRowContext(ctx, query, FlowchartStatusPending, cursor, cursor, item.ID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count pending flowchart jobs ahead: %w", err)
	}
	return count, nil
}

func (r *Repository) CountProcessingFlowcharts(ctx context.Context) (int, error) {
	const query = `
	SELECT COUNT(1)
	FROM recipes
	WHERE deleted_at IS NULL AND flowchart_status = ?`

	var count int
	if err := r.db.QueryRowContext(ctx, query, FlowchartStatusProcessing).Scan(&count); err != nil {
		return 0, fmt.Errorf("count processing flowchart jobs: %w", err)
	}
	return count, nil
}

func (r *Repository) ListAutoFlowchartCandidates(ctx context.Context, limit int) ([]Recipe, error) {
	const query = `
	SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''),
	       created_by, updated_by, created_at, updated_at
	FROM recipes
WHERE deleted_at IS NULL
  AND COALESCE(flowchart_status, '') IN (?, ?)
  AND COALESCE(TRIM(flowchart_image_url), '') = ''
  AND COALESCE(parse_status, '') NOT IN (?, ?)
ORDER BY
  CASE COALESCE(flowchart_status, '')
    WHEN ? THEN 0
    WHEN ? THEN 1
    ELSE 2
  END ASC,
  CASE
    WHEN COALESCE(flowchart_status, '') = ? THEN COALESCE(NULLIF(flowchart_finished_at, ''), NULLIF(flowchart_requested_at, ''), updated_at, created_at)
    ELSE COALESCE(NULLIF(created_at, ''), updated_at)
  END ASC,
  id ASC
LIMIT ?`

	rows, err := r.db.QueryContext(
		ctx,
		query,
		FlowchartStatusIdle,
		FlowchartStatusFailed,
		ParseStatusPending,
		ParseStatusProcessing,
		FlowchartStatusIdle,
		FlowchartStatusFailed,
		FlowchartStatusFailed,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list auto flowchart candidates: %w", err)
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
		return nil, fmt.Errorf("iterate auto flowchart candidates: %w", err)
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

func (r *Repository) MarkFlowchartProcessing(ctx context.Context, recipeID string) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
SET flowchart_status = ?, flowchart_error = ''
WHERE id = ? AND deleted_at IS NULL AND flowchart_status = ?`,
		FlowchartStatusProcessing,
		recipeID,
		FlowchartStatusPending,
	)
	if err != nil {
		return false, fmt.Errorf("mark recipe flowchart processing: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read flowchart processing rows: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) MarkAutoFlowchartPending(ctx context.Context, recipeID, requestedAt string) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
SET flowchart_status = ?, flowchart_error = '', flowchart_requested_at = ?, flowchart_finished_at = NULL
WHERE id = ? AND deleted_at IS NULL
  AND COALESCE(flowchart_status, '') IN (?, ?)
  AND COALESCE(TRIM(flowchart_image_url), '') = ''`,
		FlowchartStatusPending,
		nullableString(requestedAt),
		recipeID,
		FlowchartStatusIdle,
		FlowchartStatusFailed,
	)
	if err != nil {
		return false, fmt.Errorf("mark auto recipe flowchart pending: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read auto flowchart pending rows: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r *Repository) RequeueStaleFlowcharts(ctx context.Context, staleBefore, requestedAt string) (int64, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
SET flowchart_status = ?, flowchart_error = '', flowchart_requested_at = ?, flowchart_finished_at = NULL
WHERE deleted_at IS NULL
  AND flowchart_status = ?
  AND datetime(COALESCE(NULLIF(flowchart_requested_at, ''), NULLIF(updated_at, ''), NULLIF(created_at, ''))) <= datetime(?)`,
		FlowchartStatusPending,
		nullableString(requestedAt),
		FlowchartStatusProcessing,
		staleBefore,
	)
	if err != nil {
		return 0, fmt.Errorf("requeue stale recipe flowcharts: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read stale flowchart rows: %w", err)
	}

	return rowsAffected, nil
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

func (r *Repository) MarkFlowchartFailed(ctx context.Context, recipeID, flowchartError, finishedAt string) error {
	if _, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
SET flowchart_status = ?, flowchart_error = ?, flowchart_finished_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		FlowchartStatusFailed,
		truncateString(strings.TrimSpace(flowchartError), 300),
		finishedAt,
		recipeID,
	); err != nil {
		return fmt.Errorf("mark recipe flowchart failed: %w", err)
	}

	return nil
}

func (r *Repository) ApplyAutoParseResult(ctx context.Context, recipeID, parseSource, parseError, finishedAt string, draft Recipe) error {
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
	imageURLValue, imageURLsValue, imageMetasValue := resolveAutoParseImages(current, draft)

	imageURLsJSON, err := marshalImageURLs(imageURLsValue)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	imageMetaJSON, err := marshalImageMetas(imageMetasValue)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	ingredientValue := current.Ingredient
	if strings.TrimSpace(ingredientValue) == "" {
		ingredientValue = draft.Ingredient
	}
	summaryValue := strings.TrimSpace(draft.Summary)

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes
SET ingredient = ?, summary = ?, image_url = ?, image_urls_json = ?, image_meta_json = ?, ingredients_json = ?, steps_json = ?,
    parse_status = ?, parse_source = ?, parse_error = ?, parse_finished_at = ?, parsed_content_edited = 0, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		nullableString(ingredientValue),
		nonNullableTrimmedString(summaryValue),
		nullableString(imageURLValue),
		imageURLsJSON,
		imageMetaJSON,
		ingredientsJSON,
		stepsJSON,
		ParseStatusDone,
		parseSource,
		strings.TrimSpace(parseError),
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

func (r *Repository) ApplyFlowchartResult(ctx context.Context, recipeID, imageURL, provider, model, sourceHash, finishedAt string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
SET flowchart_image_url = ?, flowchart_provider = ?, flowchart_model = ?, flowchart_updated_at = ?, flowchart_source_hash = ?,
    flowchart_status = ?, flowchart_error = '', flowchart_finished_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		nullableString(imageURL),
		strings.TrimSpace(provider),
		strings.TrimSpace(model),
		nullableString(finishedAt),
		strings.TrimSpace(sourceHash),
		FlowchartStatusDone,
		finishedAt,
		recipeID,
	)
	if err != nil {
		return fmt.Errorf("apply recipe flowchart result: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read flowchart result rows: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *Repository) ApplyMirroredImages(ctx context.Context, recipeID string, oldImages []string, newMetas []RecipeImageMeta, updatedAt string) (bool, error) {
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
	currentImages := recipeImageURLsFromItem(current)
	if !imageSlicesEqual(currentImages, expectedOld) {
		_ = tx.Rollback()
		return false, nil
	}

	nextMetas := normalizeRecipeImageMetas(recipeImageURLsFromMetas(newMetas), newMetas)
	nextImages := recipeImageURLsFromMetas(nextMetas)
	nextImageURL := firstImageURL(nextImages)
	imageURLsJSON, err := marshalImageURLs(nextImages)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	imageMetaJSON, err := marshalImageMetas(nextMetas)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes
SET image_url = ?, image_urls_json = ?, image_meta_json = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		nullableString(nextImageURL),
		imageURLsJSON,
		imageMetaJSON,
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
		&parsedContentEdited,
		&item.PinnedAt,
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
	id, kitchen_id, title, ingredient, summary, link, image_url, image_urls_json, image_meta_json, meal_type, status, note,
	ingredients_json, steps_json, flowchart_image_url, flowchart_provider, flowchart_model, flowchart_updated_at, flowchart_source_hash,
	flowchart_status, flowchart_error, flowchart_requested_at, flowchart_finished_at,
	parse_status, parse_source, parse_error, parse_requested_at, parse_finished_at, parsed_content_edited,
	pinned_at, created_by, updated_by, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID,
		item.KitchenID,
		item.Title,
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
		item.ParsedContentEdited,
		nullableString(item.PinnedAt),
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
	SELECT id, kitchen_id, title, COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''),
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
