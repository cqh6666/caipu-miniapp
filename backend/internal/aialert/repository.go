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
	// 新失败发生时自动解除归档并重新计数（静默为时间态，不在此清除）。
	state.ArchivedAt = ""
	state.ArchivedBy = ""
	state.ArchiveReason = ""
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
	// 真实成功即恢复：清零计数并解除归档与静默（问题已解决）。
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
	archived_at = '',
	archived_by = '',
	archive_reason = '',
	muted_until = '',
	muted_by = '',
	mute_reason = '',
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

// Archive 标记归档，不删除历史失败字段。
func (r *Repository) Archive(ctx context.Context, providerID, subject, reason string) error {
	if r == nil || r.db == nil || strings.TrimSpace(providerID) == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_states
SET archived_at = ?, archived_by = ?, archive_reason = ?, updated_at = ?
WHERE provider_id = ?
`, now, strings.TrimSpace(subject), truncateText(reason, 300), now, strings.TrimSpace(providerID))
	return err
}

// Mute 静默至 mutedUntil（RFC3339）。
func (r *Repository) Mute(ctx context.Context, providerID, subject, mutedUntil, reason string) error {
	if r == nil || r.db == nil || strings.TrimSpace(providerID) == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_states
SET muted_until = ?, muted_by = ?, mute_reason = ?, updated_at = ?
WHERE provider_id = ?
`, strings.TrimSpace(mutedUntil), strings.TrimSpace(subject), truncateText(reason, 300), now, strings.TrimSpace(providerID))
	return err
}

// Unmute 立即解除静默。
func (r *Repository) Unmute(ctx context.Context, providerID string) error {
	if r == nil || r.db == nil || strings.TrimSpace(providerID) == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_states
SET muted_until = '', muted_by = '', mute_reason = '', updated_at = ?
WHERE provider_id = ?
`, now, strings.TrimSpace(providerID))
	return err
}

// MarkRecovered 复测成功恢复：清零计数并解除归档/静默。
func (r *Repository) MarkRecovered(ctx context.Context, providerID, requestID string) error {
	if r == nil || r.db == nil || strings.TrimSpace(providerID) == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_states
SET
	consecutive_failures = 0,
	last_status = 'success',
	last_error_type = '',
	last_error_message = '',
	last_http_status = 0,
	last_request_id = ?,
	last_recovered_at = ?,
	last_alerted_failure_count = 0,
	archived_at = '',
	archived_by = '',
	archive_reason = '',
	muted_until = '',
	muted_by = '',
	mute_reason = '',
	updated_at = ?
WHERE provider_id = ?
`, strings.TrimSpace(requestID), now, now, strings.TrimSpace(providerID))
	return err
}

// RecordRetestFailure 复测失败：更新最后错误信息但不累加连续失败次数
// （复测本身不应把节点推入更深告警）。
func (r *Repository) RecordRetestFailure(ctx context.Context, providerID string, outcome ProviderRetestOutcome) error {
	if r == nil || r.db == nil || strings.TrimSpace(providerID) == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_states
SET
	last_status = 'failed',
	last_error_type = ?,
	last_error_message = ?,
	last_http_status = ?,
	last_request_id = ?,
	last_failed_at = ?,
	updated_at = ?
WHERE provider_id = ?
`,
		strings.TrimSpace(outcome.ErrorType),
		truncateText(outcome.ErrorMessage, 500),
		outcome.HTTPStatus,
		strings.TrimSpace(outcome.RequestID),
		now,
		now,
		strings.TrimSpace(providerID),
	)
	return err
}

// NoteSceneConfigChanged 对某场景下的告警状态打上配置变更标记，供 pending_verify 判定。
func (r *Repository) NoteSceneConfigChanged(ctx context.Context, scene string) error {
	if r == nil || r.db == nil || strings.TrimSpace(scene) == "" {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_states
SET last_config_changed_at = ?
WHERE scene = ?
`, now, strings.TrimSpace(scene))
	return err
}

// ListProviderIDsByScene 返回某场景下有失败记录的 Provider（供配置变更事件与 pending_verify 使用）。
func (r *Repository) ListProviderIDsByScene(ctx context.Context, scene string) ([]string, error) {
	if r == nil || r.db == nil || strings.TrimSpace(scene) == "" {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT provider_id
FROM ai_provider_alert_states
WHERE scene = ? AND consecutive_failures > 0
`, strings.TrimSpace(scene))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// InsertEvent 写入一条告警处置事件。
func (r *Repository) InsertEvent(ctx context.Context, providerID, scene, eventType, reason, subject string) error {
	if r == nil || r.db == nil || strings.TrimSpace(providerID) == "" || strings.TrimSpace(eventType) == "" {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO ai_provider_alert_events (
	provider_id, scene, event_type, reason, operator_subject, created_at
) VALUES (?, ?, ?, ?, ?, ?)
`,
		strings.TrimSpace(providerID),
		strings.TrimSpace(scene),
		strings.TrimSpace(eventType),
		truncateText(reason, 300),
		strings.TrimSpace(subject),
		time.Now().UTC().Format(time.RFC3339),
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

func (r *Repository) ListStates(ctx context.Context, failureThreshold int) ([]State, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if failureThreshold <= 0 {
		failureThreshold = 3
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT`+stateColumns+`
FROM ai_provider_alert_states
ORDER BY
	CASE WHEN consecutive_failures >= ? THEN 1 ELSE 0 END DESC,
	updated_at DESC,
	provider_id ASC
`, failureThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]State, 0)
	for rows.Next() {
		item, err := scanState(rows)
		if err != nil {
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
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
