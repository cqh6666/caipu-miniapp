package airouter

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

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
					ID:              provider.ID,
					Scene:           provider.Scene,
					Name:            provider.Name,
					Enabled:         provider.Enabled,
					Priority:        provider.Priority,
					Adapter:         provider.Adapter,
					BaseURL:         provider.BaseURL,
					Model:           provider.Model,
					TimeoutSeconds:  provider.TimeoutSeconds,
					EndpointMode:    NormalizeProviderEndpointMode(extraStringValue(provider.Extra, providerExtraKeyEndpointMode)),
					ResponseFormat:  NormalizeProviderResponseFormat(extraStringValue(provider.Extra, providerExtraKeyResponseFormat)),
					Extra:           cloneProviderExtra(provider.Extra),
					APIKey:          provider.APIKeyCipher,
					apiKeyEncrypted: strings.TrimSpace(provider.APIKeyCipher) != "",
					HasAPIKey:       strings.TrimSpace(provider.APIKeyCipher) != "",
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
			item.apiKeyEncrypted = true
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
		if err := ValidateProviderExtra(provider.Extra, provider.EndpointMode); err != nil {
			return SceneConfig{}, nil, common.NewAppError(common.CodeBadRequest, err.Error(), http.StatusBadRequest).WithErr(err)
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
			provider.apiKeyEncrypted = false
		case strings.TrimSpace(provider.APIKey) != "":
			cipher, err := s.cipherBox.Encrypt(strings.TrimSpace(provider.APIKey))
			if err != nil {
				return SceneConfig{}, nil, common.ErrInternal.WithErr(err)
			}
			provider.APIKey = cipher
			provider.apiKeyEncrypted = true
			provider.APIKeyMasked = maskSecret(strings.TrimSpace(input.Providers[index].APIKey))
			provider.HasAPIKey = true
		case existing.HasAPIKey:
			provider.APIKey = existing.APIKey
			provider.apiKeyEncrypted = existing.apiKeyEncrypted
			provider.APIKeyMasked = existing.APIKeyMasked
			provider.HasAPIKey = true
		default:
			provider.APIKey = ""
			provider.apiKeyEncrypted = false
			provider.APIKeyMasked = ""
			provider.HasAPIKey = false
		}
		if existing.APIKey != "" && provider.APIKey == "" && !provider.ClearAPIKey {
			provider.APIKey = existing.APIKey
			provider.apiKeyEncrypted = existing.apiKeyEncrypted
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
