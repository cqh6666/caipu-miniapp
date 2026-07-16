package recipe

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func (r *Repository) QueueFlowchart(ctx context.Context, recipeID, requestedAt string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET flowchart_status = ?, flowchart_error = '', flowchart_requested_at = ?, flowchart_finished_at = NULL,
	    flowchart_claim_token = '', flowchart_claim_content_version = 0, flowchart_lease_expires_at = ''
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

func (r *Repository) ListPendingFlowcharts(ctx context.Context, limit int) ([]Recipe, error) {
	const query = `
	SELECT id, kitchen_id, title, COALESCE(title_source, 'manual'), COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parse_attempts, 0), COALESCE(parse_next_attempt_at, ''), COALESCE(parse_last_error_type, ''), COALESCE(parse_processing_started_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''), COALESCE(done_at, ''),
	       created_by, updated_by, created_at, updated_at, COALESCE(version, 1)
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
	SELECT id, kitchen_id, title, COALESCE(title_source, 'manual'), COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parse_attempts, 0), COALESCE(parse_next_attempt_at, ''), COALESCE(parse_last_error_type, ''), COALESCE(parse_processing_started_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''), COALESCE(done_at, ''),
	       created_by, updated_by, created_at, updated_at, COALESCE(version, 1)
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

func (r *Repository) MarkFlowchartProcessing(ctx context.Context, recipeID, leaseExpiresAt string) (string, bool, error) {
	claimToken, err := common.NewPrefixedID("flow_claim")
	if err != nil {
		return "", false, fmt.Errorf("generate flowchart claim token: %w", err)
	}
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET flowchart_status = ?, flowchart_error = '', flowchart_claim_token = ?,
	    flowchart_claim_content_version = COALESCE(content_version, 0), flowchart_lease_expires_at = ?
	WHERE id = ? AND deleted_at IS NULL AND flowchart_status = ?`,
		FlowchartStatusProcessing,
		claimToken,
		nonNullableTrimmedString(leaseExpiresAt),
		recipeID,
		FlowchartStatusPending,
	)
	if err != nil {
		return "", false, fmt.Errorf("mark recipe flowchart processing: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", false, fmt.Errorf("read flowchart processing rows: %w", err)
	}
	if rowsAffected == 0 {
		return "", false, nil
	}
	return claimToken, true, nil
}

func (r *Repository) RenewFlowchartLease(ctx context.Context, recipeID, claimToken, leaseExpiresAt string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET flowchart_lease_expires_at = ?
	WHERE id = ? AND deleted_at IS NULL AND flowchart_status = ? AND flowchart_claim_token = ?`,
		nonNullableTrimmedString(leaseExpiresAt),
		recipeID,
		FlowchartStatusProcessing,
		claimToken,
	)
	if err != nil {
		return fmt.Errorf("renew recipe flowchart lease: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read flowchart lease renewal rows: %w", err)
	}
	if rowsAffected == 0 {
		return ErrStaleJobResult
	}
	return nil
}

func (r *Repository) MarkAutoFlowchartPending(ctx context.Context, recipeID, requestedAt string) (bool, error) {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET flowchart_status = ?, flowchart_error = '', flowchart_requested_at = ?, flowchart_finished_at = NULL,
	    flowchart_claim_token = '', flowchart_claim_content_version = 0, flowchart_lease_expires_at = ''
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
	SET flowchart_status = ?, flowchart_error = '', flowchart_requested_at = ?, flowchart_finished_at = NULL,
	    flowchart_claim_token = '', flowchart_claim_content_version = 0, flowchart_lease_expires_at = ''
	WHERE deleted_at IS NULL
	  AND flowchart_status = ?
	  AND (
	    (COALESCE(flowchart_lease_expires_at, '') <> '' AND datetime(flowchart_lease_expires_at) <= datetime(?))
	    OR (COALESCE(flowchart_lease_expires_at, '') = '' AND datetime(COALESCE(NULLIF(flowchart_requested_at, ''), NULLIF(updated_at, ''), NULLIF(created_at, ''))) <= datetime(?))
	  )`,
		FlowchartStatusPending,
		nullableString(requestedAt),
		FlowchartStatusProcessing,
		requestedAt,
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

func (r *Repository) MarkFlowchartFailed(ctx context.Context, recipeID, claimToken, flowchartError, finishedAt string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET flowchart_status = ?, flowchart_error = ?, flowchart_finished_at = ?,
	    flowchart_claim_token = '', flowchart_claim_content_version = 0, flowchart_lease_expires_at = ''
	WHERE id = ? AND deleted_at IS NULL AND flowchart_status = ? AND flowchart_claim_token = ?`,
		FlowchartStatusFailed,
		truncateString(strings.TrimSpace(flowchartError), 300),
		finishedAt,
		recipeID,
		FlowchartStatusProcessing,
		claimToken,
	)
	if err != nil {
		return fmt.Errorf("mark recipe flowchart failed: %w", err)
	}
	return requireClaimedRow(result, "mark recipe flowchart failed")
}

func (r *Repository) ApplyFlowchartResult(ctx context.Context, recipeID, claimToken, imageURL, provider, model, sourceHash, finishedAt string) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE recipes
	SET flowchart_image_url = ?, flowchart_provider = ?, flowchart_model = ?, flowchart_updated_at = ?, flowchart_source_hash = ?,
	    flowchart_status = ?, flowchart_error = '', flowchart_finished_at = ?,
	    flowchart_claim_token = '', flowchart_claim_content_version = 0, flowchart_lease_expires_at = ''
	WHERE id = ? AND deleted_at IS NULL AND flowchart_status = ? AND flowchart_claim_token = ?
	  AND COALESCE(content_version, 0) = COALESCE(flowchart_claim_content_version, 0)`,
		nullableString(imageURL),
		strings.TrimSpace(provider),
		strings.TrimSpace(model),
		nullableString(finishedAt),
		strings.TrimSpace(sourceHash),
		FlowchartStatusDone,
		finishedAt,
		recipeID,
		FlowchartStatusProcessing,
		claimToken,
	)
	if err != nil {
		return fmt.Errorf("apply recipe flowchart result: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read flowchart result rows: %w", err)
	}
	if rowsAffected == 0 {
		return ErrStaleJobResult
	}

	return nil
}
