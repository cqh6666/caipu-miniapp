package aialert

import (
	"context"
	"strings"
	"time"
)

func (r *Repository) GetState(ctx context.Context, providerID string) (State, bool, error) {
	if r == nil || r.db == nil {
		return State{}, false, nil
	}
	return r.getState(ctx, nil, providerID)
}

func (r *Repository) RecordFailure(ctx context.Context, event Event, alertThreshold int) (State, bool, error) {
	if r == nil || r.db == nil {
		return State{}, false, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return State{}, false, err
	}

	state, found, err := r.getState(ctx, tx, event.ProviderID)
	if err != nil {
		_ = tx.Rollback()
		return State{}, false, err
	}

	now := eventTime(event)
	startsNewStreak := !found || state.ConsecutiveFailures == 0 || !strings.EqualFold(state.LastStatus, "failed") || strings.TrimSpace(state.FailureStreakID) == ""
	if startsNewStreak {
		state.FailureStreakID, err = newDeliveryID()
		if err != nil {
			_ = tx.Rollback()
			return State{}, false, err
		}
	}
	if !found {
		state = State{
			ProviderID:          strings.TrimSpace(event.ProviderID),
			FailureStreakID:     state.FailureStreakID,
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
	} else {
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
	}
	if err := r.upsertState(ctx, tx, state); err != nil {
		_ = tx.Rollback()
		return State{}, false, err
	}

	enqueued := false
	if alertThreshold > 0 && state.ConsecutiveFailures >= alertThreshold && state.LastAlertedFailureCount < alertThreshold {
		enqueued, err = r.enqueueDeliveryTx(ctx, tx, state, event)
		if err != nil {
			_ = tx.Rollback()
			return State{}, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return State{}, false, err
	}
	return state, enqueued, nil
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
	failure_streak_id = '',
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
	failure_streak_id = '',
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
