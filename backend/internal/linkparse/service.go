package linkparse

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/securehttp"
)

const (
	defaultHTTPTimeout     = 15 * time.Second
	defaultPromptCharLimit = 12000
)

var (
	firstURLPattern  = regexp.MustCompile(`https?://[^\s]+`)
	codeFencePattern = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")
)

type Options struct {
	AIBaseURL                string
	AIAPIKey                 string
	AIModel                  string
	AITimeout                time.Duration
	AITitleEnabled           bool
	AITitleBaseURL           string
	AITitleAPIKey            string
	AITitleModel             string
	AITitleStream            bool
	AITitleTemperature       float64
	AITitleMaxTokens         int
	AITitleTimeout           time.Duration
	BilibiliSessdataProvider func(context.Context) string
	LinkparseSidecarEnabled  bool
	LinkparseSidecarBaseURL  string
	LinkparseSidecarTimeout  time.Duration
	LinkparseSidecarAPIKey   string
	HTTPClient               *http.Client
	AIHTTPClient             *http.Client
	ResolveURLClient         *http.Client
	RuntimeConfigLoader      RuntimeConfigLoader
	AIRouter                 *airouter.Service
	Tracker                  audit.Tracker
}

type Service struct {
	httpClient               *http.Client
	resolveURLClient         *http.Client
	defaultRuntimeConfig     RuntimeConfig
	runtimeConfigLoader      RuntimeConfigLoader
	ai                       *aiClient
	titleAI                  *aiClient
	sidecar                  *sidecarClient
	aiRouter                 *airouter.Service
	bilibiliSessdataProvider func(context.Context) string
	tracker                  audit.Tracker
}

type aiClient struct {
	baseURL     string
	apiKey      string
	model       string
	httpClient  *http.Client
	stream      bool
	temperature float64
	maxTokens   int
	tracker     audit.Tracker
}

type RuntimeConfigLoader func(context.Context) RuntimeConfig

type RuntimeConfig struct {
	SummaryAI        SummaryAIConfig
	TitleAI          TitleAIConfig
	LinkparseSidecar LinkparseSidecarConfig
}

type SummaryAIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

type TitleAIConfig struct {
	Enabled     bool
	BaseURL     string
	APIKey      string
	Model       string
	Stream      bool
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

type LinkparseSidecarConfig struct {
	Enabled bool
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

func NewService(opts Options) *Service {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = securehttp.NewClient(defaultHTTPTimeout)
	}

	resolveURLClient := opts.ResolveURLClient
	if resolveURLClient == nil {
		resolveURLClient = newBilibiliResolveClient(defaultHTTPTimeout)
	}

	titleModel := strings.TrimSpace(opts.AITitleModel)
	if titleModel == "" {
		titleModel = strings.TrimSpace(opts.AIModel)
	}
	titleBaseURL := strings.TrimRight(strings.TrimSpace(opts.AITitleBaseURL), "/")
	if titleBaseURL == "" {
		titleBaseURL = strings.TrimRight(strings.TrimSpace(opts.AIBaseURL), "/")
	}
	titleAPIKey := strings.TrimSpace(opts.AITitleAPIKey)
	if titleAPIKey == "" {
		titleAPIKey = strings.TrimSpace(opts.AIAPIKey)
	}

	defaultRuntimeConfig := RuntimeConfig{
		SummaryAI: SummaryAIConfig{
			BaseURL: strings.TrimRight(strings.TrimSpace(opts.AIBaseURL), "/"),
			APIKey:  strings.TrimSpace(opts.AIAPIKey),
			Model:   strings.TrimSpace(opts.AIModel),
			Timeout: opts.AITimeout,
		},
		TitleAI: TitleAIConfig{
			Enabled:     opts.AITitleEnabled,
			BaseURL:     titleBaseURL,
			APIKey:      titleAPIKey,
			Model:       titleModel,
			Stream:      opts.AITitleStream,
			Temperature: opts.AITitleTemperature,
			MaxTokens:   opts.AITitleMaxTokens,
			Timeout:     opts.AITitleTimeout,
		},
		LinkparseSidecar: LinkparseSidecarConfig{
			Enabled: opts.LinkparseSidecarEnabled,
			BaseURL: strings.TrimRight(strings.TrimSpace(opts.LinkparseSidecarBaseURL), "/"),
			APIKey:  strings.TrimSpace(opts.LinkparseSidecarAPIKey),
			Timeout: opts.LinkparseSidecarTimeout,
		},
	}

	if defaultRuntimeConfig.SummaryAI.BaseURL == "" {
		defaultRuntimeConfig.SummaryAI.BaseURL = "https://api.openai.com/v1"
	}
	if defaultRuntimeConfig.SummaryAI.Timeout <= 0 {
		defaultRuntimeConfig.SummaryAI.Timeout = 30 * time.Second
	}
	if defaultRuntimeConfig.TitleAI.BaseURL == "" {
		defaultRuntimeConfig.TitleAI.BaseURL = defaultRuntimeConfig.SummaryAI.BaseURL
	}
	if defaultRuntimeConfig.TitleAI.Timeout <= 0 {
		defaultRuntimeConfig.TitleAI.Timeout = 3 * time.Second
	}
	if defaultRuntimeConfig.TitleAI.MaxTokens <= 0 {
		defaultRuntimeConfig.TitleAI.MaxTokens = 64
	}
	if defaultRuntimeConfig.LinkparseSidecar.Timeout <= 0 {
		defaultRuntimeConfig.LinkparseSidecar.Timeout = defaultHTTPTimeout
	}

	var summaryAI *aiClient
	var titleAI *aiClient
	var sidecar *sidecarClient
	if opts.RuntimeConfigLoader == nil {
		if strings.TrimSpace(defaultRuntimeConfig.SummaryAI.Model) != "" {
			summaryAI = &aiClient{
				baseURL:    defaultRuntimeConfig.SummaryAI.BaseURL,
				apiKey:     defaultRuntimeConfig.SummaryAI.APIKey,
				model:      defaultRuntimeConfig.SummaryAI.Model,
				httpClient: &http.Client{Timeout: defaultRuntimeConfig.SummaryAI.Timeout},
				tracker:    opts.Tracker,
			}
		}
		if strings.TrimSpace(defaultRuntimeConfig.TitleAI.Model) != "" {
			titleAI = &aiClient{
				baseURL:     defaultRuntimeConfig.TitleAI.BaseURL,
				apiKey:      defaultRuntimeConfig.TitleAI.APIKey,
				model:       defaultRuntimeConfig.TitleAI.Model,
				httpClient:  &http.Client{Timeout: defaultRuntimeConfig.TitleAI.Timeout},
				stream:      defaultRuntimeConfig.TitleAI.Stream,
				temperature: defaultRuntimeConfig.TitleAI.Temperature,
				maxTokens:   defaultRuntimeConfig.TitleAI.MaxTokens,
				tracker:     opts.Tracker,
			}
		}
		if defaultRuntimeConfig.LinkparseSidecar.Enabled && strings.TrimSpace(defaultRuntimeConfig.LinkparseSidecar.BaseURL) != "" {
			sidecar = &sidecarClient{
				baseURL: defaultRuntimeConfig.LinkparseSidecar.BaseURL,
				apiKey:  defaultRuntimeConfig.LinkparseSidecar.APIKey,
				client:  &http.Client{Timeout: defaultRuntimeConfig.LinkparseSidecar.Timeout},
				tracker: opts.Tracker,
			}
		}
	}

	return &Service{
		httpClient:               httpClient,
		resolveURLClient:         resolveURLClient,
		defaultRuntimeConfig:     defaultRuntimeConfig,
		runtimeConfigLoader:      opts.RuntimeConfigLoader,
		ai:                       summaryAI,
		titleAI:                  titleAI,
		sidecar:                  sidecar,
		aiRouter:                 opts.AIRouter,
		bilibiliSessdataProvider: opts.BilibiliSessdataProvider,
		tracker:                  opts.Tracker,
	}
}

func (s *Service) runtimeConfig(ctx context.Context) RuntimeConfig {
	cfg := s.defaultRuntimeConfig
	if s != nil && s.runtimeConfigLoader != nil {
		runtimeCfg := s.runtimeConfigLoader(ctx)
		if strings.TrimSpace(runtimeCfg.SummaryAI.BaseURL) != "" {
			cfg.SummaryAI.BaseURL = strings.TrimSpace(runtimeCfg.SummaryAI.BaseURL)
		}
		if strings.TrimSpace(runtimeCfg.SummaryAI.APIKey) != "" {
			cfg.SummaryAI.APIKey = strings.TrimSpace(runtimeCfg.SummaryAI.APIKey)
		}
		if strings.TrimSpace(runtimeCfg.SummaryAI.Model) != "" {
			cfg.SummaryAI.Model = strings.TrimSpace(runtimeCfg.SummaryAI.Model)
		}
		if runtimeCfg.SummaryAI.Timeout > 0 {
			cfg.SummaryAI.Timeout = runtimeCfg.SummaryAI.Timeout
		}

		cfg.TitleAI.Enabled = runtimeCfg.TitleAI.Enabled
		if strings.TrimSpace(runtimeCfg.TitleAI.BaseURL) != "" {
			cfg.TitleAI.BaseURL = strings.TrimSpace(runtimeCfg.TitleAI.BaseURL)
		}
		if strings.TrimSpace(runtimeCfg.TitleAI.APIKey) != "" {
			cfg.TitleAI.APIKey = strings.TrimSpace(runtimeCfg.TitleAI.APIKey)
		}
		if strings.TrimSpace(runtimeCfg.TitleAI.Model) != "" {
			cfg.TitleAI.Model = strings.TrimSpace(runtimeCfg.TitleAI.Model)
		}
		cfg.TitleAI.Stream = runtimeCfg.TitleAI.Stream
		cfg.TitleAI.Temperature = runtimeCfg.TitleAI.Temperature
		if runtimeCfg.TitleAI.MaxTokens > 0 {
			cfg.TitleAI.MaxTokens = runtimeCfg.TitleAI.MaxTokens
		}
		if runtimeCfg.TitleAI.Timeout > 0 {
			cfg.TitleAI.Timeout = runtimeCfg.TitleAI.Timeout
		}

		cfg.LinkparseSidecar.Enabled = runtimeCfg.LinkparseSidecar.Enabled
		if strings.TrimSpace(runtimeCfg.LinkparseSidecar.BaseURL) != "" {
			cfg.LinkparseSidecar.BaseURL = strings.TrimSpace(runtimeCfg.LinkparseSidecar.BaseURL)
		}
		if strings.TrimSpace(runtimeCfg.LinkparseSidecar.APIKey) != "" {
			cfg.LinkparseSidecar.APIKey = strings.TrimSpace(runtimeCfg.LinkparseSidecar.APIKey)
		}
		if runtimeCfg.LinkparseSidecar.Timeout > 0 {
			cfg.LinkparseSidecar.Timeout = runtimeCfg.LinkparseSidecar.Timeout
		}
	}

	if strings.TrimSpace(cfg.TitleAI.Model) == "" {
		cfg.TitleAI.Model = cfg.SummaryAI.Model
	}
	if strings.TrimSpace(cfg.TitleAI.BaseURL) == "" {
		cfg.TitleAI.BaseURL = cfg.SummaryAI.BaseURL
	}
	if strings.TrimSpace(cfg.TitleAI.APIKey) == "" {
		cfg.TitleAI.APIKey = cfg.SummaryAI.APIKey
	}
	return cfg
}

func (s *Service) summaryAIFor(ctx context.Context) *aiClient {
	if s != nil && s.ai != nil {
		return s.ai
	}
	cfg := s.runtimeConfig(ctx).SummaryAI
	if strings.TrimSpace(cfg.Model) == "" {
		return nil
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &aiClient{
		baseURL:    baseURL,
		apiKey:     strings.TrimSpace(cfg.APIKey),
		model:      strings.TrimSpace(cfg.Model),
		httpClient: &http.Client{Timeout: timeout},
		tracker:    s.tracker,
	}
}

func (s *Service) titleAIFor(ctx context.Context) *aiClient {
	if s != nil && s.titleAI != nil {
		return s.titleAI
	}
	cfg := s.runtimeConfig(ctx).TitleAI
	if !cfg.Enabled || strings.TrimSpace(cfg.Model) == "" {
		return nil
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 64
	}

	return &aiClient{
		baseURL:     baseURL,
		apiKey:      strings.TrimSpace(cfg.APIKey),
		model:       strings.TrimSpace(cfg.Model),
		httpClient:  &http.Client{Timeout: timeout},
		stream:      cfg.Stream,
		temperature: cfg.Temperature,
		maxTokens:   maxTokens,
		tracker:     s.tracker,
	}
}

func (s *Service) hasSummaryAI(ctx context.Context) bool {
	if s != nil && s.aiRouter != nil {
		return s.aiRouter.IsSceneAvailable(ctx, airouter.SceneSummary)
	}
	return s.summaryAIFor(ctx) != nil
}

func (s *Service) hasTitleAI(ctx context.Context) bool {
	if s != nil && s.aiRouter != nil {
		return s.aiRouter.IsSceneAvailable(ctx, airouter.SceneTitle)
	}
	return s.titleAIFor(ctx) != nil
}

func (s *Service) sidecarFor(ctx context.Context) *sidecarClient {
	if s != nil && s.sidecar != nil {
		return s.sidecar
	}
	cfg := s.runtimeConfig(ctx).LinkparseSidecar
	if !cfg.Enabled || strings.TrimSpace(cfg.BaseURL) == "" {
		return nil
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}

	return &sidecarClient{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		apiKey:  strings.TrimSpace(cfg.APIKey),
		client:  &http.Client{Timeout: timeout},
		tracker: s.tracker,
	}
}
