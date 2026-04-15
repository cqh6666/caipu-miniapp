package aialert

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetState(ctx context.Context, providerID string) (State, bool, error) {
	if r == nil || r.db == nil {
		return State{}, false, nil
	}
	return r.getState(ctx, nil, providerID)
}

func (r *Repository) RecordFailure(ctx context.Context, event Event) (State, error) {
	if r == nil || r.db == nil {
		return State{}, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return State{}, err
	}

	state, found, err := r.getState(ctx, tx, event.ProviderID)
	if err != nil {
		_ = tx.Rollback()
		return State{}, err
	}

	now := eventTime(event)
	if !found {
		state = State{
			ProviderID:          strings.TrimSpace(event.ProviderID),
			Scene:               strings.TrimSpace(event.Scene),
			ProviderName:        strings.TrimSpace(event.ProviderName),
			Model:               strings.TrimSpace(event.Model),
			ConsecutiveFailures: 1,
			LastStatus:          "failed",
			LastErrorType:       strings.TrimSpace(event.ErrorType),
			LastErrorMessage:    truncateText(event.ErrorMessage, 500),
			LastHTTPStatus:      event.HTTPStatus,
			LastRequestID:       strings.TrimSpace(event.RequestID),
			LastFailedAt:        now,
			UpdatedAt:           now,
		}
		if err := r.upsertState(ctx, tx, state); err != nil {
			_ = tx.Rollback()
			return State{}, err
		}
		if err := tx.Commit(); err != nil {
			return State{}, err
		}
		return state, nil
	}

	state.Scene = strings.TrimSpace(event.Scene)
	state.ProviderName = strings.TrimSpace(event.ProviderName)
	state.Model = strings.TrimSpace(event.Model)
	state.ConsecutiveFailures++
	state.LastStatus = "failed"
	state.LastErrorType = strings.TrimSpace(event.ErrorType)
	state.LastErrorMessage = truncateText(event.ErrorMessage, 500)
	state.LastHTTPStatus = event.HTTPStatus
	state.LastRequestID = strings.TrimSpace(event.RequestID)
	state.LastFailedAt = now
	state.UpdatedAt = now
	if err := r.upsertState(ctx, tx, state); err != nil {
		_ = tx.Rollback()
		return State{}, err
	}
	if err := tx.Commit(); err != nil {
		return State{}, err
	}
	return state, nil
}

func (r *Repository) RecordSuccess(ctx context.Context, event Event) error {
	if r == nil || r.db == nil {
		return nil
	}

	now := eventTime(event)
	result, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_states
SET
	scene = ?,
	provider_name = ?,
	model = ?,
	consecutive_failures = 0,
	last_status = 'success',
	last_error_type = '',
	last_error_message = '',
	last_http_status = 0,
	last_request_id = ?,
	last_recovered_at = ?,
	last_alerted_failure_count = 0,
	updated_at = ?
WHERE provider_id = ?
`,
		strings.TrimSpace(event.Scene),
		strings.TrimSpace(event.ProviderName),
		strings.TrimSpace(event.Model),
		strings.TrimSpace(event.RequestID),
		now,
		now,
		strings.TrimSpace(event.ProviderID),
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected > 0 {
		return err
	}
	return nil
}

func (r *Repository) MarkAlertSent(ctx context.Context, providerID string, failureCount int, alertedAt string) error {
	if r == nil || r.db == nil {
		return nil
	}
	if strings.TrimSpace(providerID) == "" {
		return nil
	}
	if strings.TrimSpace(alertedAt) == "" {
		alertedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_states
SET
	last_alerted_at = ?,
	last_alerted_failure_count = ?,
	updated_at = ?
WHERE provider_id = ?
`,
		alertedAt,
		failureCount,
		alertedAt,
		strings.TrimSpace(providerID),
	)
	return err
}

func (r *Repository) ListRecentFailures(ctx context.Context, providerID string, limit int) ([]FailureSummary, error) {
	if r == nil || r.db == nil || strings.TrimSpace(providerID) == "" || limit <= 0 {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT
	COALESCE(scene, ''),
	COALESCE(model, ''),
	http_status,
	COALESCE(error_type, ''),
	COALESCE(error_message, ''),
	COALESCE(request_id, ''),
	COALESCE(created_at, '')
FROM ai_call_logs
WHERE provider = ?
  AND status <> 'success'
ORDER BY created_at DESC, id DESC
LIMIT ?
`, strings.TrimSpace(providerID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]FailureSummary, 0, limit)
	for rows.Next() {
		var item FailureSummary
		if err := rows.Scan(
			&item.Scene,
			&item.Model,
			&item.HTTPStatus,
			&item.ErrorType,
			&item.ErrorMessage,
			&item.RequestID,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) getState(ctx context.Context, tx *sql.Tx, providerID string) (State, bool, error) {
	if strings.TrimSpace(providerID) == "" {
		return State{}, false, nil
	}

	queryer := queryRunner(r.db)
	if tx != nil {
		queryer = tx
	}

	var state State
	err := queryer.QueryRowContext(ctx, `
SELECT
	provider_id,
	COALESCE(scene, ''),
	COALESCE(provider_name, ''),
	COALESCE(model, ''),
	consecutive_failures,
	COALESCE(last_status, ''),
	COALESCE(last_error_type, ''),
	COALESCE(last_error_message, ''),
	last_http_status,
	COALESCE(last_request_id, ''),
	COALESCE(last_failed_at, ''),
	COALESCE(last_recovered_at, ''),
	COALESCE(last_alerted_at, ''),
	last_alerted_failure_count,
	COALESCE(updated_at, '')
FROM ai_provider_alert_states
WHERE provider_id = ?
LIMIT 1
`, strings.TrimSpace(providerID)).Scan(
		&state.ProviderID,
		&state.Scene,
		&state.ProviderName,
		&state.Model,
		&state.ConsecutiveFailures,
		&state.LastStatus,
		&state.LastErrorType,
		&state.LastErrorMessage,
		&state.LastHTTPStatus,
		&state.LastRequestID,
		&state.LastFailedAt,
		&state.LastRecoveredAt,
		&state.LastAlertedAt,
		&state.LastAlertedFailureCount,
		&state.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return State{}, false, nil
	}
	return state, err == nil, err
}

func (r *Repository) upsertState(ctx context.Context, tx *sql.Tx, state State) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO ai_provider_alert_states (
	provider_id,
	scene,
	provider_name,
	model,
	consecutive_failures,
	last_status,
	last_error_type,
	last_error_message,
	last_http_status,
	last_request_id,
	last_failed_at,
	last_recovered_at,
	last_alerted_at,
	last_alerted_failure_count,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider_id) DO UPDATE SET
	scene = excluded.scene,
	provider_name = excluded.provider_name,
	model = excluded.model,
	consecutive_failures = excluded.consecutive_failures,
	last_status = excluded.last_status,
	last_error_type = excluded.last_error_type,
	last_error_message = excluded.last_error_message,
	last_http_status = excluded.last_http_status,
	last_request_id = excluded.last_request_id,
	last_failed_at = excluded.last_failed_at,
	last_recovered_at = excluded.last_recovered_at,
	last_alerted_at = excluded.last_alerted_at,
	last_alerted_failure_count = excluded.last_alerted_failure_count,
	updated_at = excluded.updated_at
`,
		state.ProviderID,
		state.Scene,
		state.ProviderName,
		state.Model,
		state.ConsecutiveFailures,
		state.LastStatus,
		state.LastErrorType,
		state.LastErrorMessage,
		state.LastHTTPStatus,
		state.LastRequestID,
		state.LastFailedAt,
		state.LastRecoveredAt,
		state.LastAlertedAt,
		state.LastAlertedFailureCount,
		state.UpdatedAt,
	)
	return err
}

type queryRunner interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func eventTime(event Event) string {
	value := strings.TrimSpace(event.OccurredAt)
	if value != "" {
		return value
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
