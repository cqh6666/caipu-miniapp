package app

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

type linkParseRuntimeProvider interface {
	SummaryAI(context.Context) appsettings.SummaryAIConfig
	TitleAI(context.Context) appsettings.TitleAIConfig
	LinkparseSidecar(context.Context) appsettings.LinkparseSidecarConfig
}

type aiCompatibilityRuntimeProvider interface {
	SummaryAI(context.Context) appsettings.SummaryAIConfig
	TitleAI(context.Context) appsettings.TitleAIConfig
	FlowchartAI(context.Context) appsettings.FlowchartAIConfig
}

type flowchartRuntimeProvider interface {
	FlowchartAI(context.Context) appsettings.FlowchartAIConfig
}

type appSettingsServiceRef struct {
	service *appsettings.Service
	logger  *slog.Logger
}

func (r *appSettingsServiceRef) bilibiliSessdata(ctx context.Context) string {
	if r == nil || r.service == nil {
		return ""
	}
	sessdata, err := r.service.CurrentBilibiliSessdata(ctx)
	if err != nil {
		if r.logger != nil {
			r.logger.Warn("failed to load bilibili sessdata", "err", err)
		}
		return ""
	}
	return sessdata
}

func newLinkParseService(
	cfg config.Config,
	runtimeProvider linkParseRuntimeProvider,
	aiRouter *airouter.Service,
	tracker audit.Tracker,
	sessdataProvider func(context.Context) string,
) *linkparse.Service {
	return linkparse.NewService(linkparse.Options{
		AIBaseURL:                cfg.AIBaseURL,
		AIAPIKey:                 cfg.AIAPIKey,
		AIModel:                  cfg.AIModel,
		AITimeout:                time.Duration(cfg.AITimeoutSeconds) * time.Second,
		AITitleEnabled:           cfg.AITitleEnabled,
		AITitleBaseURL:           cfg.AITitleBaseURL,
		AITitleAPIKey:            cfg.AITitleAPIKey,
		AITitleModel:             cfg.AITitleModel,
		AITitleStream:            cfg.AITitleStream,
		AITitleTemperature:       cfg.AITitleTemperature,
		AITitleMaxTokens:         cfg.AITitleMaxTokens,
		AITitleTimeout:           time.Duration(cfg.AITitleTimeoutSeconds) * time.Second,
		LinkparseSidecarEnabled:  cfg.LinkparseSidecarEnabled,
		LinkparseSidecarBaseURL:  cfg.LinkparseSidecarBaseURL,
		LinkparseSidecarTimeout:  time.Duration(cfg.LinkparseSidecarTimeoutSec) * time.Second,
		LinkparseSidecarAPIKey:   cfg.LinkparseSidecarAPIKey,
		RuntimeConfigLoader:      buildLinkParseRuntimeConfigLoader(runtimeProvider),
		AIRouter:                 aiRouter,
		Tracker:                  tracker,
		BilibiliSessdataProvider: sessdataProvider,
	})
}

func buildLinkParseRuntimeConfigLoader(provider linkParseRuntimeProvider) linkparse.RuntimeConfigLoader {
	return func(ctx context.Context) linkparse.RuntimeConfig {
		summary := provider.SummaryAI(ctx)
		title := provider.TitleAI(ctx)
		sidecar := provider.LinkparseSidecar(ctx)
		return linkparse.RuntimeConfig{
			SummaryAI: linkparse.SummaryAIConfig{
				BaseURL: summary.BaseURL,
				APIKey:  summary.APIKey,
				Model:   summary.Model,
				Timeout: summary.Timeout,
			},
			TitleAI: linkparse.TitleAIConfig{
				Enabled:     title.Enabled,
				BaseURL:     title.BaseURL,
				APIKey:      title.APIKey,
				Model:       title.Model,
				Stream:      title.Stream,
				Temperature: title.Temperature,
				MaxTokens:   title.MaxTokens,
				Timeout:     title.Timeout,
			},
			LinkparseSidecar: linkparse.LinkparseSidecarConfig{
				Enabled: sidecar.Enabled,
				BaseURL: sidecar.BaseURL,
				APIKey:  sidecar.APIKey,
				Timeout: sidecar.Timeout,
			},
		}
	}
}

func newRecipeFlowchartGenerator(
	cfg config.Config,
	runtimeProvider flowchartRuntimeProvider,
	aiRouter *airouter.Service,
	tracker audit.Tracker,
	uploadService *upload.Service,
) *recipe.FlowchartGenerator {
	return recipe.NewFlowchartGenerator(recipe.FlowchartOptions{
		BaseURL:             cfg.AIFlowchartBaseURL,
		APIKey:              cfg.AIFlowchartAPIKey,
		Model:               cfg.AIFlowchartModel,
		EndpointMode:        cfg.AIFlowchartEndpointMode,
		ResponseFormat:      cfg.AIFlowchartResponseFormat,
		Timeout:             time.Duration(cfg.AIFlowchartTimeoutSeconds) * time.Second,
		RuntimeConfigLoader: buildFlowchartRuntimeConfigLoader(runtimeProvider),
		AIRouter:            aiRouter,
		Tracker:             tracker,
	}, uploadService)
}

func buildFlowchartRuntimeConfigLoader(provider flowchartRuntimeProvider) recipe.RuntimeConfigLoader {
	return func(ctx context.Context) recipe.FlowchartRuntimeConfig {
		flowchart := provider.FlowchartAI(ctx)
		return recipe.FlowchartRuntimeConfig{
			BaseURL:        flowchart.BaseURL,
			APIKey:         flowchart.APIKey,
			Model:          flowchart.Model,
			EndpointMode:   flowchart.EndpointMode,
			ResponseFormat: flowchart.ResponseFormat,
			Timeout:        flowchart.Timeout,
		}
	}
}

func buildAIRoutingCompatibilityLoader(runtimeProvider aiCompatibilityRuntimeProvider) airouter.CompatibilityLoader {
	return func(ctx context.Context, scene airouter.Scene) airouter.SceneConfig {
		switch scene {
		case airouter.SceneSummary:
			summary := runtimeProvider.SummaryAI(ctx)
			enabled := summary.BaseURL != "" && summary.Model != ""
			return airouter.SceneConfig{
				Scene:       scene,
				Enabled:     enabled,
				Strategy:    airouter.StrategyPriorityFailover,
				MaxAttempts: 1,
				RetryOn:     airouter.DefaultRetryOn(),
				Breaker:     airouter.DefaultBreakerConfig(),
				Providers: []airouter.ProviderConfig{
					{
						ID:             "summary-compat",
						Name:           "兼容单节点",
						Adapter:        airouter.AdapterOpenAICompatible,
						Enabled:        enabled,
						Priority:       10,
						BaseURL:        summary.BaseURL,
						APIKey:         summary.APIKey,
						APIKeyMasked:   maskCompatSecret(summary.APIKey),
						HasAPIKey:      strings.TrimSpace(summary.APIKey) != "",
						Model:          summary.Model,
						TimeoutSeconds: int(summary.Timeout.Seconds()),
					},
				},
			}
		case airouter.SceneTitle:
			summary := runtimeProvider.SummaryAI(ctx)
			title := runtimeProvider.TitleAI(ctx)
			if strings.TrimSpace(title.BaseURL) == "" {
				title.BaseURL = summary.BaseURL
			}
			if strings.TrimSpace(title.APIKey) == "" {
				title.APIKey = summary.APIKey
			}
			if strings.TrimSpace(title.Model) == "" {
				title.Model = summary.Model
			}
			enabled := title.Enabled && title.BaseURL != "" && title.Model != ""
			return airouter.SceneConfig{
				Scene:       scene,
				Enabled:     enabled,
				Strategy:    airouter.StrategyRoundRobinFailover,
				MaxAttempts: 1,
				RetryOn:     airouter.DefaultRetryOn(),
				Breaker:     airouter.DefaultBreakerConfig(),
				RequestOptions: airouter.RequestOptions{
					Stream:      title.Stream,
					Temperature: title.Temperature,
					MaxTokens:   title.MaxTokens,
				},
				Providers: []airouter.ProviderConfig{
					{
						ID:             "title-compat",
						Name:           "兼容单节点",
						Adapter:        airouter.AdapterOpenAICompatible,
						Enabled:        enabled,
						Priority:       10,
						BaseURL:        title.BaseURL,
						APIKey:         title.APIKey,
						APIKeyMasked:   maskCompatSecret(title.APIKey),
						HasAPIKey:      strings.TrimSpace(title.APIKey) != "",
						Model:          title.Model,
						TimeoutSeconds: int(title.Timeout.Seconds()),
					},
				},
			}
		case airouter.SceneFlowchart:
			flowchart := runtimeProvider.FlowchartAI(ctx)
			enabled := flowchart.BaseURL != "" && flowchart.Model != ""
			return airouter.SceneConfig{
				Scene:       scene,
				Enabled:     enabled,
				Strategy:    airouter.StrategyPriorityFailover,
				MaxAttempts: 1,
				RetryOn:     airouter.DefaultRetryOn(),
				Breaker:     airouter.DefaultBreakerConfig(),
				Providers: []airouter.ProviderConfig{
					{
						ID:             "flowchart-compat",
						Name:           "兼容单节点",
						Adapter:        airouter.AdapterOpenAICompatible,
						Enabled:        enabled,
						Priority:       10,
						BaseURL:        flowchart.BaseURL,
						APIKey:         flowchart.APIKey,
						APIKeyMasked:   maskCompatSecret(flowchart.APIKey),
						HasAPIKey:      strings.TrimSpace(flowchart.APIKey) != "",
						Model:          flowchart.Model,
						TimeoutSeconds: int(flowchart.Timeout.Seconds()),
						EndpointMode:   airouter.NormalizeProviderEndpointMode(flowchart.EndpointMode),
						ResponseFormat: airouter.NormalizeProviderResponseFormat(flowchart.ResponseFormat),
					},
				},
			}
		default:
			return airouter.SceneConfig{
				Scene:       scene,
				Strategy:    airouter.StrategyPriorityFailover,
				MaxAttempts: 1,
				RetryOn:     airouter.DefaultRetryOn(),
				Breaker:     airouter.DefaultBreakerConfig(),
			}
		}
	}
}

func buildAIRoutingTestInputBuilder() airouter.TestInputBuilder {
	return func(scene airouter.Scene) (airouter.ChatCompletionInput, bool) {
		switch scene {
		case airouter.SceneSummary:
			return linkparse.BuildSummaryRouteTestInput(), true
		case airouter.SceneTitle:
			return linkparse.BuildTitleRouteTestInput(), true
		case airouter.SceneFlowchart:
			return recipe.BuildFlowchartRouteTestInput(), true
		default:
			return airouter.ChatCompletionInput{}, false
		}
	}
}

func maskCompatSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}
