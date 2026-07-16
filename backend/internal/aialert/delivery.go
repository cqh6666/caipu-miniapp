package aialert

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const deliveryClaimLease = 30 * time.Second

func newDeliveryID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate alert delivery id: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}

func (r *Repository) enqueueDeliveryTx(ctx context.Context, tx *sql.Tx, state State, event Event) (bool, error) {
	eventID, err := newDeliveryID()
	if err != nil {
		return false, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO ai_provider_alert_deliveries (
	event_id,
	failure_streak_id,
	provider_id,
	scene,
	trigger_source,
	target_type,
	target_id,
	request_id,
	status,
	available_at,
	created_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?, ?)
`,
		eventID,
		state.FailureStreakID,
		state.ProviderID,
		state.Scene,
		strings.TrimSpace(event.TriggerSource),
		strings.TrimSpace(event.TargetType),
		strings.TrimSpace(event.TargetID),
		strings.TrimSpace(event.RequestID),
		now,
		now,
		now,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

func (r *Repository) ClaimNextDelivery(ctx context.Context, lease time.Duration) (Delivery, bool, error) {
	if r == nil || r.db == nil {
		return Delivery{}, false, nil
	}
	if lease <= 0 {
		lease = deliveryClaimLease
	}
	claimToken, err := newDeliveryID()
	if err != nil {
		return Delivery{}, false, err
	}
	nowTime := time.Now().UTC()
	now := nowTime.Format(time.RFC3339Nano)
	expiresAt := nowTime.Add(lease).Format(time.RFC3339Nano)

	var delivery Delivery
	err = r.db.QueryRowContext(ctx, `
UPDATE ai_provider_alert_deliveries
SET
	status = 'sending',
	attempt_count = attempt_count + 1,
	claim_token = ?,
	claim_expires_at = ?,
	updated_at = ?
WHERE event_id = (
	SELECT event_id
	FROM ai_provider_alert_deliveries
	WHERE
		(status = 'pending' AND available_at <= ?)
		OR (status = 'sending' AND claim_expires_at != '' AND claim_expires_at <= ?)
	ORDER BY available_at ASC, created_at ASC, event_id ASC
	LIMIT 1
)
RETURNING
	event_id,
	failure_streak_id,
	provider_id,
	COALESCE(scene, ''),
	COALESCE(trigger_source, ''),
	COALESCE(target_type, ''),
	COALESCE(target_id, ''),
	COALESCE(request_id, ''),
	status,
	attempt_count,
	claim_token,
	claim_expires_at,
	available_at,
	COALESCE(last_error, ''),
	created_at,
	updated_at,
	COALESCE(sent_at, '')
`, claimToken, expiresAt, now, now, now).Scan(
		&delivery.EventID,
		&delivery.FailureStreakID,
		&delivery.ProviderID,
		&delivery.Scene,
		&delivery.TriggerSource,
		&delivery.TargetType,
		&delivery.TargetID,
		&delivery.RequestID,
		&delivery.Status,
		&delivery.AttemptCount,
		&delivery.ClaimToken,
		&delivery.ClaimExpiresAt,
		&delivery.AvailableAt,
		&delivery.LastError,
		&delivery.CreatedAt,
		&delivery.UpdatedAt,
		&delivery.SentAt,
	)
	if err == sql.ErrNoRows {
		return Delivery{}, false, nil
	}
	if err != nil {
		return Delivery{}, false, err
	}
	return delivery, true, nil
}

func (r *Repository) MarkDeliverySent(ctx context.Context, delivery Delivery, failureCount int, sentAt string) error {
	if r == nil || r.db == nil {
		return nil
	}
	if strings.TrimSpace(sentAt) == "" {
		sentAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
UPDATE ai_provider_alert_deliveries
SET
	status = 'sent',
	claim_token = '',
	claim_expires_at = '',
	last_error = '',
	sent_at = ?,
	updated_at = ?
WHERE event_id = ? AND status = 'sending' AND claim_token = ?
`, sentAt, sentAt, delivery.EventID, delivery.ClaimToken)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if affected != 1 {
		_ = tx.Rollback()
		return fmt.Errorf("alert delivery claim lost: %s", delivery.EventID)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE ai_provider_alert_states
SET
	last_alerted_at = ?,
	last_alerted_failure_count = ?,
	updated_at = ?
WHERE provider_id = ? AND failure_streak_id = ?
`, sentAt, failureCount, sentAt, delivery.ProviderID, delivery.FailureStreakID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *Repository) MarkDeliveryFailed(ctx context.Context, delivery Delivery, cause error, retryDelay time.Duration) error {
	if r == nil || r.db == nil {
		return nil
	}
	if retryDelay < 0 {
		retryDelay = 0
	}
	nowTime := time.Now().UTC()
	now := nowTime.Format(time.RFC3339Nano)
	availableAt := nowTime.Add(retryDelay).Format(time.RFC3339Nano)
	message := ""
	if cause != nil {
		message = truncateText(cause.Error(), 500)
	}
	result, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_deliveries
SET
	status = 'pending',
	claim_token = '',
	claim_expires_at = '',
	available_at = ?,
	last_error = ?,
	updated_at = ?
WHERE event_id = ? AND status = 'sending' AND claim_token = ?
`, availableAt, message, now, delivery.EventID, delivery.ClaimToken)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("alert delivery claim lost: %s", delivery.EventID)
	}
	return nil
}

func (r *Repository) MarkDeliveryCancelled(ctx context.Context, delivery Delivery, reason string) error {
	if r == nil || r.db == nil {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := r.db.ExecContext(ctx, `
UPDATE ai_provider_alert_deliveries
SET
	status = 'cancelled',
	claim_token = '',
	claim_expires_at = '',
	last_error = ?,
	updated_at = ?
WHERE event_id = ? AND status = 'sending' AND claim_token = ?
`, truncateText(reason, 500), now, delivery.EventID, delivery.ClaimToken)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("alert delivery claim lost: %s", delivery.EventID)
	}
	return nil
}
