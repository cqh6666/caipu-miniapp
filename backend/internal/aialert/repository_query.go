package aialert

import (
	"context"
	"strings"
)

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
