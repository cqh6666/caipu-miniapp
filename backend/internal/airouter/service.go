package airouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
)

type CompatibilityLoader func(context.Context, Scene) SceneConfig
type TestInputBuilder func(Scene) (ChatCompletionInput, bool)

const (
	providerExtraKeyEndpointMode           = "endpoint_mode"
	providerExtraKeyResponseFormat         = "response_format"
	providerExtraKeyThinkingType           = "thinking_type"
	providerExtraKeyReasoningEffort        = "reasoning_effort"
	providerExtraKeyImageSize              = "size"
	providerExtraKeyImageQuality           = "quality"
	providerExtraKeyImageBackground        = "background"
	providerExtraKeyImageOutputFormat      = "output_format"
	providerExtraKeyImageOutputCompression = "output_compression"
	providerExtraKeyImageN                 = "n"
	defaultImageOutputFormat               = "png"
)

type Service struct {
	repo             *Repository
	cipherBox        *cipherBox
	compatibility    CompatibilityLoader
	testInputBuilder TestInputBuilder
	tracker          audit.Tracker
	alertTracker     aialert.Tracker
	breaker          *breakerStore
	roundRobinMu     sync.Mutex
	roundRobinNext   map[Scene]int
}

func (s *Service) ConfigureCredentialKeys(secret, version string, previous []credentialcipher.Key) error {
	box, err := newVersionedCipherBox(secret, version, previous)
	if err != nil {
		return err
	}
	s.cipherBox = box
	return nil
}

func NewService(repo *Repository, secret string, compatibility CompatibilityLoader, tracker audit.Tracker, alertTracker aialert.Tracker) *Service {
	return &Service{
		repo:           repo,
		cipherBox:      newCipherBox(secret),
		compatibility:  compatibility,
		tracker:        tracker,
		alertTracker:   alertTracker,
		breaker:        newBreakerStore(),
		roundRobinNext: make(map[Scene]int),
	}
}

func (s *Service) SetTestInputBuilder(builder TestInputBuilder) {
	if s == nil {
		return
	}
	s.testInputBuilder = builder
}

func (s *Service) ListScenes(ctx context.Context) ([]SceneSummaryView, error) {
	items := make([]SceneSummaryView, 0, len(AllScenes()))
	for _, scene := range AllScenes() {
		config, err := s.GetScene(ctx, scene)
		if err != nil {
			return nil, err
		}
		activeCount := len(enabledProviders(config.Providers))
		items = append(items, SceneSummaryView{
			Scene:               scene,
			Enabled:             config.Enabled,
			Strategy:            config.Strategy,
			ProviderCount:       len(config.Providers),
			ActiveProviderCount: activeCount,
			UpdatedBy:           config.UpdatedBy,
			UpdatedAt:           config.UpdatedAt,
			Source:              config.Source,
			CompatibilityMode:   config.CompatibilityMode,
		})
	}
	return items, nil
}

func (s *Service) GetScene(ctx context.Context, scene Scene) (SceneConfig, error) {
	if !IsValidScene(string(scene)) {
		return SceneConfig{}, common.ErrNotFound
	}

	if s == nil || s.repo == nil {
		return s.compatibilityScene(ctx, scene), nil
	}

	record, providers, found, err := s.repo.loadScene(ctx, scene)
	if err != nil {
		return SceneConfig{}, err
	}
	if found {
		config, err := s.buildSceneConfig(record, providers)
		if err != nil {
			return SceneConfig{}, err
		}
		config.Source = "db"
		config.CompatibilityMode = sceneUsesCompatibility(config)
		return config, nil
	}

	config := s.compatibilityScene(ctx, scene)
	config.Enabled = false
	config.Source = "compat"
	config.CompatibilityMode = true
	for index := range config.Providers {
		config.Providers[index].APIKey = ""
		if config.Providers[index].HasAPIKey && config.Providers[index].APIKeyMasked == "" {
			config.Providers[index].APIKeyMasked = "****"
		}
	}
	return config, nil
}

func (s *Service) IsSceneAvailable(ctx context.Context, scene Scene) bool {
	config, err := s.runtimeScene(ctx, scene)
	if err != nil {
		return false
	}
	return config.Enabled && len(enabledProviders(config.Providers)) > 0
}

func (s *Service) SaveScene(ctx context.Context, subject, requestID string, scene Scene, input SceneConfig) (SceneConfig, error) {
	if s == nil || s.repo == nil {
		return SceneConfig{}, common.ErrInternal
	}
	normalized, auditPairs, err := s.prepareSceneMutation(ctx, scene, input)
	if err != nil {
		return SceneConfig{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.repo.db.BeginTx(ctx, nil)
	if err != nil {
		return SceneConfig{}, err
	}

	retryPolicyJSON, _ := json.Marshal(normalized.RetryOn)
	requestOptionsJSON, _ := json.Marshal(normalized.RequestOptions)
	_, err = tx.ExecContext(ctx, `
INSERT INTO ai_route_scenes (
	scene,
	enabled,
	strategy,
	max_attempts,
	retry_policy_json,
	breaker_failure_threshold,
	breaker_cooldown_seconds,
	request_options_json,
	updated_by_subject,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(scene) DO UPDATE SET
	enabled = excluded.enabled,
	strategy = excluded.strategy,
	max_attempts = excluded.max_attempts,
	retry_policy_json = excluded.retry_policy_json,
	breaker_failure_threshold = excluded.breaker_failure_threshold,
	breaker_cooldown_seconds = excluded.breaker_cooldown_seconds,
	request_options_json = excluded.request_options_json,
	updated_by_subject = excluded.updated_by_subject,
	updated_at = excluded.updated_at
`,
		string(scene),
		boolToInt(normalized.Enabled),
		string(normalized.Strategy),
		normalized.MaxAttempts,
		string(retryPolicyJSON),
		normalized.Breaker.FailureThreshold,
		normalized.Breaker.CooldownSeconds,
		string(requestOptionsJSON),
		strings.TrimSpace(subject),
		now,
	)
	if err != nil {
		_ = tx.Rollback()
		return SceneConfig{}, err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM ai_route_providers WHERE scene = ?`, string(scene)); err != nil {
		_ = tx.Rollback()
		return SceneConfig{}, err
	}

	for _, provider := range normalized.Providers {
		extraJSON, _ := json.Marshal(provider.Extra)
		_, err := tx.ExecContext(ctx, `
INSERT INTO ai_route_providers (
	id,
	scene,
	name,
	adapter,
	enabled,
	priority,
	weight,
	base_url,
	api_key_ciphertext,
	model,
	timeout_seconds,
	extra_json,
	updated_by_subject,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
			provider.ID,
			string(scene),
			provider.Name,
			provider.Adapter,
			boolToInt(provider.Enabled),
			provider.Priority,
			provider.Weight,
			provider.BaseURL,
			provider.APIKey,
			provider.Model,
			provider.TimeoutSeconds,
			string(extraJSON),
			strings.TrimSpace(subject),
			now,
		)
		if err != nil {
			_ = tx.Rollback()
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return SceneConfig{}, common.NewAppError(common.CodeBadRequest, "provider id must be globally unique", http.StatusBadRequest).WithErr(err)
			}
			return SceneConfig{}, err
		}
	}

	for _, pair := range auditPairs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO app_setting_audits (
	group_name,
	setting_key,
	action,
	old_value_masked,
	new_value_masked,
	operator_subject,
	request_id,
	created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
			pair.groupName,
			pair.settingKey,
			"update",
			pair.oldValue,
			pair.newValue,
			strings.TrimSpace(subject),
			strings.TrimSpace(requestID),
			now,
		); err != nil {
			_ = tx.Rollback()
			return SceneConfig{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return SceneConfig{}, err
	}

	s.resetRoundRobin(scene)
	return s.GetScene(ctx, scene)
}

func (s *Service) TestScene(ctx context.Context, subject, requestID string, scene Scene, input SceneConfig) (TestResult, error) {
	normalized, _, err := s.prepareSceneMutation(ctx, scene, input)
	if err != nil {
		return TestResult{}, err
	}
	if !normalized.Enabled {
		normalized.Enabled = true
	}

	result, routeErr := s.routeChat(ctx, normalized, s.sceneTestInput(scene))
	testResult := TestResult{
		OK:            routeErr == nil,
		Message:       "route test succeeded",
		FinalProvider: result.ProviderID,
		FinalModel:    result.Model,
		Attempts:      result.Attempts,
	}
	if routeErr != nil {
		testResult.Message = routeErr.Error()
	}

	if s.repo != nil && s.repo.db != nil {
		groupName := fmt.Sprintf("ai.routing.%s", scene)
		settingKey := fmt.Sprintf("ai.routing.%s.scene", scene)
		summary := testResult.Message
		if testResult.OK && testResult.FinalProvider != "" {
			summary = fmt.Sprintf("ok via %s", testResult.FinalProvider)
		}
		_, _ = s.repo.db.ExecContext(ctx, `
INSERT INTO app_setting_audits (
	group_name,
	setting_key,
	action,
	old_value_masked,
	new_value_masked,
	operator_subject,
	request_id,
	created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
			groupName,
			settingKey,
			"test",
			"",
			truncateText(summary, 180),
			strings.TrimSpace(subject),
			strings.TrimSpace(requestID),
			time.Now().UTC().Format(time.RFC3339),
		)
	}

	if routeErr != nil {
		return testResult, nil
	}
	return testResult, nil
}

func (s *Service) sceneTestInput(scene Scene) ChatCompletionInput {
	if s != nil && s.testInputBuilder != nil {
		if input, ok := s.testInputBuilder(scene); ok {
			return input
		}
	}
	return buildSceneTestInput(scene)
}

func (s *Service) RouteChat(ctx context.Context, scene Scene, input ChatCompletionInput) (ChatCompletionResult, error) {
	config, err := s.runtimeScene(ctx, scene)
	if err != nil {
		return ChatCompletionResult{}, err
	}
	return s.routeChat(ctx, config, input)
}

func (s *Service) runtimeScene(ctx context.Context, scene Scene) (SceneConfig, error) {
	if !IsValidScene(string(scene)) {
		return SceneConfig{}, common.ErrNotFound
	}

	if s != nil && s.repo != nil {
		record, providers, found, err := s.repo.loadScene(ctx, scene)
		if err != nil {
			return SceneConfig{}, err
		}
		if found {
			config, err := s.buildSceneConfig(record, providers)
			if err != nil {
				return SceneConfig{}, err
			}
			if config.Enabled && len(enabledProviders(config.Providers)) > 0 {
				return config, nil
			}
		}
	}

	return s.compatibilityScene(ctx, scene), nil
}

func (s *Service) compatibilityScene(ctx context.Context, scene Scene) SceneConfig {
	if s == nil || s.compatibility == nil {
		return defaultSceneConfig(scene)
	}
	config := s.compatibility(ctx, scene)
	if config.Scene == "" {
		config.Scene = scene
	}
	normalizeSceneConfig(&config)
	return config
}

func (s *Service) buildSceneConfig(record sceneRecord, providers []providerRecord) (SceneConfig, error) {
	config := SceneConfig{
		Scene:       record.Scene,
		Enabled:     record.Enabled,
		Strategy:    record.Strategy,
		MaxAttempts: record.MaxAttempts,
		RetryOn:     append([]string(nil), record.RetryOn...),
		Breaker: BreakerConfig{
			FailureThreshold: record.BreakerFailureThreshold,
			CooldownSeconds:  record.BreakerCooldownSeconds,
		},
		RequestOptions: record.RequestOptions,
		UpdatedBy:      record.UpdatedBy,
		UpdatedAt:      record.UpdatedAt,
		Providers:      make([]ProviderConfig, 0, len(providers)),
	}
	for _, provider := range providers {
		view := ProviderConfig{
			ID:             provider.ID,
			Scene:          provider.Scene,
			Name:           provider.Name,
			Adapter:        provider.Adapter,
			Enabled:        provider.Enabled,
			Priority:       provider.Priority,
			Weight:         provider.Weight,
			BaseURL:        strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/"),
			APIKey:         provider.APIKeyCipher,
			Model:          strings.TrimSpace(provider.Model),
			TimeoutSeconds: provider.TimeoutSeconds,
			EndpointMode:   NormalizeProviderEndpointMode(extraStringValue(provider.Extra, providerExtraKeyEndpointMode)),
			ResponseFormat: NormalizeProviderResponseFormat(extraStringValue(provider.Extra, providerExtraKeyResponseFormat)),
			Extra:          cloneProviderExtra(provider.Extra),
			UpdatedBy:      provider.UpdatedBy,
			UpdatedAt:      provider.UpdatedAt,
			HasAPIKey:      strings.TrimSpace(provider.APIKeyCipher) != "",
		}
		if view.HasAPIKey {
			plain, err := s.cipherBox.Decrypt(provider.APIKeyCipher)
			if err == nil {
				view.APIKeyMasked = maskSecret(plain)
			}
		}
		config.Providers = append(config.Providers, view)
	}
	normalizeSceneConfig(&config)
	return config, nil
}
