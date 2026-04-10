package airouter

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
)

type Repository struct {
	db *sql.DB
}

type sceneRecord struct {
	Scene                  Scene
	Enabled                bool
	Strategy               Strategy
	MaxAttempts            int
	RetryOn                []string
	BreakerFailureThreshold int
	BreakerCooldownSeconds int
	RequestOptions         RequestOptions
	UpdatedBy              string
	UpdatedAt              string
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
	var record sceneRecord
	var retryPolicyJSON string
	var requestOptionsJSON string
	var enabledInt int

	err := r.db.QueryRowContext(ctx, `
SELECT
	scene,
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

	rows, err := r.db.QueryContext(ctx, `
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

func (r *Repository) listSceneRecords(ctx context.Context) (map[Scene]sceneRecord, map[Scene][]providerRecord, error) {
	sceneRows, err := r.db.QueryContext(ctx, `
SELECT
	scene,
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
`)
	if err != nil {
		return nil, nil, err
	}
	defer sceneRows.Close()

	scenes := make(map[Scene]sceneRecord, 4)
	for sceneRows.Next() {
		var item sceneRecord
		var retryPolicyJSON string
		var requestOptionsJSON string
		var enabledInt int
		if err := sceneRows.Scan(
			&item.Scene,
			&enabledInt,
			&item.Strategy,
			&item.MaxAttempts,
			&retryPolicyJSON,
			&item.BreakerFailureThreshold,
			&item.BreakerCooldownSeconds,
			&requestOptionsJSON,
			&item.UpdatedBy,
			&item.UpdatedAt,
		); err != nil {
			return nil, nil, err
		}
		item.Enabled = enabledInt == 1
		_ = json.Unmarshal([]byte(retryPolicyJSON), &item.RetryOn)
		_ = json.Unmarshal([]byte(requestOptionsJSON), &item.RequestOptions)
		scenes[item.Scene] = item
	}
	if err := sceneRows.Err(); err != nil {
		return nil, nil, err
	}

	providerRows, err := r.db.QueryContext(ctx, `
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
ORDER BY scene ASC, priority ASC, id ASC
`)
	if err != nil {
		return nil, nil, err
	}
	defer providerRows.Close()

	providers := make(map[Scene][]providerRecord, 4)
	for providerRows.Next() {
		var item providerRecord
		var enabledInt int
		var extraJSON string
		if err := providerRows.Scan(
			&item.ID,
			&item.Scene,
			&item.Name,
			&item.Adapter,
			&enabledInt,
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
			return nil, nil, err
		}
		item.Enabled = enabledInt == 1
		_ = json.Unmarshal([]byte(extraJSON), &item.Extra)
		providers[item.Scene] = append(providers[item.Scene], item)
	}
	if err := providerRows.Err(); err != nil {
		return nil, nil, err
	}

	for scene := range providers {
		sort.SliceStable(providers[scene], func(i, j int) bool {
			if providers[scene][i].Priority != providers[scene][j].Priority {
				return providers[scene][i].Priority < providers[scene][j].Priority
			}
			return providers[scene][i].ID < providers[scene][j].ID
		})
	}

	return scenes, providers, nil
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
