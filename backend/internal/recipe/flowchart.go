package recipe

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

type FlowchartOptions struct {
	BaseURL             string
	APIKey              string
	Model               string
	EndpointMode        string
	ResponseFormat      string
	Timeout             time.Duration
	RuntimeConfigLoader RuntimeConfigLoader
	AIRouter            *airouter.Service
	Tracker             audit.Tracker
}

type FlowchartGenerator struct {
	defaultConfig FlowchartRuntimeConfig
	configLoader  RuntimeConfigLoader
	aiRouter      *airouter.Service
	tracker       audit.Tracker
	uploader      *upload.Service
}

type FlowchartResult struct {
	ImageURL        string
	SourceHash      string
	Provider        string
	Model           string
	FallbackUsed    bool
	AttemptCount    int
	StartedProvider string
	RouteStrategy   string
}

type RuntimeConfigLoader func(context.Context) FlowchartRuntimeConfig

type FlowchartRuntimeConfig struct {
	BaseURL        string
	APIKey         string
	Model          string
	EndpointMode   string
	ResponseFormat string
	Timeout        time.Duration
}

func NewFlowchartGenerator(opts FlowchartOptions, uploader *upload.Service) *FlowchartGenerator {
	if uploader == nil {
		return nil
	}

	return &FlowchartGenerator{
		defaultConfig: FlowchartRuntimeConfig{
			BaseURL:        strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
			APIKey:         strings.TrimSpace(opts.APIKey),
			Model:          strings.TrimSpace(opts.Model),
			EndpointMode:   strings.TrimSpace(opts.EndpointMode),
			ResponseFormat: strings.TrimSpace(opts.ResponseFormat),
			Timeout:        opts.Timeout,
		},
		configLoader: opts.RuntimeConfigLoader,
		aiRouter:     opts.AIRouter,
		tracker:      opts.Tracker,
		uploader:     uploader,
	}
}

func (g *FlowchartGenerator) IsConfigured() bool {
	if g == nil || g.uploader == nil {
		return false
	}
	if g.aiRouter != nil && g.aiRouter.IsSceneAvailable(context.Background(), airouter.SceneFlowchart) {
		return true
	}
	return g.clientFor(context.Background()) != nil
}

func (g *FlowchartGenerator) Generate(ctx context.Context, item Recipe) (FlowchartResult, error) {
	if !g.IsConfigured() {
		return FlowchartResult{}, common.NewAppError(common.CodeInternalServer, "flowchart generation is not configured", http.StatusServiceUnavailable)
	}

	input, err := buildFlowchartPromptInput(item)
	if err != nil {
		return FlowchartResult{}, err
	}

	prompt := buildFlowchartPrompt(input)

	var content string
	result := FlowchartResult{}
	if g.aiRouter != nil {
		routeResult, routeErr := g.aiRouter.RouteChat(ctx, airouter.SceneFlowchart, airouter.ChatCompletionInput{
			Messages: []airouter.ChatMessage{
				{
					Role:    "system",
					Content: "你是一个料理流程图生成助手。请严格按用户要求生成手绘风格料理流程信息图，不要输出额外解释。",
				},
				{
					Role:    "user",
					Content: prompt,
				},
			},
			Temperature:    floatPtr(0.4),
			ContentKind:    "flowchart",
			AdditionalMeta: map[string]any{"content_kind": "flowchart"},
			ValidateContent: func(content string) error {
				if extractFlowchartImageURL(content) == "" {
					return fmt.Errorf("flowchart generation did not return an image")
				}
				return nil
			},
		})
		result.Provider = routeResult.ProviderID
		result.Model = routeResult.Model
		result.FallbackUsed = routeResult.FallbackUsed
		result.AttemptCount = routeResult.AttemptCount
		result.StartedProvider = routeResult.StartedProvider
		result.RouteStrategy = string(routeResult.Strategy)
		if routeErr != nil {
			return result, routeErr
		}
		content = routeResult.Content
	} else {
		client := g.clientFor(ctx)
		if client == nil {
			return FlowchartResult{}, common.NewAppError(common.CodeInternalServer, "flowchart generation is not configured", http.StatusServiceUnavailable)
		}
		content, err = client.generate(ctx, prompt)
		if err != nil {
			return FlowchartResult{}, err
		}
		result.Provider = "openai-compatible"
		result.Model = client.model
		result.AttemptCount = 1
	}

	remoteURL := extractFlowchartImageURL(content)
	if remoteURL == "" {
		return FlowchartResult{}, common.NewAppError(common.CodeInternalServer, "flowchart generation did not return an image", http.StatusBadGateway)
	}

	image, err := g.uploader.SaveRemoteImage(ctx, remoteURL)
	if err != nil {
		return FlowchartResult{}, err
	}

	return FlowchartResult{
		ImageURL:        image.URL,
		SourceHash:      hashFlowchartPromptInput(input),
		Provider:        result.Provider,
		Model:           result.Model,
		FallbackUsed:    result.FallbackUsed,
		AttemptCount:    result.AttemptCount,
		StartedProvider: result.StartedProvider,
		RouteStrategy:   result.RouteStrategy,
	}, nil
}

func (g *FlowchartGenerator) clientFor(ctx context.Context) *flowchartClient {
	if g == nil {
		return nil
	}
	cfg := g.defaultConfig
	if g.configLoader != nil {
		runtimeCfg := g.configLoader(ctx)
		if strings.TrimSpace(runtimeCfg.BaseURL) != "" {
			cfg.BaseURL = strings.TrimSpace(runtimeCfg.BaseURL)
		}
		if strings.TrimSpace(runtimeCfg.APIKey) != "" {
			cfg.APIKey = strings.TrimSpace(runtimeCfg.APIKey)
		}
		if strings.TrimSpace(runtimeCfg.Model) != "" {
			cfg.Model = strings.TrimSpace(runtimeCfg.Model)
		}
		if strings.TrimSpace(runtimeCfg.EndpointMode) != "" {
			cfg.EndpointMode = strings.TrimSpace(runtimeCfg.EndpointMode)
		}
		if strings.TrimSpace(runtimeCfg.ResponseFormat) != "" {
			cfg.ResponseFormat = strings.TrimSpace(runtimeCfg.ResponseFormat)
		}
		if runtimeCfg.Timeout > 0 {
			cfg.Timeout = runtimeCfg.Timeout
		}
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return nil
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 45 * time.Second
	}
	return &flowchartClient{
		baseURL:        strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		apiKey:         strings.TrimSpace(cfg.APIKey),
		model:          strings.TrimSpace(cfg.Model),
		endpointMode:   airouter.NormalizeProviderEndpointMode(cfg.EndpointMode),
		responseFormat: airouter.NormalizeProviderResponseFormat(cfg.ResponseFormat),
		httpClient:     &http.Client{Timeout: cfg.Timeout},
		tracker:        g.tracker,
	}
}
