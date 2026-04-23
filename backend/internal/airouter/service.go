package airouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type CompatibilityLoader func(context.Context, Scene) SceneConfig
type TestInputBuilder func(Scene) (ChatCompletionInput, bool)

var (
	markdownImageURLPattern = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)\)`)
	plainURLPattern         = regexp.MustCompile(`https?://[^\s)]+`)
	dataImageURLPattern     = regexp.MustCompile(`data:image/[a-zA-Z0-9.+-]+;base64,[A-Za-z0-9+/=]+`)
)

const (
	providerExtraKeyEndpointMode   = "endpoint_mode"
	providerExtraKeyResponseFormat = "response_format"
	defaultImageOutputFormat       = "png"
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

type auditPair struct {
	groupName  string
	settingKey string
	oldValue   string
	newValue   string
}

func (s *Service) prepareSceneMutation(ctx context.Context, scene Scene, input SceneConfig) (SceneConfig, []auditPair, error) {
	if !IsValidScene(string(scene)) {
		return SceneConfig{}, nil, common.ErrNotFound
	}

	previous, _ := s.GetScene(ctx, scene)
	previousPersisted := previous
	if previousPersisted.Source != "db" {
		previousPersisted = SceneConfig{Scene: scene}
	}

	currentPersistedMap := make(map[string]ProviderConfig, len(previous.Providers))
	if s != nil && s.repo != nil {
		_, rawProviders, found, err := s.repo.loadScene(ctx, scene)
		if err != nil {
			return SceneConfig{}, nil, err
		}
		if found {
			for _, provider := range rawProviders {
				item := ProviderConfig{
					ID:             provider.ID,
					Scene:          provider.Scene,
					Name:           provider.Name,
					Enabled:        provider.Enabled,
					Priority:       provider.Priority,
					Adapter:        provider.Adapter,
					BaseURL:        provider.BaseURL,
					Model:          provider.Model,
					TimeoutSeconds: provider.TimeoutSeconds,
					EndpointMode:   NormalizeProviderEndpointMode(extraStringValue(provider.Extra, providerExtraKeyEndpointMode)),
					ResponseFormat: NormalizeProviderResponseFormat(extraStringValue(provider.Extra, providerExtraKeyResponseFormat)),
					Extra:          cloneProviderExtra(provider.Extra),
					APIKey:         provider.APIKeyCipher,
					HasAPIKey:      strings.TrimSpace(provider.APIKeyCipher) != "",
				}
				if item.HasAPIKey {
					plain, decryptErr := s.cipherBox.Decrypt(provider.APIKeyCipher)
					if decryptErr == nil {
						item.APIKeyMasked = maskSecret(plain)
					}
				}
				currentPersistedMap[item.ID] = item
			}
		}
	}
	compatibilityProviders := s.compatibilityScene(ctx, scene).Providers
	for _, provider := range compatibilityProviders {
		if _, exists := currentPersistedMap[provider.ID]; exists {
			continue
		}
		item := provider
		if item.HasAPIKey && strings.TrimSpace(item.APIKey) != "" {
			cipher, err := s.cipherBox.Encrypt(strings.TrimSpace(item.APIKey))
			if err != nil {
				return SceneConfig{}, nil, common.ErrInternal.WithErr(err)
			}
			item.APIKey = cipher
		}
		currentPersistedMap[item.ID] = item
	}

	config := input
	config.Scene = scene
	normalizeSceneConfig(&config)
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 2
	}
	if config.MaxAttempts > len(config.Providers) && len(config.Providers) > 0 {
		config.MaxAttempts = len(config.Providers)
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}
	if len(config.RetryOn) == 0 {
		config.RetryOn = DefaultRetryOn()
	}
	if !IsValidStrategy(string(config.Strategy)) {
		return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "invalid strategy", http.StatusBadRequest)
	}

	ids := make(map[string]struct{}, len(config.Providers))
	for index, provider := range config.Providers {
		provider.ID = strings.TrimSpace(provider.ID)
		if provider.ID == "" {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "provider id is required", http.StatusBadRequest)
		}
		if _, exists := ids[provider.ID]; exists {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "provider id must be unique within scene", http.StatusBadRequest)
		}
		ids[provider.ID] = struct{}{}
		provider.Scene = scene
		provider.Name = strings.TrimSpace(provider.Name)
		if provider.Name == "" {
			provider.Name = provider.ID
		}
		provider.Adapter = strings.TrimSpace(provider.Adapter)
		if provider.Adapter == "" {
			provider.Adapter = AdapterOpenAICompatible
		}
		if provider.Adapter != AdapterOpenAICompatible {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "unsupported adapter", http.StatusBadRequest)
		}
		provider.BaseURL = strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
		provider.Model = strings.TrimSpace(provider.Model)
		rawEndpointMode := strings.TrimSpace(string(input.Providers[index].EndpointMode))
		if !IsValidProviderEndpointMode(rawEndpointMode) {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "provider endpointMode is invalid", http.StatusBadRequest)
		}
		provider.EndpointMode = NormalizeProviderEndpointMode(rawEndpointMode)
		rawResponseFormat := strings.TrimSpace(string(input.Providers[index].ResponseFormat))
		if !IsValidProviderResponseFormat(rawResponseFormat) {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "provider responseFormat is invalid", http.StatusBadRequest)
		}
		provider.ResponseFormat = NormalizeProviderResponseFormat(rawResponseFormat)
		if provider.EndpointMode != EndpointModeImagesGenerations {
			provider.ResponseFormat = ResponseFormatAuto
		}
		if provider.EndpointMode == EndpointModeImagesGenerations && scene != SceneFlowchart {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "images generation endpoint is only supported for flowchart scene", http.StatusBadRequest)
		}
		if provider.Priority <= 0 {
			provider.Priority = (index + 1) * 10
		}
		if provider.Weight <= 0 {
			provider.Weight = 100
		}
		if provider.TimeoutSeconds <= 0 {
			provider.TimeoutSeconds = 30
		}
		if provider.TimeoutSeconds > 600 {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "provider timeout is too large", http.StatusBadRequest)
		}
		if provider.Enabled && provider.BaseURL == "" {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "provider baseURL is required", http.StatusBadRequest)
		}
		if provider.Enabled && provider.Model == "" {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, "provider model is required", http.StatusBadRequest)
		}

		existing := currentPersistedMap[provider.ID]
		switch {
		case provider.ClearAPIKey:
			provider.APIKey = ""
		case strings.TrimSpace(provider.APIKey) != "":
			cipher, err := s.cipherBox.Encrypt(strings.TrimSpace(provider.APIKey))
			if err != nil {
				return SceneConfig{}, nil, common.ErrInternal.WithErr(err)
			}
			provider.APIKey = cipher
			provider.APIKeyMasked = maskSecret(strings.TrimSpace(input.Providers[index].APIKey))
			provider.HasAPIKey = true
		case existing.HasAPIKey:
			provider.APIKey = existing.APIKey
			provider.APIKeyMasked = existing.APIKeyMasked
			provider.HasAPIKey = true
		default:
			provider.APIKey = ""
			provider.APIKeyMasked = ""
			provider.HasAPIKey = false
		}
		if existing.APIKey != "" && provider.APIKey == "" && !provider.ClearAPIKey {
			provider.APIKey = existing.APIKey
			provider.APIKeyMasked = existing.APIKeyMasked
			provider.HasAPIKey = true
		}
		provider.Extra = providerExtraForPersistence(provider.Extra, provider.EndpointMode, provider.ResponseFormat)
		config.Providers[index] = provider
	}

	sceneKey := fmt.Sprintf("ai.routing.%s.scene", scene)
	groupName := fmt.Sprintf("ai.routing.%s", scene)
	audits := []auditPair{
		{
			groupName:  groupName,
			settingKey: sceneKey,
			oldValue:   truncateText(sceneAuditSummary(previousPersisted), 240),
			newValue:   truncateText(sceneAuditSummary(config), 240),
		},
	}

	oldProviderMap := make(map[string]ProviderConfig, len(previousPersisted.Providers))
	for _, provider := range previousPersisted.Providers {
		oldProviderMap[provider.ID] = provider
	}
	newProviderMap := make(map[string]ProviderConfig, len(config.Providers))
	for _, provider := range config.Providers {
		newProviderMap[provider.ID] = provider
	}

	idSet := make(map[string]struct{}, len(oldProviderMap)+len(newProviderMap))
	for id := range oldProviderMap {
		idSet[id] = struct{}{}
	}
	for id := range newProviderMap {
		idSet[id] = struct{}{}
	}

	idsOrdered := make([]string, 0, len(idSet))
	for id := range idSet {
		idsOrdered = append(idsOrdered, id)
	}
	sort.Strings(idsOrdered)
	for _, id := range idsOrdered {
		oldProvider := oldProviderMap[id]
		newProvider := newProviderMap[id]
		oldSummary := providerAuditSummary(oldProvider)
		newSummary := providerAuditSummary(newProvider)
		if oldSummary == newSummary {
			continue
		}
		audits = append(audits, auditPair{
			groupName:  groupName,
			settingKey: fmt.Sprintf("ai.routing.%s.provider.%s", scene, id),
			oldValue:   truncateText(oldSummary, 240),
			newValue:   truncateText(newSummary, 240),
		})
	}

	return config, audits, nil
}

func (s *Service) routeChat(ctx context.Context, config SceneConfig, input ChatCompletionInput) (ChatCompletionResult, error) {
	normalizeSceneConfig(&config)
	providers := enabledProviders(config.Providers)
	if !config.Enabled || len(providers) == 0 {
		return ChatCompletionResult{}, common.NewAppError(common.CodeInternalServer, "ai routing is not configured for this scene", http.StatusServiceUnavailable)
	}

	order := buildAttemptOrder(config.Scene, config.Strategy, providers, s.currentRoundRobinStart(config.Scene, len(providers)))
	if len(order) == 0 {
		return ChatCompletionResult{}, common.NewAppError(common.CodeInternalServer, "ai routing has no enabled providers", http.StatusServiceUnavailable)
	}

	maxAttempts := config.MaxAttempts
	if maxAttempts <= 0 || maxAttempts > len(order) {
		maxAttempts = len(order)
	}

	now := time.Now()
	result := ChatCompletionResult{
		Strategy: config.Strategy,
		Attempts: make([]AttemptResult, 0, len(order)),
	}
	actualAttempts := 0
	var lastErr error

	for _, candidate := range order {
		if actualAttempts >= maxAttempts {
			break
		}

		if open, openUntil := s.breaker.isOpen(config.Scene, candidate.ID, now); open {
			result.Attempts = append(result.Attempts, AttemptResult{
				ProviderID:       candidate.ID,
				ProviderName:     candidate.Name,
				Model:            candidate.Model,
				Status:           audit.CallStatusFailed,
				ErrorType:        ErrorTypeBreakerOpen,
				ErrorMessage:     "provider skipped by breaker",
				SkippedByBreaker: true,
				BreakerOpenUntil: openUntil.UTC().Format(time.RFC3339),
			})
			continue
		}

		actualAttempts++
		if result.StartedProvider == "" {
			result.StartedProvider = candidate.ID
		}

		content, endpoint, httpStatus, latencyMS, callErr := s.callOpenAICompatible(ctx, config, candidate, input)
		if callErr == nil && input.ValidateContent != nil {
			callErr = normalizeValidationError(input.ValidateContent(content))
		}
		if callErr == nil {
			s.breaker.markSuccess(config.Scene, candidate.ID)
			result.Content = content
			result.ProviderID = candidate.ID
			result.ProviderName = candidate.Name
			result.Model = candidate.Model
			result.FallbackUsed = actualAttempts > 1
			result.AttemptCount = actualAttempts
			result.Attempts = append(result.Attempts, AttemptResult{
				ProviderID:   candidate.ID,
				ProviderName: candidate.Name,
				Model:        candidate.Model,
				Status:       audit.CallStatusSuccess,
				HTTPStatus:   httpStatus,
				LatencyMS:    latencyMS,
			})
			s.logCall(ctx, config, candidate, actualAttempts, endpoint, httpStatus, latencyMS, nil, input)
			s.trackProviderAlert(ctx, config, candidate, httpStatus, nil, input)
			if config.Strategy == StrategyRoundRobinFailover {
				s.setRoundRobinNext(config.Scene, candidate.originalIndex+1, len(providers))
			}
			return result, nil
		}

		s.logCall(ctx, config, candidate, actualAttempts, endpoint, httpStatus, latencyMS, callErr, input)
		s.trackProviderAlert(ctx, config, candidate, httpStatus, callErr, input)
		errorType := routeErrorType(callErr)
		attempt := AttemptResult{
			ProviderID:   candidate.ID,
			ProviderName: candidate.Name,
			Model:        candidate.Model,
			Status:       audit.CallStatusFromError(callErr),
			HTTPStatus:   httpStatus,
			LatencyMS:    latencyMS,
			ErrorType:    errorType,
			ErrorMessage: callErr.Error(),
		}
		if shouldRetry(config.RetryOn, errorType) {
			if openUntil := s.breaker.markFailure(config.Scene, candidate.ID, config.Breaker, time.Now()); !openUntil.IsZero() {
				attempt.BreakerOpenUntil = openUntil.UTC().Format(time.RFC3339)
			}
		}
		result.Attempts = append(result.Attempts, attempt)
		lastErr = callErr
		if !shouldRetry(config.RetryOn, errorType) {
			break
		}
	}

	result.AttemptCount = actualAttempts
	result.FallbackUsed = actualAttempts > 1
	if lastErr == nil {
		lastErr = common.NewAppError(common.CodeInternalServer, "all providers are cooling down", http.StatusBadGateway)
	}
	return result, lastErr
}

func (s *Service) trackProviderAlert(ctx context.Context, config SceneConfig, provider orderedProvider, httpStatus int, err error, input ChatCompletionInput) {
	if s == nil || s.alertTracker == nil || input.ContentKind == "route_test" {
		return
	}

	event := aialert.Event{
		Scene:        string(config.Scene),
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		Model:        provider.Model,
		HTTPStatus:   httpStatus,
		ErrorType:    routeErrorType(err),
		ErrorMessage: errorMessage(err),
		RequestID:    common.RequestID(ctx),
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if meta, ok := audit.CurrentRequestMeta(ctx); ok {
		event.TriggerSource = meta.TriggerSource
		event.TargetType = meta.TargetType
		event.TargetID = meta.TargetID
	}
	if err == nil {
		s.alertTracker.RecordSuccess(ctx, event)
		return
	}
	s.alertTracker.RecordFailure(ctx, event)
}

func sceneUsesCompatibility(config SceneConfig) bool {
	return !config.Enabled || len(enabledProviders(config.Providers)) == 0
}

func buildSceneTestInput(scene Scene) ChatCompletionInput {
	switch scene {
	case SceneTitle:
		return ChatCompletionInput{
			Messages: []ChatMessage{
				{
					Role:    "system",
					Content: "你是一个菜谱标题清洗助手。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\"}。",
				},
				{
					Role:    "user",
					Content: "请只返回一个 JSON，title 填写“西红柿炒鸡蛋”。",
				},
			},
			MaxTokens:       intPtr(64),
			ContentKind:     "route_test",
			ValidateContent: validateTitleTestContent,
		}
	case SceneFlowchart:
		return ChatCompletionInput{
			Messages: []ChatMessage{
				{
					Role:    "system",
					Content: "你是一个流程图生成测试助手。请生成一张最小可用的测试流程图，允许返回图片 URL、markdown 图片或 data url，不要输出额外解释。",
				},
				{
					Role:    "user",
					Content: "请生成一张“西红柿炒鸡蛋”测试流程图，内容尽量简单，只要能验证出图链路即可。",
				},
			},
			MaxTokens:       intPtr(256),
			ContentKind:     "route_test",
			ValidateContent: validateFlowchartTestContent,
		}
	default:
		return ChatCompletionInput{
			Messages: []ChatMessage{
				{
					Role:    "system",
					Content: "你是一个菜谱整理助手。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\",\"ingredient\":\"\",\"summary\":\"\",\"mainIngredients\":[],\"secondaryIngredients\":[],\"steps\":[{\"title\":\"\",\"detail\":\"\"}],\"note\":\"\"}。",
				},
				{
					Role:    "user",
					Content: "请返回一个最小可用的测试菜谱 JSON，主题是西红柿炒鸡蛋。",
				},
			},
			MaxTokens:       intPtr(1024),
			ContentKind:     "route_test",
			ValidateContent: validateSummaryTestContent,
		}
	}
}

type orderedProvider struct {
	ProviderConfig
	originalIndex int
}

func buildAttemptOrder(scene Scene, strategy Strategy, providers []ProviderConfig, start int) []orderedProvider {
	ordered := make([]orderedProvider, 0, len(providers))
	for index, provider := range providers {
		ordered = append(ordered, orderedProvider{
			ProviderConfig: provider,
			originalIndex:  index,
		})
	}
	if strategy != StrategyRoundRobinFailover || len(ordered) == 0 {
		return ordered
	}
	if start < 0 {
		start = 0
	}
	start = start % len(ordered)
	return append(ordered[start:], ordered[:start]...)
}

func (s *Service) currentRoundRobinStart(scene Scene, count int) int {
	if count <= 0 {
		return 0
	}
	s.roundRobinMu.Lock()
	defer s.roundRobinMu.Unlock()
	return s.roundRobinNext[scene] % count
}

func (s *Service) setRoundRobinNext(scene Scene, next int, count int) {
	if count <= 0 {
		return
	}
	s.roundRobinMu.Lock()
	defer s.roundRobinMu.Unlock()
	s.roundRobinNext[scene] = next % count
}

func (s *Service) resetRoundRobin(scene Scene) {
	s.roundRobinMu.Lock()
	defer s.roundRobinMu.Unlock()
	delete(s.roundRobinNext, scene)
}

type openAIChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	Stream      *bool         `json:"stream,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content json.RawMessage `json:"content"`
			Images  []struct {
				Type     string `json:"type"`
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"images"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type openAIImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Quality        string `json:"quality,omitempty"`
	OutputFormat   string `json:"output_format,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

type openAIImageGenerationResponse struct {
	Data []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *Service) callOpenAICompatible(ctx context.Context, config SceneConfig, provider orderedProvider, input ChatCompletionInput) (string, string, int, int64, error) {
	startedAt := time.Now()
	endpointMode := NormalizeProviderEndpointMode(string(provider.EndpointMode))
	endpointPath := "/chat/completions"
	var body []byte

	switch endpointMode {
	case EndpointModeImagesGenerations:
		endpointPath = "/images/generations"
		request := openAIImageGenerationRequest{
			Model:        provider.Model,
			Prompt:       buildImageGenerationPrompt(input.Messages),
			OutputFormat: defaultImageOutputFormat,
		}
		if request.Prompt == "" {
			return "", endpointPath, 0, 0, common.NewAppError(common.CodeBadRequest, "image generation prompt is required", http.StatusBadRequest)
		}
		responseFormat := NormalizeProviderResponseFormat(string(provider.ResponseFormat))
		if responseFormat != ResponseFormatAuto {
			request.ResponseFormat = string(responseFormat)
		}
		marshaled, err := json.Marshal(request)
		if err != nil {
			return "", endpointPath, 0, 0, common.ErrInternal.WithErr(err)
		}
		body = marshaled
	default:
		request := openAIChatRequest{
			Model:    provider.Model,
			Messages: input.Messages,
		}

		stream := config.RequestOptions.Stream
		if input.Stream != nil {
			stream = *input.Stream
		}
		temperature := config.RequestOptions.Temperature
		if input.Temperature != nil {
			temperature = *input.Temperature
		}
		maxTokens := config.RequestOptions.MaxTokens
		if input.MaxTokens != nil {
			maxTokens = *input.MaxTokens
		}

		if provider.Scene == SceneTitle || input.Stream != nil || config.RequestOptions.Stream {
			request.Stream = &stream
		}
		if provider.Scene == SceneTitle || input.Temperature != nil || config.RequestOptions.Temperature != 0 {
			request.Temperature = &temperature
		}
		if maxTokens > 0 {
			request.MaxTokens = &maxTokens
		}

		marshaled, err := json.Marshal(request)
		if err != nil {
			return "", endpointPath, 0, 0, common.ErrInternal.WithErr(err)
		}
		body = marshaled
	}

	timeout := time.Duration(provider.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(provider.BaseURL, "/")+endpointPath, bytes.NewReader(body))
	if err != nil {
		return "", endpointPath, 0, 0, common.ErrInternal.WithErr(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(provider.APIKey) != "" {
		plain := strings.TrimSpace(provider.APIKey)
		decrypted, decryptErr := s.cipherBox.Decrypt(provider.APIKey)
		if decryptErr == nil {
			plain = strings.TrimSpace(decrypted)
		}
		req.Header.Set("Authorization", "Bearer "+plain)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", endpointPath, 0, time.Since(startedAt).Milliseconds(), classifyRequestError(err, timeout)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), classifyHTTPError(resp.StatusCode, strings.TrimSpace(string(data)))
	}

	switch endpointMode {
	case EndpointModeImagesGenerations:
		var parsed openAIImageGenerationResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), &typedError{
				errorType:  ErrorTypeInvalidResponse,
				message:    "invalid image generation response",
				httpStatus: http.StatusBadGateway,
				cause:      err,
			}
		}
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), classifyHTTPError(resp.StatusCode, strings.TrimSpace(parsed.Error.Message))
		}
		content := extractGeneratedImageContent(parsed.Data, NormalizeProviderResponseFormat(string(provider.ResponseFormat)))
		if content == "" {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), &typedError{
				errorType:  ErrorTypeInvalidResponse,
				message:    "image generation response contained no image",
				httpStatus: http.StatusBadGateway,
			}
		}
		return content, endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), nil
	default:
		var parsed openAIChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), &typedError{
				errorType:  ErrorTypeInvalidResponse,
				message:    "invalid chat completion response",
				httpStatus: http.StatusBadGateway,
				cause:      err,
			}
		}
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), classifyHTTPError(resp.StatusCode, strings.TrimSpace(parsed.Error.Message))
		}
		if len(parsed.Choices) == 0 {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), &typedError{
				errorType:  ErrorTypeInvalidResponse,
				message:    "chat completion response contained no choices",
				httpStatus: http.StatusBadGateway,
			}
		}

		content := extractMessageContent(parsed.Choices[0].Message.Content)
		if provider.Scene == SceneFlowchart {
			if imageURL := extractMessageImageURL(parsed.Choices[0].Message.Images); imageURL != "" {
				content = imageURL
			}
		}
		if content == "" {
			return "", endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), &typedError{
				errorType:  ErrorTypeInvalidResponse,
				message:    "chat completion response was empty",
				httpStatus: http.StatusBadGateway,
			}
		}

		return content, endpointPath, resp.StatusCode, time.Since(startedAt).Milliseconds(), nil
	}
}

func (s *Service) logCall(ctx context.Context, config SceneConfig, provider orderedProvider, attempt int, endpoint string, httpStatus int, latencyMS int64, err error, input ChatCompletionInput) {
	if s == nil || s.tracker == nil {
		return
	}
	jobCtx, ok := audit.CurrentJobContext(ctx)
	if !ok || jobCtx.JobRunID <= 0 {
		return
	}

	meta := map[string]any{
		"scene":               string(config.Scene),
		"route_strategy":      string(config.Strategy),
		"attempt":             attempt,
		"provider_adapter":    provider.Adapter,
		"is_fallback_attempt": attempt > 1,
	}
	if input.ContentKind != "" {
		meta["content_kind"] = input.ContentKind
	}
	for key, value := range input.AdditionalMeta {
		meta[key] = value
	}

	status := audit.CallStatusSuccess
	if err != nil {
		status = audit.CallStatusFromError(err)
	}
	_ = s.tracker.LogCall(ctx, audit.CallLogInput{
		JobRunID:     jobCtx.JobRunID,
		Scene:        jobCtx.Scene,
		Provider:     provider.ID,
		Endpoint:     endpoint,
		Model:        provider.Model,
		Status:       status,
		HTTPStatus:   httpStatus,
		LatencyMS:    latencyMS,
		ErrorType:    routeErrorType(err),
		ErrorMessage: errorMessage(err),
		RequestID:    common.RequestID(ctx),
		Meta:         meta,
	})
}

func extractMessageContent(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		lines := make([]string, 0, len(parts))
		for _, part := range parts {
			if strings.TrimSpace(part.Text) == "" {
				continue
			}
			lines = append(lines, strings.TrimSpace(part.Text))
		}
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}

	return strings.TrimSpace(string(raw))
}

func buildImageGenerationPrompt(messages []ChatMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		text := strings.TrimSpace(message.Content)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func extractGeneratedImageContent(items []struct {
	URL     string `json:"url"`
	B64JSON string `json:"b64_json"`
}, responseFormat ProviderResponseFormat) string {
	for _, item := range items {
		url := normalizeImageReference(item.URL)
		b64 := strings.TrimSpace(item.B64JSON)
		switch responseFormat {
		case ResponseFormatImageURL:
			if url != "" {
				return url
			}
		case ResponseFormatB64JSON:
			if b64 != "" {
				return "data:image/" + defaultImageOutputFormat + ";base64," + b64
			}
		default:
			if url != "" {
				return url
			}
			if b64 != "" {
				return "data:image/" + defaultImageOutputFormat + ";base64," + b64
			}
		}
	}
	return ""
}

func extractMessageImageURL(images []struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}) string {
	for _, image := range images {
		if value := normalizeImageReference(image.ImageURL.URL); value != "" {
			return value
		}
	}
	return ""
}

func classifyRequestError(err error, timeout time.Duration) error {
	if audit.IsTimeoutError(err) {
		message := "upstream request timed out"
		if timeout > 0 {
			message = fmt.Sprintf("%s after %s", message, timeout.Round(time.Second))
		}
		return &typedError{
			errorType:  ErrorTypeTimeout,
			message:    message,
			httpStatus: http.StatusBadGateway,
			cause:      err,
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return &typedError{
			errorType:  ErrorTypeNetwork,
			message:    "network error while calling upstream",
			httpStatus: http.StatusBadGateway,
			cause:      err,
		}
	}

	return &typedError{
		errorType:  ErrorTypeUnknown,
		message:    "request to upstream failed",
		httpStatus: http.StatusBadGateway,
		cause:      err,
	}
}

func classifyHTTPError(status int, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = fmt.Sprintf("upstream returned status %d", status)
	}

	switch {
	case status == http.StatusTooManyRequests:
		return &typedError{errorType: ErrorTypeRateLimit, message: message, httpStatus: http.StatusBadGateway}
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return &typedError{errorType: ErrorTypeAuth, message: message, httpStatus: http.StatusBadGateway}
	case status >= 500:
		return &typedError{errorType: ErrorTypeUpstream, message: message, httpStatus: http.StatusBadGateway}
	case status >= 400:
		return &typedError{errorType: ErrorTypeBadRequest, message: message, httpStatus: http.StatusBadGateway}
	default:
		return &typedError{errorType: ErrorTypeUnknown, message: message, httpStatus: http.StatusBadGateway}
	}
}

func normalizeValidationError(err error) error {
	if err == nil {
		return nil
	}

	type typed interface {
		AuditErrorType() string
	}
	var typedErr typed
	if errors.As(err, &typedErr) && strings.TrimSpace(typedErr.AuditErrorType()) != "" {
		return err
	}

	var appErr *common.AppError
	if errors.As(err, &appErr) {
		return &typedError{
			errorType:  ErrorTypeBusiness,
			message:    strings.TrimSpace(appErr.Message),
			httpStatus: appErr.HTTPStatus,
			cause:      err,
		}
	}

	return &typedError{
		errorType:  ErrorTypeInvalidResponse,
		message:    truncateText(strings.TrimSpace(err.Error()), 180),
		httpStatus: http.StatusBadGateway,
		cause:      err,
	}
}

func validateSummaryTestContent(content string) error {
	content = trimCodeFenceContent(content)
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("summary response was empty")
	}
	var payload struct {
		Title string `json:"title"`
		Steps []struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Title) == "" {
		return fmt.Errorf("summary title is required")
	}
	if len(payload.Steps) == 0 {
		return fmt.Errorf("summary steps are required")
	}
	for _, step := range payload.Steps {
		if strings.TrimSpace(step.Title) == "" || strings.TrimSpace(step.Detail) == "" {
			return fmt.Errorf("summary steps must contain title and detail")
		}
	}
	return nil
}

func validateTitleTestContent(content string) error {
	content = trimCodeFenceContent(content)
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("title response was empty")
	}
	var payload struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return err
	}
	if strings.TrimSpace(payload.Title) == "" {
		return fmt.Errorf("title is required")
	}
	return nil
}

func validateFlowchartTestContent(content string) error {
	content = trimCodeFenceContent(content)
	if extractImageURL(content) == "" {
		return fmt.Errorf("flowchart response did not contain an image url")
	}
	return nil
}

func trimCodeFenceContent(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return content
	}
	start := 0
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
		start = 1
	}
	end := len(lines)
	if end > start && strings.HasPrefix(strings.TrimSpace(lines[end-1]), "```") {
		end--
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func extractImageURL(content string) string {
	if matches := markdownImageURLPattern.FindStringSubmatch(content); len(matches) == 2 {
		if value := normalizeImageReference(matches[1]); value != "" {
			return value
		}
	}
	if dataURL := dataImageURLPattern.FindString(content); dataURL != "" {
		return normalizeImageReference(dataURL)
	}
	for _, candidate := range plainURLPattern.FindAllString(content, -1) {
		if value := normalizeImageReference(candidate); value != "" {
			return value
		}
	}
	return ""
}

func normalizeImageReference(value string) string {
	value = strings.TrimSpace(strings.TrimRight(value, "])}>.,;!\"'"))
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "data:image/"):
		return value
	default:
		return ""
	}
}

func extraStringValue(extra map[string]any, key string) string {
	if len(extra) == 0 {
		return ""
	}
	value, ok := extra[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func cloneProviderExtra(extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(extra))
	for key, value := range extra {
		cloned[key] = value
	}
	return cloned
}

func providerExtraForPersistence(extra map[string]any, endpointMode ProviderEndpointMode, responseFormat ProviderResponseFormat) map[string]any {
	cloned := cloneProviderExtra(extra)
	if cloned == nil {
		cloned = make(map[string]any, 2)
	}
	if endpointMode == EndpointModeChatCompletions {
		delete(cloned, providerExtraKeyEndpointMode)
	} else {
		cloned[providerExtraKeyEndpointMode] = string(endpointMode)
	}
	if responseFormat == ResponseFormatAuto {
		delete(cloned, providerExtraKeyResponseFormat)
	} else {
		cloned[providerExtraKeyResponseFormat] = string(responseFormat)
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func enabledProviders(items []ProviderConfig) []ProviderConfig {
	enabled := make([]ProviderConfig, 0, len(items))
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		if strings.TrimSpace(item.BaseURL) == "" || strings.TrimSpace(item.Model) == "" {
			continue
		}
		enabled = append(enabled, item)
	}
	sort.SliceStable(enabled, func(i, j int) bool {
		if enabled[i].Priority != enabled[j].Priority {
			return enabled[i].Priority < enabled[j].Priority
		}
		return enabled[i].ID < enabled[j].ID
	})
	return enabled
}

func normalizeSceneConfig(config *SceneConfig) {
	if config == nil {
		return
	}
	if config.Scene == "" {
		config.Scene = SceneSummary
	}
	if !IsValidStrategy(string(config.Strategy)) {
		config.Strategy = StrategyPriorityFailover
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 2
	}
	if len(config.RetryOn) == 0 {
		config.RetryOn = DefaultRetryOn()
	}
	config.RetryOn = normalizeRetryOn(config.RetryOn)
	if config.Breaker.FailureThreshold <= 0 {
		config.Breaker.FailureThreshold = DefaultBreakerConfig().FailureThreshold
	}
	if config.Breaker.CooldownSeconds <= 0 {
		config.Breaker.CooldownSeconds = DefaultBreakerConfig().CooldownSeconds
	}
	for index := range config.Providers {
		config.Providers[index].BaseURL = strings.TrimRight(strings.TrimSpace(config.Providers[index].BaseURL), "/")
		config.Providers[index].Model = strings.TrimSpace(config.Providers[index].Model)
		if config.Providers[index].Adapter == "" {
			config.Providers[index].Adapter = AdapterOpenAICompatible
		}
		config.Providers[index].EndpointMode = NormalizeProviderEndpointMode(string(config.Providers[index].EndpointMode))
		config.Providers[index].ResponseFormat = NormalizeProviderResponseFormat(string(config.Providers[index].ResponseFormat))
		if config.Providers[index].EndpointMode != EndpointModeImagesGenerations {
			config.Providers[index].ResponseFormat = ResponseFormatAuto
		}
		config.Providers[index].Extra = providerExtraForPersistence(
			config.Providers[index].Extra,
			config.Providers[index].EndpointMode,
			config.Providers[index].ResponseFormat,
		)
		if config.Providers[index].Priority <= 0 {
			config.Providers[index].Priority = (index + 1) * 10
		}
		if config.Providers[index].TimeoutSeconds <= 0 {
			config.Providers[index].TimeoutSeconds = 30
		}
	}
	sort.SliceStable(config.Providers, func(i, j int) bool {
		if config.Providers[i].Priority != config.Providers[j].Priority {
			return config.Providers[i].Priority < config.Providers[j].Priority
		}
		return config.Providers[i].ID < config.Providers[j].ID
	})
}

func defaultSceneConfig(scene Scene) SceneConfig {
	config := SceneConfig{
		Scene:       scene,
		Enabled:     false,
		Strategy:    StrategyPriorityFailover,
		MaxAttempts: 2,
		RetryOn:     DefaultRetryOn(),
		Breaker:     DefaultBreakerConfig(),
		Providers:   []ProviderConfig{},
		Source:      "empty",
	}
	if scene == SceneTitle {
		config.RequestOptions.MaxTokens = 64
	}
	return config
}

func sceneAuditSummary(config SceneConfig) string {
	if config.Scene == "" {
		return ""
	}
	return fmt.Sprintf(
		"enabled=%t strategy=%s maxAttempts=%d retryOn=%s breaker=%d/%ds providers=%d requestOptions=%s",
		config.Enabled,
		config.Strategy,
		config.MaxAttempts,
		strings.Join(config.RetryOn, ","),
		config.Breaker.FailureThreshold,
		config.Breaker.CooldownSeconds,
		len(config.Providers),
		requestOptionsSummary(config.RequestOptions),
	)
}

func providerAuditSummary(provider ProviderConfig) string {
	if provider.ID == "" {
		return ""
	}
	apiKey := ""
	if provider.HasAPIKey {
		apiKey = provider.APIKeyMasked
		if apiKey == "" {
			apiKey = "****"
		}
	}
	return fmt.Sprintf(
		"name=%s enabled=%t priority=%d adapter=%s baseURL=%s model=%s timeout=%ds endpoint=%s responseFormat=%s apiKey=%s",
		provider.Name,
		provider.Enabled,
		provider.Priority,
		provider.Adapter,
		provider.BaseURL,
		provider.Model,
		provider.TimeoutSeconds,
		provider.EndpointMode,
		provider.ResponseFormat,
		apiKey,
	)
}

func requestOptionsSummary(options RequestOptions) string {
	parts := make([]string, 0, 3)
	if options.Stream {
		parts = append(parts, "stream=true")
	}
	if options.Temperature != 0 {
		parts = append(parts, fmt.Sprintf("temperature=%g", options.Temperature))
	}
	if options.MaxTokens > 0 {
		parts = append(parts, fmt.Sprintf("maxTokens=%d", options.MaxTokens))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ",")
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func shouldRetry(retryOn []string, errorType string) bool {
	for _, item := range retryOn {
		if strings.TrimSpace(item) == strings.TrimSpace(errorType) {
			return true
		}
	}
	return false
}

func routeErrorType(err error) string {
	if err == nil {
		return ""
	}
	type typed interface {
		AuditErrorType() string
	}
	var typedErr typed
	if errors.As(err, &typedErr) {
		return strings.TrimSpace(typedErr.AuditErrorType())
	}
	return audit.ErrorTypeFromError(err)
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func intPtr(value int) *int {
	return &value
}
