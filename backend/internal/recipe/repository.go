package recipe

import (
	"context"
	"database/sql"
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
	SELECT id, kitchen_id, title, COALESCE(title_source, 'manual'), COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parse_attempts, 0), COALESCE(parse_next_attempt_at, ''), COALESCE(parse_last_error_type, ''), COALESCE(parse_processing_started_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''), COALESCE(done_at, ''),
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

	if filter.TitleKeyword != "" {
		query += " AND title LIKE ?"
		args = append(args, "%"+filter.TitleKeyword+"%")
	}

	if filter.IngredientKeyword != "" {
		query += " AND (ingredient LIKE ? OR ingredients_json LIKE ?)"
		keyword := "%" + filter.IngredientKeyword + "%"
		args = append(args, keyword, keyword)
	}

	if filter.TitleOrIngredientKeyword != "" {
		query += " AND (title LIKE ? OR ingredient LIKE ? OR ingredients_json LIKE ?)"
		keyword := "%" + filter.TitleOrIngredientKeyword + "%"
		args = append(args, keyword, keyword, keyword)
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
	SELECT id, kitchen_id, title, COALESCE(title_source, 'manual'), COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parse_attempts, 0), COALESCE(parse_next_attempt_at, ''), COALESCE(parse_last_error_type, ''), COALESCE(parse_processing_started_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''), COALESCE(done_at, ''),
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

	item.DoneAt = resolveRecipeDoneAt("", item.Status, item.CreatedAt)

	if err := insertRecipe(ctx, tx, item); err != nil {
		_ = tx.Rollback()
		return Recipe{}, err
	}

	if err := insertRecipeStatusEvent(ctx, tx, item.KitchenID, item.ID, "", item.Status, item.CreatedBy, item.CreatedAt, "api"); err != nil {
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

	current, err := findRecipeByIDTx(ctx, tx, item.ID)
	if err != nil {
		_ = tx.Rollback()
		return Recipe{}, err
	}
	item.DoneAt = resolveRecipeDoneAt(current.DoneAt, item.Status, item.UpdatedAt)

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
	SET title = ?, title_source = ?, ingredient = ?, summary = ?, link = ?, image_url = ?, image_urls_json = ?, image_meta_json = ?, meal_type = ?, status = ?, note = ?,
	    ingredients_json = ?, steps_json = ?, flowchart_image_url = ?, flowchart_provider = ?, flowchart_model = ?, flowchart_updated_at = ?, flowchart_source_hash = ?,
	    flowchart_status = ?, flowchart_error = ?, flowchart_requested_at = ?, flowchart_finished_at = ?,
	    parse_status = ?, parse_source = ?, parse_error = ?,
	    parse_requested_at = ?, parse_finished_at = ?, parse_attempts = ?, parse_next_attempt_at = ?, parse_last_error_type = ?, parse_processing_started_at = ?,
	    parsed_content_edited = ?, pinned_at = ?, done_at = ?, updated_by = ?, updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
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

	if current.Status != item.Status {
		if err := insertRecipeStatusEvent(ctx, tx, item.KitchenID, item.ID, current.Status, item.Status, item.UpdatedBy, item.UpdatedAt, "api"); err != nil {
			_ = tx.Rollback()
			return Recipe{}, err
		}
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

func (r *Repository) UpdateStatus(ctx context.Context, recipeID string, kitchenID int64, status string, updatedBy int64, touchedAt string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update status tx: %w", err)
	}

	current, err := findRecipeByIDTx(ctx, tx, recipeID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	doneAt := resolveRecipeDoneAt(current.DoneAt, status, touchedAt)

	result, err := tx.ExecContext(
		ctx,
		`UPDATE recipes SET status = ?, done_at = ?, updated_by = ? WHERE id = ? AND deleted_at IS NULL`,
		status,
		doneAt,
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

	if current.Status != status {
		if err := insertRecipeStatusEvent(ctx, tx, kitchenID, recipeID, current.Status, status, updatedBy, touchedAt, "api"); err != nil {
			_ = tx.Rollback()
			return err
		}
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
