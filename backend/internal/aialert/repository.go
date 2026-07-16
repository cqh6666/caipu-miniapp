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
SELECT`+stateColumns+`
FROM ai_provider_alert_states
WHERE provider_id = ?
LIMIT 1
`, strings.TrimSpace(providerID)).Scan(scanStateTargets(&state)...)
	if err == sql.ErrNoRows {
		return State{}, false, nil
	}
	return state, err == nil, err
}

func (r *Repository) upsertState(ctx context.Context, tx *sql.Tx, state State) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO ai_provider_alert_states (
	provider_id,
	failure_streak_id,
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
	archived_at,
	archived_by,
	archive_reason,
	muted_until,
	muted_by,
	mute_reason,
	last_config_changed_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider_id) DO UPDATE SET
	failure_streak_id = excluded.failure_streak_id,
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
	archived_at = excluded.archived_at,
	archived_by = excluded.archived_by,
	archive_reason = excluded.archive_reason,
	muted_until = excluded.muted_until,
	muted_by = excluded.muted_by,
	mute_reason = excluded.mute_reason,
	last_config_changed_at = excluded.last_config_changed_at,
	updated_at = excluded.updated_at
`,
		state.ProviderID,
		state.FailureStreakID,
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
		state.ArchivedAt,
		state.ArchivedBy,
		state.ArchiveReason,
		state.MutedUntil,
		state.MutedBy,
		state.MuteReason,
		state.LastConfigChangedAt,
		state.UpdatedAt,
	)
	return err
}

type queryRunner interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type stateScanner interface {
	Scan(dest ...any) error
}

func scanState(scanner stateScanner) (State, error) {
	var state State
	err := scanner.Scan(scanStateTargets(&state)...)
	return state, err
}

func scanStateTargets(state *State) []any {
	return []any{
		&state.ProviderID,
		&state.FailureStreakID,
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
		&state.ArchivedAt,
		&state.ArchivedBy,
		&state.ArchiveReason,
		&state.MutedUntil,
		&state.MutedBy,
		&state.MuteReason,
		&state.LastConfigChangedAt,
		&state.UpdatedAt,
	}
}

// stateColumns 是所有 SELECT 的统一列清单，顺序必须与 scanStateTargets 对齐。
const stateColumns = `
	provider_id,
	COALESCE(failure_streak_id, ''),
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
	COALESCE(archived_at, ''),
	COALESCE(archived_by, ''),
	COALESCE(archive_reason, ''),
	COALESCE(muted_until, ''),
	COALESCE(muted_by, ''),
	COALESCE(mute_reason, ''),
	COALESCE(last_config_changed_at, ''),
	COALESCE(updated_at, '')`

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
