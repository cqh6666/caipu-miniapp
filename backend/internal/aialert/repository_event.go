package aialert

import (
	"context"
	"strings"
	"time"
)

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
