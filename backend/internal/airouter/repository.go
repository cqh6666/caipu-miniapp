package airouter

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
)

type Repository struct {
	db *sql.DB
}

type sceneRecord struct {
	Scene                   Scene
	Version                 int
	Enabled                 bool
	Strategy                Strategy
	MaxAttempts             int
	RetryOn                 []string
	BreakerFailureThreshold int
	BreakerCooldownSeconds  int
	RequestOptions          RequestOptions
	UpdatedBy               string
	UpdatedAt               string
}

type providerRecord struct {
	ID             string
	Scene          Scene
	Name           string
	Adapter        string
	Enabled        bool
	Priority       int
	Weight         int
	BaseURL        string
	APIKeyCipher   string
	Model          string
	TimeoutSeconds int
	Extra          map[string]any
	UpdatedBy      string
	UpdatedAt      string
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) loadScene(ctx context.Context, scene Scene) (sceneRecord, []providerRecord, bool, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return sceneRecord{}, nil, false, err
	}
	record, providers, found, err := loadSceneSnapshot(ctx, tx, scene)
	if err != nil {
		_ = tx.Rollback()
		return sceneRecord{}, nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return sceneRecord{}, nil, false, err
	}
	return record, providers, found, nil
}

type sceneQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadSceneSnapshot(ctx context.Context, queryer sceneQueryer, scene Scene) (sceneRecord, []providerRecord, bool, error) {
	var record sceneRecord
	var retryPolicyJSON string
	var requestOptionsJSON string
	var enabledInt int

	err := queryer.QueryRowContext(ctx, `
SELECT
	scene,
	version,
	enabled,
	COALESCE(strategy, ''),
	max_attempts,
	COALESCE(retry_policy_json, '[]'),
	breaker_failure_threshold,
	breaker_cooldown_seconds,
	COALESCE(request_options_json, '{}'),
	COALESCE(updated_by_subject, ''),
	COALESCE(updated_at, '')
FROM ai_route_scenes
WHERE scene = ?
LIMIT 1
`, string(scene)).Scan(
		&record.Scene,
		&record.Version,
		&enabledInt,
		&record.Strategy,
		&record.MaxAttempts,
		&retryPolicyJSON,
		&record.BreakerFailureThreshold,
		&record.BreakerCooldownSeconds,
		&requestOptionsJSON,
		&record.UpdatedBy,
		&record.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return sceneRecord{}, nil, false, nil
	}
	if err != nil {
		return sceneRecord{}, nil, false, err
	}
	record.Enabled = enabledInt == 1
	if err := json.Unmarshal([]byte(retryPolicyJSON), &record.RetryOn); err != nil {
		record.RetryOn = nil
	}
	if err := json.Unmarshal([]byte(requestOptionsJSON), &record.RequestOptions); err != nil {
		record.RequestOptions = RequestOptions{}
	}

	rows, err := queryer.QueryContext(ctx, `
SELECT
	id,
	scene,
	COALESCE(name, ''),
	COALESCE(adapter, ''),
	enabled,
	priority,
	weight,
	COALESCE(base_url, ''),
	COALESCE(api_key_ciphertext, ''),
	COALESCE(model, ''),
	timeout_seconds,
	COALESCE(extra_json, '{}'),
	COALESCE(updated_by_subject, ''),
	COALESCE(updated_at, '')
FROM ai_route_providers
WHERE scene = ?
ORDER BY priority ASC, id ASC
`, string(scene))
	if err != nil {
		return sceneRecord{}, nil, false, err
	}
	defer rows.Close()

	providers := make([]providerRecord, 0, 4)
	for rows.Next() {
		var item providerRecord
		var enabledProvider int
		var extraJSON string
		if err := rows.Scan(
			&item.ID,
			&item.Scene,
			&item.Name,
			&item.Adapter,
			&enabledProvider,
			&item.Priority,
			&item.Weight,
			&item.BaseURL,
			&item.APIKeyCipher,
			&item.Model,
			&item.TimeoutSeconds,
			&extraJSON,
			&item.UpdatedBy,
			&item.UpdatedAt,
		); err != nil {
			return sceneRecord{}, nil, false, err
		}
		item.Enabled = enabledProvider == 1
		if err := json.Unmarshal([]byte(extraJSON), &item.Extra); err != nil {
			item.Extra = nil
		}
		providers = append(providers, item)
	}
	if err := rows.Err(); err != nil {
		return sceneRecord{}, nil, false, err
	}

	return record, providers, true, nil
}

func normalizeRetryOn(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
