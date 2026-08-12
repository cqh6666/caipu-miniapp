package linkparse

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
)

const (
	defaultHTTPTimeout     = 15 * time.Second
	defaultPromptCharLimit = 12000
)

var (
	firstURLPattern  = regexp.MustCompile(`https?://[^\s]+`)
	codeFencePattern = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")
)

type AIRouter interface {
	IsSceneAvailable(context.Context, airouter.Scene) bool
	RouteChat(context.Context, airouter.Scene, airouter.ChatCompletionInput) (airouter.ChatCompletionResult, error)
}

type Options struct {
	BilibiliSessdataProvider func(context.Context) string
	LinkparseSidecarEnabled  bool
	LinkparseSidecarBaseURL  string
	LinkparseSidecarTimeout  time.Duration
	LinkparseSidecarAPIKey   string
	RuntimeConfigLoader      RuntimeConfigLoader
	AIRouter                 AIRouter
	Tracker                  audit.Tracker
}

type Service struct {
	defaultRuntimeConfig     RuntimeConfig
	runtimeConfigLoader      RuntimeConfigLoader
	sidecar                  *sidecarClient
	aiRouter                 AIRouter
	bilibiliSessdataProvider func(context.Context) string
	tracker                  audit.Tracker
}

type RuntimeConfigLoader func(context.Context) RuntimeConfig

type RuntimeConfig struct {
	LinkparseSidecar LinkparseSidecarConfig
}

type LinkparseSidecarConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

func NewService(opts Options) *Service {
	defaultRuntimeConfig := RuntimeConfig{
		LinkparseSidecar: LinkparseSidecarConfig{
			Enabled: opts.LinkparseSidecarEnabled,
			BaseURL: strings.TrimRight(strings.TrimSpace(opts.LinkparseSidecarBaseURL), "/"),
			APIKey:  strings.TrimSpace(opts.LinkparseSidecarAPIKey),
			Timeout: opts.LinkparseSidecarTimeout,
		},
	}
	if defaultRuntimeConfig.LinkparseSidecar.Timeout <= 0 {
		defaultRuntimeConfig.LinkparseSidecar.Timeout = defaultHTTPTimeout
	}

	var sidecar *sidecarClient
	if opts.RuntimeConfigLoader == nil && defaultRuntimeConfig.LinkparseSidecar.Enabled && defaultRuntimeConfig.LinkparseSidecar.BaseURL != "" {
		sidecar = newSidecarClient(defaultRuntimeConfig.LinkparseSidecar, opts.Tracker)
	}

	return &Service{
		defaultRuntimeConfig:     defaultRuntimeConfig,
		runtimeConfigLoader:      opts.RuntimeConfigLoader,
		sidecar:                  sidecar,
		aiRouter:                 opts.AIRouter,
		bilibiliSessdataProvider: opts.BilibiliSessdataProvider,
		tracker:                  opts.Tracker,
	}
}

func (s *Service) runtimeConfig(ctx context.Context) RuntimeConfig {
	cfg := s.defaultRuntimeConfig
	if s != nil && s.runtimeConfigLoader != nil {
		runtimeCfg := s.runtimeConfigLoader(ctx).LinkparseSidecar
		cfg.LinkparseSidecar.Enabled = runtimeCfg.Enabled
		if strings.TrimSpace(runtimeCfg.BaseURL) != "" {
			cfg.LinkparseSidecar.BaseURL = strings.TrimSpace(runtimeCfg.BaseURL)
		}
		if strings.TrimSpace(runtimeCfg.APIKey) != "" {
			cfg.LinkparseSidecar.APIKey = strings.TrimSpace(runtimeCfg.APIKey)
		}
		if runtimeCfg.Timeout > 0 {
			cfg.LinkparseSidecar.Timeout = runtimeCfg.Timeout
		}
	}
	return cfg
}

func (s *Service) hasSummaryAI(ctx context.Context) bool {
	return s != nil && s.aiRouter != nil && s.aiRouter.IsSceneAvailable(ctx, airouter.SceneSummary)
}

func (s *Service) hasTitleAI(ctx context.Context) bool {
	return s != nil && s.aiRouter != nil && s.aiRouter.IsSceneAvailable(ctx, airouter.SceneTitle)
}

func (s *Service) sidecarFor(ctx context.Context) *sidecarClient {
	if s != nil && s.sidecar != nil {
		return s.sidecar
	}
	if s == nil {
		return nil
	}
	cfg := s.runtimeConfig(ctx).LinkparseSidecar
	if !cfg.Enabled || strings.TrimSpace(cfg.BaseURL) == "" {
		return nil
	}
	return newSidecarClient(cfg, s.tracker)
}

func newSidecarClient(cfg LinkparseSidecarConfig, tracker audit.Tracker) *sidecarClient {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	return &sidecarClient{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		apiKey:  strings.TrimSpace(cfg.APIKey),
		client:  &http.Client{Timeout: timeout},
		tracker: tracker,
	}
}
