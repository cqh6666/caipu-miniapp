package appsettings

import (
	"context"
	"sync"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
)

type runtimeFieldDefinition struct {
	Group             string
	Key               string
	Label             string
	Description       string
	ValueType         string
	IsSecret          bool
	IsRestartRequired bool
	DefaultValue      string
}

func (p *RuntimeProvider) ConfigureCredentialKeys(secret, version string, previous []credentialcipher.Key) error {
	box, err := newVersionedCipherBox(secret, version, previous)
	if err != nil {
		return err
	}
	p.cipherBox = box
	return nil
}

type runtimeGroupDefinition struct {
	Name            string
	Title           string
	Description     string
	HiddenFromAdmin bool
	Fields          []runtimeFieldDefinition
}

type RuntimeProvider struct {
	repo             *Repository
	cipherBox        *cipherBox
	groups           []runtimeGroupDefinition
	groupIndex       map[string]runtimeGroupDefinition
	fieldIndex       map[string]runtimeFieldDefinition
	mu               sync.RWMutex
	cachedAt         time.Time
	cachedSettings   map[string]runtimeSettingRecord
	bilibiliVerifier func(context.Context, string) error
	alertSender      aialert.Sender
}

func NewRuntimeProvider(repo *Repository, secret string, cfg config.Config) *RuntimeProvider {
	groups := buildRuntimeGroups(cfg)
	groupIndex := make(map[string]runtimeGroupDefinition, len(groups))
	fieldIndex := make(map[string]runtimeFieldDefinition)
	for _, group := range groups {
		if !group.HiddenFromAdmin {
			groupIndex[group.Name] = group
		}
		for _, field := range group.Fields {
			fieldIndex[field.Group+"."+field.Key] = field
		}
	}

	return &RuntimeProvider{
		repo:           repo,
		cipherBox:      newCipherBox(secret),
		groups:         groups,
		groupIndex:     groupIndex,
		fieldIndex:     fieldIndex,
		cachedSettings: make(map[string]runtimeSettingRecord),
	}
}

func (p *RuntimeProvider) SetBilibiliVerifier(verify func(context.Context, string) error) {
	p.bilibiliVerifier = verify
}

func (p *RuntimeProvider) SetAIAlertSender(sender aialert.Sender) {
	p.alertSender = sender
}

func (p *RuntimeProvider) SummaryAI(ctx context.Context) SummaryAIConfig {
	return SummaryAIConfig{
		BaseURL: p.getString(ctx, "ai.summary.base_url"),
		APIKey:  p.getString(ctx, "ai.summary.api_key"),
		Model:   p.getString(ctx, "ai.summary.model"),
		Timeout: time.Duration(p.getInt(ctx, "ai.summary.timeout_seconds", 30)) * time.Second,
	}
}

func (p *RuntimeProvider) FlowchartAI(ctx context.Context) FlowchartAIConfig {
	return FlowchartAIConfig{
		BaseURL:        p.getString(ctx, "ai.flowchart.base_url"),
		APIKey:         p.getString(ctx, "ai.flowchart.api_key"),
		Model:          p.getString(ctx, "ai.flowchart.model"),
		EndpointMode:   p.getString(ctx, "ai.flowchart.endpoint_mode"),
		ResponseFormat: p.getString(ctx, "ai.flowchart.response_format"),
		Timeout:        time.Duration(p.getInt(ctx, "ai.flowchart.timeout_seconds", 45)) * time.Second,
	}
}

func (p *RuntimeProvider) TitleAI(ctx context.Context) TitleAIConfig {
	return TitleAIConfig{
		Enabled:     p.getBool(ctx, "ai.title.enabled", false),
		BaseURL:     p.getString(ctx, "ai.title.base_url"),
		APIKey:      p.getString(ctx, "ai.title.api_key"),
		Model:       p.getString(ctx, "ai.title.model"),
		Stream:      p.getBool(ctx, "ai.title.stream", false),
		Temperature: p.getFloat(ctx, "ai.title.temperature", 0),
		MaxTokens:   p.getInt(ctx, "ai.title.max_tokens", 64),
		Timeout:     time.Duration(p.getInt(ctx, "ai.title.timeout_seconds", 3)) * time.Second,
	}
}

func (p *RuntimeProvider) AIProviderAlert(ctx context.Context) aialert.Config {
	return aialert.Config{
		Enabled:           p.getBool(ctx, "ai.provider_alert.enabled", false),
		FailureThreshold:  p.getInt(ctx, "ai.provider_alert.failure_threshold", 3),
		ActiveWindowHours: p.getInt(ctx, "ai.provider_alert.active_window_hours", 72),
		SMTPHost:          p.getString(ctx, "ai.provider_alert.smtp_host"),
		SMTPPort:          p.getInt(ctx, "ai.provider_alert.smtp_port", 587),
		SMTPUsername:      p.getString(ctx, "ai.provider_alert.smtp_username"),
		SMTPPassword:      p.getString(ctx, "ai.provider_alert.smtp_password"),
		FromEmail:         p.getString(ctx, "ai.provider_alert.from_email"),
		ToEmails:          p.getString(ctx, "ai.provider_alert.to_emails"),
	}
}

func (p *RuntimeProvider) LinkparseSidecar(ctx context.Context) LinkparseSidecarConfig {
	return LinkparseSidecarConfig{
		Enabled: p.getBool(ctx, "sidecar.linkparse.enabled", false),
		BaseURL: p.getString(ctx, "sidecar.linkparse.base_url"),
		APIKey:  p.getString(ctx, "sidecar.linkparse.api_key"),
		Timeout: time.Duration(p.getInt(ctx, "sidecar.linkparse.timeout_seconds", 150)) * time.Second,
	}
}

func (p *RuntimeProvider) MiniProgramFeatures(ctx context.Context) MiniProgramFeatureConfig {
	return MiniProgramFeatureConfig{
		DietAssistantEnabled: p.getBool(ctx, "miniapp.features.diet_assistant_enabled", false),
	}
}

func (p *RuntimeProvider) ListRuntimeGroups(ctx context.Context) ([]RuntimeSettingGroupView, error) {
	settings, err := p.loadSettings(ctx)
	if err != nil {
		return nil, err
	}

	groups := make([]RuntimeSettingGroupView, 0, len(p.groupIndex))
	for _, group := range p.groups {
		if group.HiddenFromAdmin {
			continue
		}
		view := RuntimeSettingGroupView{
			Name:        group.Name,
			Title:       group.Title,
			Description: group.Description,
			Fields:      make([]RuntimeSettingFieldView, 0, len(group.Fields)),
		}
		for _, field := range group.Fields {
			view.Fields = append(view.Fields, p.buildFieldView(field, settings))
		}
		groups = append(groups, view)
	}
	return groups, nil
}

func (p *RuntimeProvider) GetRuntimeGroup(ctx context.Context, groupName string) (RuntimeSettingGroupView, error) {
	groups, err := p.ListRuntimeGroups(ctx)
	if err != nil {
		return RuntimeSettingGroupView{}, err
	}
	for _, group := range groups {
		if group.Name == groupName {
			return group, nil
		}
	}
	return RuntimeSettingGroupView{}, common.ErrNotFound
}
