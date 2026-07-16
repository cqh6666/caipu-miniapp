package recipe

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func (r *Repository) ListPendingAutoParse(ctx context.Context, limit int) ([]Recipe, error) {
	const query = `
	SELECT id, kitchen_id, title, COALESCE(title_source, 'manual'), COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parse_attempts, 0), COALESCE(parse_next_attempt_at, ''), COALESCE(parse_last_error_type, ''), COALESCE(parse_processing_started_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''), COALESCE(done_at, ''),
	       created_by, updated_by, created_at, updated_at, COALESCE(version, 1)
	FROM recipes
	WHERE deleted_at IS NULL AND parse_status = ?
	  AND (
	    COALESCE(parse_next_attempt_at, '') = ''
	    OR datetime(parse_next_attempt_at) <= datetime('now')
	  )
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
	SELECT id, kitchen_id, title, COALESCE(title_source, 'manual'), COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parse_attempts, 0), COALESCE(parse_next_attempt_at, ''), COALESCE(parse_last_error_type, ''), COALESCE(parse_processing_started_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''), COALESCE(done_at, ''),
	       created_by, updated_by, created_at, updated_at, COALESCE(version, 1)
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

func (r *Repository) MarkAutoParseProcessing(ctx context.Context, recipeID, parseSource, startedAt, leaseExpiresAt string) (string, bool, error) {
	claimToken, err := common.NewPrefixedID("parse_claim")
	if err != nil {
		return "", false, fmt.Errorf("generate auto parse claim token: %w", err)
	}
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET parse_status = ?, parse_source = ?, parse_error = '', parse_processing_started_at = ?,
	    parse_finished_at = NULL, parse_attempts = COALESCE(parse_attempts, 0) + 1,
	    parse_next_attempt_at = '', parse_last_error_type = '', parse_claim_token = ?,
	    parse_claim_content_version = COALESCE(content_version, 0), parse_lease_expires_at = ?
	WHERE id = ? AND deleted_at IS NULL AND parse_status = ?`,
		ParseStatusProcessing,
		parseSource,
		nonNullableTrimmedString(startedAt),
		claimToken,
		nonNullableTrimmedString(leaseExpiresAt),
		recipeID,
		ParseStatusPending,
	)
	if err != nil {
		return "", false, fmt.Errorf("mark recipe auto parse processing: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", false, fmt.Errorf("read auto parse processing rows: %w", err)
	}

	if rowsAffected == 0 {
		return "", false, nil
	}
	return claimToken, true, nil
}

func (r *Repository) RenewAutoParseLease(ctx context.Context, recipeID, claimToken, leaseExpiresAt string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET parse_lease_expires_at = ?
	WHERE id = ? AND deleted_at IS NULL AND parse_status = ? AND parse_claim_token = ?`,
		nonNullableTrimmedString(leaseExpiresAt),
		recipeID,
		ParseStatusProcessing,
		claimToken,
	)
	if err != nil {
		return fmt.Errorf("renew recipe auto parse lease: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read auto parse lease renewal rows: %w", err)
	}
	if rowsAffected == 0 {
		return ErrStaleJobResult
	}
	return nil
}

func (r *Repository) MarkAutoParsePending(ctx context.Context, recipeID, parseSource, requestedAt string) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET parse_status = ?, parse_source = ?, parse_error = '', parse_requested_at = ?, parse_finished_at = NULL,
	    parse_attempts = 0, parse_next_attempt_at = '', parse_last_error_type = '', parse_processing_started_at = '',
	    parse_claim_token = '', parse_claim_content_version = 0, parse_lease_expires_at = '',
	    updated_at = ?
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

func (r *Repository) RequeueStaleAutoParse(ctx context.Context, staleBefore, requestedAt string) (int64, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET parse_status = ?, parse_error = '', parse_requested_at = ?, parse_finished_at = NULL,
	    parse_next_attempt_at = '', parse_processing_started_at = '',
	    parse_claim_token = '', parse_claim_content_version = 0, parse_lease_expires_at = ''
	WHERE deleted_at IS NULL
	  AND parse_status = ?
	  AND (
	    (COALESCE(parse_lease_expires_at, '') <> '' AND datetime(parse_lease_expires_at) <= datetime(?))
	    OR (COALESCE(parse_lease_expires_at, '') = '' AND datetime(COALESCE(NULLIF(parse_processing_started_at, ''), NULLIF(parse_requested_at, ''), NULLIF(updated_at, ''), NULLIF(created_at, ''))) <= datetime(?))
	  )`,
		ParseStatusPending,
		nullableString(requestedAt),
		ParseStatusProcessing,
		requestedAt,
		staleBefore,
	)
	if err != nil {
		return 0, fmt.Errorf("requeue stale recipe auto parse jobs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read stale auto parse rows: %w", err)
	}

	return rowsAffected, nil
}

func (r *Repository) MarkAutoParseFailed(ctx context.Context, recipeID, claimToken, parseSource, parseError, errorType, finishedAt string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET parse_status = ?, parse_source = ?, parse_error = ?, parse_finished_at = ?,
	    parse_next_attempt_at = '', parse_last_error_type = ?, parse_processing_started_at = '',
	    parse_claim_token = '', parse_claim_content_version = 0, parse_lease_expires_at = ''
	WHERE id = ? AND deleted_at IS NULL AND parse_status = ? AND parse_claim_token = ?`,
		ParseStatusFailed,
		parseSource,
		truncateString(strings.TrimSpace(parseError), 300),
		finishedAt,
		nonNullableTrimmedString(errorType),
		recipeID,
		ParseStatusProcessing,
		claimToken,
	)
	if err != nil {
		return fmt.Errorf("mark recipe auto parse failed: %w", err)
	}
	return requireClaimedRow(result, "mark recipe auto parse failed")
}

func (r *Repository) MarkAutoParseRetryPending(ctx context.Context, recipeID, claimToken, parseSource, parseError, errorType, nextAttemptAt string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET parse_status = ?, parse_source = ?, parse_error = ?, parse_finished_at = NULL, parse_next_attempt_at = ?,
	    parse_last_error_type = ?, parse_processing_started_at = '',
	    parse_claim_token = '', parse_claim_content_version = 0, parse_lease_expires_at = ''
	WHERE id = ? AND deleted_at IS NULL AND parse_status = ? AND parse_claim_token = ?`,
		ParseStatusPending,
		parseSource,
		truncateString(strings.TrimSpace(parseError), 300),
		nonNullableTrimmedString(nextAttemptAt),
		nonNullableTrimmedString(errorType),
		recipeID,
		ParseStatusProcessing,
		claimToken,
	)
	if err != nil {
		return fmt.Errorf("mark recipe auto parse retry pending: %w", err)
	}
	return requireClaimedRow(result, "mark recipe auto parse retry pending")
}

func (r *Repository) ApplyAutoParseResult(ctx context.Context, recipeID, claimToken, parseSource, parseError, finishedAt string, draft Recipe) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin apply auto parse tx: %w", err)
	}

	current, err := findRecipeByIDTx(ctx, tx, recipeID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	var currentClaim string
	var contentVersion int64
	var claimContentVersion int64
	var parsedContentEdited bool
	var parseStatus string
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(parse_claim_token, ''), COALESCE(content_version, 0), COALESCE(parse_claim_content_version, 0),
       COALESCE(parsed_content_edited, 0), COALESCE(parse_status, '')
FROM recipes WHERE id = ? AND deleted_at IS NULL`, recipeID).Scan(
		&currentClaim,
		&contentVersion,
		&claimContentVersion,
		&parsedContentEdited,
		&parseStatus,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("read auto parse claim: %w", err)
	}
	if parseStatus != ParseStatusProcessing || currentClaim == "" || currentClaim != claimToken {
		_ = tx.Rollback()
		return ErrStaleJobResult
	}
	if contentVersion != claimContentVersion || parsedContentEdited {
		result, updateErr := tx.ExecContext(ctx, `
UPDATE recipes
SET parse_status = ?, parse_error = ?, parse_finished_at = ?, parse_last_error_type = ?,
    parse_processing_started_at = '', parse_claim_token = '', parse_claim_content_version = 0,
    parse_lease_expires_at = ''
WHERE id = ? AND deleted_at IS NULL AND parse_status = ? AND parse_claim_token = ?`,
			ParseStatusFailed,
			"菜谱已被人工修改，自动解析结果未应用",
			finishedAt,
			"content_conflict",
			recipeID,
			ParseStatusProcessing,
			claimToken,
		)
		if updateErr != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record auto parse content conflict: %w", updateErr)
		}
		if err := requireClaimedRow(result, "record auto parse content conflict"); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit auto parse content conflict: %w", err)
		}
		return ErrAutoParseContentChanged
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
SET title = ?, title_source = ?, ingredient = ?, summary = ?, image_url = ?, image_urls_json = ?, image_meta_json = ?, ingredients_json = ?, steps_json = ?,
	    parse_status = ?, parse_source = ?, parse_error = ?, parse_finished_at = ?,
	    parse_next_attempt_at = '', parse_last_error_type = '', parse_processing_started_at = '',
	    parse_claim_token = '', parse_claim_content_version = 0, parse_lease_expires_at = '',
	    parsed_content_edited = 0, updated_at = ?
	WHERE id = ? AND deleted_at IS NULL AND parse_status = ? AND parse_claim_token = ?
	  AND COALESCE(content_version, 0) = COALESCE(parse_claim_content_version, 0)
	  AND COALESCE(parsed_content_edited, 0) = 0`,
		resolveAutoParseTitle(current, draft),
		resolveAutoParseTitleSource(current, draft),
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
		ParseStatusProcessing,
		claimToken,
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
	    parse_attempts = 0, parse_next_attempt_at = '', parse_last_error_type = '', parse_processing_started_at = '',
	    parse_claim_token = '', parse_claim_content_version = 0, parse_lease_expires_at = '',
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

func requireClaimedRow(result sql.Result, operation string) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read %s rows: %w", operation, err)
	}
	if rowsAffected == 0 {
		return ErrStaleJobResult
	}
	return nil
}
