package recipe

import (
	"context"
	"database/sql"
	"fmt"
)

func (r *Repository) ListImageMirrorCandidates(ctx context.Context, limit int) ([]Recipe, error) {
	const query = `
	SELECT id, kitchen_id, title, COALESCE(title_source, 'manual'), COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parse_attempts, 0), COALESCE(parse_next_attempt_at, ''), COALESCE(parse_last_error_type, ''), COALESCE(parse_processing_started_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''), COALESCE(done_at, ''),
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
