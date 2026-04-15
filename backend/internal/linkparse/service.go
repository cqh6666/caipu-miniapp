package linkparse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const (
	defaultHTTPTimeout     = 15 * time.Second
	defaultPromptCharLimit = 12000
	maxParsedSteps         = 6
	maxRawParsedSteps      = 12
)

var (
	bvidPattern                         = regexp.MustCompile(`(?i)(BV[0-9A-Za-z]{10})`)
	avidPattern                         = regexp.MustCompile(`(?i)(?:^|/|[?&])av([0-9]+)`)
	firstURLPattern                     = regexp.MustCompile(`https?://[^\s]+`)
	stepVerbPattern                     = regexp.MustCompile(`(切|洗|腌|拌|加|放|倒|下锅|翻炒|炒|煎|炸|蒸|煮|炖|焖|焯|烤|淋|撒|搅|收汁|出锅|开吃|冷藏|静置)`)
	stepOrderPattern                    = regexp.MustCompile(`^(先|再|然后|接着|最后|随后|第一步|第二步|第三步|第四步)`)
	ingredientUnitPattern               = regexp.MustCompile(`[\p{Han}A-Za-z][\p{Han}A-Za-z0-9()（）-]{0,14}\s*\d+(?:\.\d+)?\s*(?:g|kg|克|千克|ml|毫升|l|升|勺|汤匙|茶匙|匙|杯|个|颗|根|把|片|块|斤|两|袋|盒|碗)`)
	ingredientLoosePattern              = regexp.MustCompile(`[\p{Han}A-Za-z][\p{Han}A-Za-z0-9()（）-]{0,14}\s*(?:适量|少许)`)
	ingredientSpacingPattern            = regexp.MustCompile(`([\p{Han}A-Za-z])(\d)`)
	secondaryIngredientPattern          = regexp.MustCompile(`(?i)(常用调味料|调味|葱|姜|蒜|香叶|桂皮|八角|花椒|胡椒|盐|糖|冰糖|白糖|红糖|生抽|老抽|蚝油|料酒|鸡精|味精|醋|陈醋|米醋|香醋|豆瓣酱|辣椒|小米椒|淀粉|清水|热水|食用油|香油|芝麻油|花椒粉|辣椒粉|五香粉|十三香|孜然|芝麻|香菜|葱花)`)
	secondaryIngredientExceptionPattern = regexp.MustCompile(`(?i)^(洋葱|红葱头|葱头)`)
	ingredientSuffixPattern             = regexp.MustCompile(`\s*(?:\d+(?:\.\d+)?\s*(?:g|kg|克|千克|ml|毫升|l|升|勺|汤匙|茶匙|匙|杯|个|颗|根|把|片|块|斤|两|袋|盒|碗)|半个|半颗|半根|半头|适量|少许)$`)
	codeFencePattern                    = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")
	summaryWhitespacePattern            = regexp.MustCompile(`\s+`)
	previewBracketPattern               = regexp.MustCompile(`[【\[]([^】\]]+)[】\]]`)
	previewPlatformPattern              = regexp.MustCompile(`\s*-\s*(哔哩哔哩|小红书)\s*$`)
	previewShareSuffix                  = regexp.MustCompile(`复制后打开【小红书】查看笔记!?`)
	previewWhitespacePattern            = regexp.MustCompile(`\s+`)
	previewSplitPattern                 = regexp.MustCompile(`[!！?？~～|｜/·•,:，。；;、\s]+`)
	previewLowConfidence                = regexp.MustCompile(`(?i)(教程|做法|分享|来咯|来啦|来了|最好吃|就是这个味|超级软烂|超软烂|入口即化|香迷糊|巨好吃|真的绝了|一学就会|零失败|保姆级|超下饭|超级入味)`)
	previewNarrativePattern             = regexp.MustCompile(`(?i)(我做了|我家|我们家|拿手菜|私房菜|祖传|开店|饭店|餐馆|摆摊|多年|[0-9一二三四五六七八九十两]+年)`)
	previewDishPattern                  = regexp.MustCompile(`(?i)(炖|炒|烧|煮|蒸|焖|拌|炸|卤|煎|烤|焗|煲|炝|凉拌|清蒸|红烧|糖醋|牛腩|牛肉|排骨|鸡翅|鸡腿|五花肉|里脊|番茄|西红柿|土豆|茄子|豆腐|虾|鱼|面|饭|粥|汤|蛋)`)
	previewDescriptorPattern            = regexp.MustCompile(`(?i)(鲜香|入味|浓稠|软烂|下饭|香辣|酸甜|麻辣|清爽|酥脆|嫩滑|家常|科学)`)
	verifySubtitleProbes                = []subtitleProbe{
		{BVID: "BV1frwnepEE7", CID: 27914735061},
		{BVID: "BV1gY411C7BY", CID: 1026481904},
		{BVID: "BV1Pw4m1k7pU", CID: 1621665057},
	}
	previewTitleNoisePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(.+?)(?:最好吃的做法|家常做法|详细做法|做法分享|做法教程|做法来了|做法来咯|做法来啦|教程来咯|教程来啦|教程来了|教程分享|教程|做法).*$`),
		regexp.MustCompile(`(?i)(.+?)(?:就是这个味|超级软烂|超软烂|入口即化|香迷糊了?|巨好吃|好吃到哭|一学就会|零失败|保姆级|超下饭|真的绝了?|超级入味).*$`),
	}
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

type videoRef struct {
	BVID string
	AID  int64
	Page int
	URL  string
}

type subtitleProbe struct {
	BVID string
	CID  int64
}

type bilibiliViewResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Title string `json:"title"`
		Desc  string `json:"desc"`
		Pic   string `json:"pic"`
		BVID  string `json:"bvid"`
		AID   int64  `json:"aid"`
		Owner struct {
			Name string `json:"name"`
		} `json:"owner"`
		Pages []struct {
			CID  int64  `json:"cid"`
			Page int    `json:"page"`
			Part string `json:"part"`
		} `json:"pages"`
	} `json:"data"`
}

type bilibiliPlayerResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		NeedLoginSubtitle bool `json:"need_login_subtitle"`
		Subtitle          struct {
			Subtitles []struct {
				Lang        string `json:"lan"`
				LangDoc     string `json:"lan_doc"`
				SubtitleURL string `json:"subtitle_url"`
			} `json:"subtitles"`
		} `json:"subtitle"`
	} `json:"data"`
}

type bilibiliSubtitleFile struct {
	Body []struct {
		From    float64 `json:"from"`
		To      float64 `json:"to"`
		Content string  `json:"content"`
	} `json:"body"`
}

type aiSummaryResponse struct {
	Title                string          `json:"title"`
	Ingredient           string          `json:"ingredient"`
	Summary              string          `json:"summary"`
	MainIngredients      []string        `json:"mainIngredients"`
	SecondaryIngredients []string        `json:"secondaryIngredients"`
	Ingredients          []string        `json:"ingredients"`
	Steps                json.RawMessage `json:"steps"`
	Note                 string          `json:"note"`
}

func (r aiSummaryResponse) toParsedContent() (ParsedContent, error) {
	steps, legacySteps, err := parseParsedContentSteps(r.Steps)
	if err != nil {
		return ParsedContent{}, err
	}

	return ParsedContent{
		MainIngredients:      r.MainIngredients,
		SecondaryIngredients: r.SecondaryIngredients,
		Steps:                steps,
		legacyIngredients:    r.Ingredients,
		legacySteps:          legacySteps,
	}, nil
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
	Stream      *bool               `json:"stream,omitempty"`
	MaxTokens   *int                `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewService(opts Options) *Service {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	resolveURLClient := opts.ResolveURLClient
	if resolveURLClient == nil {
		resolveURLClient = httpClient
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

func (s *Service) summarizeBilibiliDraft(ctx context.Context, result BilibiliParseResult) (RecipeDraft, airouter.ChatCompletionResult, error) {
	if s != nil && s.aiRouter != nil {
		routeResult, err := s.aiRouter.RouteChat(ctx, airouter.SceneSummary, airouter.ChatCompletionInput{
			Messages:    buildBilibiliSummaryMessages(result),
			Temperature: floatPtr(0.2),
			ContentKind: "summary_bilibili",
			ValidateContent: func(content string) error {
				_, err := summaryDraftFromAIContent(content)
				return err
			},
		})
		if err != nil {
			return RecipeDraft{}, routeResult, err
		}
		draft, err := summaryDraftFromAIContent(routeResult.Content)
		if err != nil {
			return RecipeDraft{}, airouter.ChatCompletionResult{}, err
		}
		return draft, routeResult, nil
	}

	client := s.summaryAIFor(ctx)
	if client == nil {
		return RecipeDraft{}, airouter.ChatCompletionResult{}, common.NewAppError(common.CodeInternalServer, "summary ai is not configured", http.StatusServiceUnavailable)
	}
	draft, err := client.summarize(ctx, result)
	if err != nil {
		return RecipeDraft{}, airouter.ChatCompletionResult{}, err
	}
	return draft, legacyRouteResult(client.model), nil
}

func (s *Service) summarizeXiaohongshuDraft(ctx context.Context, result XiaohongshuParseResult) (RecipeDraft, airouter.ChatCompletionResult, error) {
	if s != nil && s.aiRouter != nil {
		routeResult, err := s.aiRouter.RouteChat(ctx, airouter.SceneSummary, airouter.ChatCompletionInput{
			Messages:    buildXiaohongshuSummaryMessages(result),
			Temperature: floatPtr(0.2),
			ContentKind: "summary_xiaohongshu",
			ValidateContent: func(content string) error {
				_, err := summaryDraftFromAIContent(content)
				return err
			},
		})
		if err != nil {
			return RecipeDraft{}, routeResult, err
		}
		draft, err := summaryDraftFromAIContent(routeResult.Content)
		if err != nil {
			return RecipeDraft{}, airouter.ChatCompletionResult{}, err
		}
		return draft, routeResult, nil
	}

	client := s.summaryAIFor(ctx)
	if client == nil {
		return RecipeDraft{}, airouter.ChatCompletionResult{}, common.NewAppError(common.CodeInternalServer, "summary ai is not configured", http.StatusServiceUnavailable)
	}
	draft, err := client.summarizeXiaohongshu(ctx, result)
	if err != nil {
		return RecipeDraft{}, airouter.ChatCompletionResult{}, err
	}
	return draft, legacyRouteResult(client.model), nil
}

func (s *Service) refineTitleWithAI(ctx context.Context, rawTitle string) (string, airouter.ChatCompletionResult, error) {
	if s != nil && s.aiRouter != nil {
		routeResult, err := s.aiRouter.RouteChat(ctx, airouter.SceneTitle, airouter.ChatCompletionInput{
			Messages:    buildTitleRefineMessages(rawTitle),
			ContentKind: "title_refine",
			ValidateContent: func(content string) error {
				_, err := parseTitleRefineContent(content)
				return err
			},
		})
		if err != nil {
			return "", routeResult, err
		}
		title, err := parseTitleRefineContent(routeResult.Content)
		if err != nil {
			return "", airouter.ChatCompletionResult{}, err
		}
		return title, routeResult, nil
	}

	client := s.titleAIFor(ctx)
	if client == nil {
		return "", airouter.ChatCompletionResult{}, common.NewAppError(common.CodeInternalServer, "title ai is not configured", http.StatusServiceUnavailable)
	}
	title, err := client.refineTitle(ctx, rawTitle)
	if err != nil {
		return "", airouter.ChatCompletionResult{}, err
	}
	return title, legacyRouteResult(client.model), nil
}

func buildBilibiliSummaryMessages(result BilibiliParseResult) []airouter.ChatMessage {
	return []airouter.ChatMessage{
		{
			Role:    "system",
			Content: "你是一个菜谱整理助手。请根据 B 站视频字幕和简介，提炼适合家庭复刻的菜谱草稿。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\",\"ingredient\":\"\",\"summary\":\"\",\"mainIngredients\":[],\"secondaryIngredients\":[],\"steps\":[{\"title\":\"\",\"detail\":\"\"}],\"note\":\"\"}。steps 必须返回 3 到 6 步；如果原始做法更细，请合并相邻动作，不要拆得过碎，也不要超过 6 步。每一步都要有简短 title 和完整 detail，尽量保留明确的食材名、用量、顺序、火候和动作；不确定的信息不要编造，可以在 note 里提醒用户回看原视频确认。 " + buildIngredientPromptRuleText() + " " + buildSummaryPromptRuleText(),
		},
		{
			Role:    "user",
			Content: buildAISummaryPrompt(result),
		},
	}
}

func buildXiaohongshuSummaryMessages(result XiaohongshuParseResult) []airouter.ChatMessage {
	return []airouter.ChatMessage{
		{
			Role:    "system",
			Content: "你是一个菜谱整理助手。请根据小红书图文笔记正文、标签和图片描述线索，提炼适合家庭复刻的菜谱草稿。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\",\"ingredient\":\"\",\"summary\":\"\",\"mainIngredients\":[],\"secondaryIngredients\":[],\"steps\":[{\"title\":\"\",\"detail\":\"\"}],\"note\":\"\"}。steps 必须返回 3 到 6 步；如果原始做法更细，请合并相邻动作，不要拆得过碎，也不要超过 6 步。每一步都要有简短 title 和完整 detail，尽量保留明确的食材名、用量、顺序、火候和动作；不确定的信息不要编造，可以在 note 里提醒用户回看原笔记和配图确认。 " + buildIngredientPromptRuleText() + " " + buildSummaryPromptRuleText(),
		},
		{
			Role:    "user",
			Content: buildXiaohongshuAISummaryPrompt(result),
		},
	}
}

func buildTitleRefineMessages(rawTitle string) []airouter.ChatMessage {
	return []airouter.ChatMessage{
		{
			Role: "system",
			Content: "你是一个菜谱标题提取助手。请从视频或笔记的原始标题里，提取最适合作为菜谱名的核心菜名。\n\n" +
				"## 判断标准\n" +
				"菜名必须包含「食材」或「烹饪方式」（炒/炖/蒸/煮/烤/卤/拌/煎/焖/红烧/糖醋/凉拌等）中的至少一项。\n" +
				"保留完整的食材搭配关系，不要只留单个食材。例如「番茄土豆炖牛腩」不能缩成「牛腩」。\n\n" +
				"## 拒绝规则\n" +
				"如果标题里没有具体菜名（比如是 vlog、生活日记、合集、探店等），返回 {\"title\":\"\"}。\n" +
				"宁可返回空也不要硬凑一个不是菜名的结果。\n\n" +
				"## 去除内容\n" +
				"去掉平台词（哔哩哔哩/小红书）、教程词（教程/做法/分享）、营销词（巨好吃/零失败/保姆级）、口感修饰（超软烂/入口即化）、系列名、人名。\n\n" +
				"## 示例\n" +
				"- \"番茄土豆炖牛腩教程来咯超级软烂\" → {\"title\":\"番茄土豆炖牛腩\"}\n" +
				"- \"我做了20年的拿手菜西红柿土豆炖牛腩\" → {\"title\":\"西红柿土豆炖牛腩\"}\n" +
				"- \"蒜香排骨最好吃的做法\" → {\"title\":\"蒜香排骨\"}\n" +
				"- \"周末给全家做了一桌好菜\" → {\"title\":\"\"}\n" +
				"- \"跟着婆婆学做菜｜家常红烧肉\" → {\"title\":\"家常红烧肉\"}\n\n" +
				"必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\"}。标题尽量 3 到 12 个汉字，最长不超过 14 个字。",
		},
		{
			Role:    "user",
			Content: "原始标题: " + strings.TrimSpace(rawTitle),
		},
	}
}

func summaryDraftFromAIContent(content string) (RecipeDraft, error) {
	content = strings.TrimSpace(codeFencePattern.ReplaceAllString(strings.TrimSpace(content), "$1"))
	if content == "" {
		return RecipeDraft{}, fmt.Errorf("ai response was empty")
	}

	var summary aiSummaryResponse
	if err := json.Unmarshal([]byte(content), &summary); err != nil {
		return RecipeDraft{}, err
	}
	parsedContent, err := summary.toParsedContent()
	if err != nil {
		return RecipeDraft{}, err
	}
	return RecipeDraft{
		Title:         summary.Title,
		Ingredient:    summary.Ingredient,
		Summary:       summary.Summary,
		Note:          summary.Note,
		ParsedContent: parsedContent,
	}, nil
}

func parseTitleRefineContent(content string) (string, error) {
	content = strings.TrimSpace(codeFencePattern.ReplaceAllString(strings.TrimSpace(content), "$1"))
	if content == "" {
		return "", fmt.Errorf("title ai response was empty")
	}

	var response struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return "", err
	}
	return strings.TrimSpace(response.Title), nil
}

func legacyRouteResult(model string) airouter.ChatCompletionResult {
	return airouter.ChatCompletionResult{
		ProviderID:   airouter.AdapterOpenAICompatible,
		ProviderName: airouter.AdapterOpenAICompatible,
		Model:        strings.TrimSpace(model),
		Strategy:     airouter.StrategyPriorityFailover,
		AttemptCount: 1,
	}
}

func floatPtr(value float64) *float64 {
	return &value
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

func (s *Service) PreviewLink(ctx context.Context, rawInput string) (LinkPreviewResult, error) {
	switch DetectParsePlatform(rawInput) {
	case "bilibili":
		return s.PreviewBilibili(ctx, rawInput)
	case "xiaohongshu":
		return s.PreviewXiaohongshu(ctx, rawInput)
	default:
		return LinkPreviewResult{}, common.NewAppError(common.CodeBadRequest, "unsupported preview link", http.StatusBadRequest)
	}
}

func (s *Service) PreviewBilibili(ctx context.Context, rawInput string) (LinkPreviewResult, error) {
	if sidecar := s.sidecarFor(ctx); sidecar != nil {
		result, err := s.fetchBilibiliViaSidecar(ctx, rawInput, bilibiliFetchOptions{})
		if err != nil {
			return LinkPreviewResult{}, err
		}
		titleOutcome := s.finalizePreviewTitle(ctx, firstNonEmpty(result.Title, result.Part))

		return LinkPreviewResult{
			Platform:     "bilibili",
			Link:         result.Link,
			CanonicalURL: result.Link,
			Title:        titleOutcome.Title,
			TitleSource:  titleOutcome.Source,
			CoverURL:     strings.TrimSpace(result.CoverURL),
			ImageURLs:    draftImageURLs(strings.TrimSpace(result.CoverURL)),
			Warnings:     result.Warnings,
		}, nil
	}

	inputURL, err := extractInputURL(rawInput)
	if err != nil {
		return LinkPreviewResult{}, err
	}

	ref, warnings, err := s.resolveVideoRef(ctx, inputURL)
	if err != nil {
		return LinkPreviewResult{}, err
	}

	view, err := s.fetchView(ctx, ref, s.currentSessdata(ctx))
	if err != nil {
		return LinkPreviewResult{}, err
	}

	page, pageWarnings := pickPage(view.Data.Pages, ref.Page)
	warnings = append(warnings, pageWarnings...)
	titleOutcome := s.finalizePreviewTitle(ctx, firstNonEmpty(view.Data.Title, page.Part))

	return LinkPreviewResult{
		Platform:     "bilibili",
		Link:         ref.URL,
		CanonicalURL: ref.URL,
		Title:        titleOutcome.Title,
		TitleSource:  titleOutcome.Source,
		CoverURL:     strings.TrimSpace(view.Data.Pic),
		ImageURLs:    draftImageURLs(strings.TrimSpace(view.Data.Pic)),
		Warnings:     warnings,
	}, nil
}

func (s *Service) ParseBilibili(ctx context.Context, rawInput string) (BilibiliParseResult, error) {
	trackedCtx, _, finish := s.startTrackedJob(ctx, audit.SceneParseSummary, rawInput, "manual_link", map[string]any{
		"platform": "bilibili",
	})
	ctx = trackedCtx
	finishResult := func(result BilibiliParseResult, routeInfo airouter.ChatCompletionResult, err error) {
		if finish == nil {
			return
		}
		meta := map[string]any{
			"platform":     "bilibili",
			"summary_mode": strings.TrimSpace(result.SummaryMode),
			"warnings":     len(result.Warnings),
		}
		if routeInfo.AttemptCount > 0 {
			meta["route_strategy"] = string(routeInfo.Strategy)
			meta["attempt_count"] = routeInfo.AttemptCount
			meta["started_provider"] = routeInfo.StartedProvider
		}
		jobResult := audit.JobResult{
			Status:        audit.JobStatusSuccess,
			FinalProvider: "heuristic",
			FinalModel:    "",
			FallbackUsed:  strings.TrimSpace(result.SummaryMode) == "heuristic",
			FinishedAt:    audit.NowRFC3339(),
			Meta:          meta,
		}
		if result.SummaryMode == "ai" {
			jobResult.FinalProvider = firstNonEmpty(routeInfo.ProviderID, airouter.AdapterOpenAICompatible)
			jobResult.FinalModel = routeInfo.Model
			jobResult.FallbackUsed = routeInfo.FallbackUsed
		} else if routeInfo.AttemptCount > 0 {
			jobResult.FallbackUsed = true
		}
		if err != nil {
			jobResult.Status = audit.JobStatusFromError(err)
			jobResult.ErrorMessage = err.Error()
			jobResult.FinalProvider = ""
			jobResult.FinalModel = ""
			jobResult.FallbackUsed = false
		}
		_ = finish(ctx, jobResult)
	}
	var summaryRoute airouter.ChatCompletionResult

	if sidecar := s.sidecarFor(ctx); sidecar != nil {
		result, err := s.fetchBilibiliViaSidecar(ctx, rawInput, bilibiliFetchOptions{IncludeTranscript: true})
		if err != nil {
			finishResult(BilibiliParseResult{}, airouter.ChatCompletionResult{}, err)
			return BilibiliParseResult{}, err
		}

		if result.SubtitleText == "" {
			result.SummaryMode = "heuristic"
			result.RecipeDraft = summarizeHeuristically(result, "")
			result.Warnings = append(result.Warnings, "当前视频没有可直接访问的字幕，已使用标题和简介生成降级草稿。")
			finishResult(result, summaryRoute, nil)
			return result, nil
		}

		if s.hasSummaryAI(ctx) {
			draft, routeInfo, err := s.summarizeBilibiliDraft(ctx, result)
			summaryRoute = routeInfo
			if err == nil {
				result.SummaryMode = "ai"
				result.RecipeDraft = normalizeDraft(result, draft)
				finishResult(result, routeInfo, nil)
				return result, nil
			}
			result.Warnings = append(result.Warnings, buildAISummaryFallbackWarning(err))
		}

		result.SummaryMode = "heuristic"
		result.RecipeDraft = summarizeHeuristically(result, result.SubtitleText)
		finishResult(result, summaryRoute, nil)
		return result, nil
	}

	inputURL, err := extractInputURL(rawInput)
	if err != nil {
		finishResult(BilibiliParseResult{}, airouter.ChatCompletionResult{}, err)
		return BilibiliParseResult{}, err
	}

	ref, warnings, err := s.resolveVideoRef(ctx, inputURL)
	if err != nil {
		finishResult(BilibiliParseResult{}, airouter.ChatCompletionResult{}, err)
		return BilibiliParseResult{}, err
	}

	sessdata := s.currentSessdata(ctx)

	view, err := s.fetchView(ctx, ref, sessdata)
	if err != nil {
		finishResult(BilibiliParseResult{}, airouter.ChatCompletionResult{}, err)
		return BilibiliParseResult{}, err
	}

	page, pageWarnings := pickPage(view.Data.Pages, ref.Page)
	warnings = append(warnings, pageWarnings...)

	result := BilibiliParseResult{
		Source:      "bilibili",
		Link:        ref.URL,
		Title:       strings.TrimSpace(view.Data.Title),
		Description: strings.TrimSpace(view.Data.Desc),
		Part:        strings.TrimSpace(page.Part),
		Author:      strings.TrimSpace(view.Data.Owner.Name),
		CoverURL:    strings.TrimSpace(view.Data.Pic),
		BVID:        strings.TrimSpace(view.Data.BVID),
		AID:         view.Data.AID,
		CID:         page.CID,
		Page:        page.Page,
		Warnings:    warnings,
	}

	subtitles, err := s.fetchSubtitles(ctx, result.BVID, result.CID, sessdata)
	if err != nil {
		finishResult(BilibiliParseResult{}, airouter.ChatCompletionResult{}, err)
		return BilibiliParseResult{}, err
	}

	selectedSubtitle := selectSubtitle(subtitles)
	if selectedSubtitle == nil {
		result.SummaryMode = "heuristic"
		result.RecipeDraft = summarizeHeuristically(result, "")
		result.Warnings = append(result.Warnings, "当前视频没有可直接访问的字幕，已使用标题和简介生成降级草稿。")
		finishResult(result, summaryRoute, nil)
		return result, nil
	}

	subtitleFile, err := s.fetchSubtitleFile(ctx, selectedSubtitle.SubtitleURL, sessdata)
	if err != nil {
		finishResult(BilibiliParseResult{}, airouter.ChatCompletionResult{}, err)
		return BilibiliParseResult{}, err
	}

	subtitleText, segments := buildSubtitleText(subtitleFile)
	result.SubtitleAvailable = subtitleText != ""
	result.SubtitleLanguage = firstNonEmpty(selectedSubtitle.LangDoc, selectedSubtitle.Lang)
	result.SubtitleSegments = segments
	result.SubtitleText = subtitleText

	if result.SubtitleText == "" {
		result.SummaryMode = "heuristic"
		result.RecipeDraft = summarizeHeuristically(result, "")
		result.Warnings = append(result.Warnings, "字幕列表存在，但未提取到可用文本，已回退到标题和简介总结。")
		finishResult(result, summaryRoute, nil)
		return result, nil
	}

	if s.hasSummaryAI(ctx) {
		draft, routeInfo, err := s.summarizeBilibiliDraft(ctx, result)
		summaryRoute = routeInfo
		if err == nil {
			result.SummaryMode = "ai"
			result.RecipeDraft = normalizeDraft(result, draft)
			finishResult(result, routeInfo, nil)
			return result, nil
		}
		result.Warnings = append(result.Warnings, buildAISummaryFallbackWarning(err))
	}

	result.SummaryMode = "heuristic"
	result.RecipeDraft = summarizeHeuristically(result, result.SubtitleText)
	finishResult(result, summaryRoute, nil)
	return result, nil
}

func (s *Service) VerifyBilibiliSessdata(ctx context.Context, sessdata string) error {
	sessdata = strings.TrimSpace(sessdata)
	if sessdata == "" {
		return common.NewAppError(common.CodeBadRequest, "SESSDATA is required", http.StatusBadRequest)
	}

	for _, probe := range verifySubtitleProbes {
		subtitles, err := s.fetchSubtitles(ctx, probe.BVID, probe.CID, sessdata)
		if err != nil {
			continue
		}

		selected := selectSubtitle(subtitles)
		if selected != nil && strings.TrimSpace(selected.SubtitleURL) != "" {
			return nil
		}
	}

	return common.NewAppError(common.CodeBadRequest, "当前 SESSDATA 无法获取 B 站字幕，请更新后重试", http.StatusBadRequest)
}

func extractInputURL(rawInput string) (string, error) {
	value, err := extractSupportedURL(rawInput)
	if err != nil {
		return "", common.NewAppError(common.CodeBadRequest, "invalid bilibili url", http.StatusBadRequest)
	}

	u, err := url.Parse(value)
	if err != nil {
		return "", common.NewAppError(common.CodeBadRequest, "invalid bilibili url", http.StatusBadRequest)
	}
	if strings.TrimSpace(u.Host) == "" {
		return "", common.NewAppError(common.CodeBadRequest, "invalid bilibili url", http.StatusBadRequest)
	}

	return u.String(), nil
}

func (s *Service) resolveVideoRef(ctx context.Context, rawURL string) (videoRef, []string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return videoRef{}, nil, common.NewAppError(common.CodeBadRequest, "invalid bilibili url", http.StatusBadRequest)
	}

	if ref, ok := parseVideoRef(u); ok {
		return ref, nil, nil
	}

	if !isResolvableBilibiliHost(u.Host) {
		return videoRef{}, nil, common.NewAppError(common.CodeBadRequest, "only bilibili links are supported in this POC", http.StatusBadRequest)
	}

	resolvedURL, err := s.resolveFinalURL(ctx, rawURL)
	if err != nil {
		return videoRef{}, nil, err
	}

	resolved, err := url.Parse(resolvedURL)
	if err != nil {
		return videoRef{}, nil, common.NewAppError(common.CodeBadRequest, "invalid bilibili redirect url", http.StatusBadRequest)
	}

	if ref, ok := parseVideoRef(resolved); ok {
		return ref, []string{"已自动展开 B 站短链接。"}, nil
	}

	return videoRef{}, nil, common.NewAppError(common.CodeBadRequest, "could not extract BV/AV id from bilibili url", http.StatusBadRequest)
}

func parseVideoRef(u *url.URL) (videoRef, bool) {
	if u == nil {
		return videoRef{}, false
	}

	host := strings.ToLower(strings.TrimSpace(u.Host))
	if !isResolvableBilibiliHost(host) {
		return videoRef{}, false
	}

	normalizedURL := u.String()
	page := 1
	if rawPage := strings.TrimSpace(u.Query().Get("p")); rawPage != "" {
		if value, err := strconv.Atoi(rawPage); err == nil && value > 0 {
			page = value
		}
	}

	full := normalizedURL
	if match := bvidPattern.FindStringSubmatch(full); len(match) == 2 {
		return videoRef{
			BVID: match[1],
			Page: page,
			URL:  normalizedURL,
		}, true
	}

	if match := avidPattern.FindStringSubmatch(full); len(match) == 2 {
		aid, err := strconv.ParseInt(match[1], 10, 64)
		if err == nil && aid > 0 {
			return videoRef{
				AID:  aid,
				Page: page,
				URL:  normalizedURL,
			}, true
		}
	}

	return videoRef{}, false
}

func isResolvableBilibiliHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return strings.Contains(host, "bilibili.com") || strings.Contains(host, "b23.tv") || strings.Contains(host, "bili2233.cn")
}

func (s *Service) resolveFinalURL(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", common.NewAppError(common.CodeBadRequest, "invalid bilibili url", http.StatusBadRequest)
	}

	addBilibiliHeaders(req, "")
	resp, err := s.resolveURLClient.Do(req)
	if err != nil {
		return "", common.NewAppError(common.CodeBadRequest, "failed to resolve bilibili url", http.StatusBadRequest).WithErr(err)
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))

	return resp.Request.URL.String(), nil
}

func (s *Service) fetchView(ctx context.Context, ref videoRef, sessdata string) (bilibiliViewResponse, error) {
	params := url.Values{}
	if ref.BVID != "" {
		params.Set("bvid", ref.BVID)
	}
	if ref.AID > 0 {
		params.Set("aid", strconv.FormatInt(ref.AID, 10))
	}

	endpoint := "https://api.bilibili.com/x/web-interface/view?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return bilibiliViewResponse{}, common.ErrInternal.WithErr(err)
	}

	addBilibiliHeaders(req, sessdata)

	var payload bilibiliViewResponse
	if err := s.doJSON(req, &payload); err != nil {
		return bilibiliViewResponse{}, err
	}
	if payload.Code != 0 {
		return bilibiliViewResponse{}, common.NewAppError(common.CodeBadRequest, firstNonEmpty(payload.Message, "failed to fetch bilibili video info"), http.StatusBadRequest)
	}
	if payload.Data.AID == 0 || strings.TrimSpace(payload.Data.BVID) == "" || len(payload.Data.Pages) == 0 {
		return bilibiliViewResponse{}, common.NewAppError(common.CodeBadRequest, "bilibili video info is incomplete", http.StatusBadRequest)
	}

	return payload, nil
}

func (s *Service) fetchSubtitles(ctx context.Context, bvid string, cid int64, sessdata string) ([]struct {
	Lang        string `json:"lan"`
	LangDoc     string `json:"lan_doc"`
	SubtitleURL string `json:"subtitle_url"`
}, error) {
	endpoint := fmt.Sprintf("https://api.bilibili.com/x/player/v2?bvid=%s&cid=%d", url.QueryEscape(bvid), cid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, common.ErrInternal.WithErr(err)
	}

	addBilibiliHeaders(req, sessdata)

	var payload bilibiliPlayerResponse
	if err := s.doJSON(req, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		return nil, common.NewAppError(common.CodeBadRequest, firstNonEmpty(payload.Message, "failed to fetch bilibili subtitles"), http.StatusBadRequest)
	}

	return payload.Data.Subtitle.Subtitles, nil
}

func pickPage(pages []struct {
	CID  int64  `json:"cid"`
	Page int    `json:"page"`
	Part string `json:"part"`
}, requestedPage int) (struct {
	CID  int64  `json:"cid"`
	Page int    `json:"page"`
	Part string `json:"part"`
}, []string) {
	if requestedPage <= 0 {
		requestedPage = 1
	}

	for _, page := range pages {
		if page.Page == requestedPage {
			return page, nil
		}
	}

	return pages[0], []string{"请求的分 P 不存在，已回退到第一页。"}
}

func selectSubtitle(items []struct {
	Lang        string `json:"lan"`
	LangDoc     string `json:"lan_doc"`
	SubtitleURL string `json:"subtitle_url"`
}) *struct {
	Lang        string `json:"lan"`
	LangDoc     string `json:"lan_doc"`
	SubtitleURL string `json:"subtitle_url"`
} {
	if len(items) == 0 {
		return nil
	}

	preferred := []string{"zh-CN", "zh-Hans", "zh-Hant", "zh", "ai-zh"}
	for _, lang := range preferred {
		for _, item := range items {
			if strings.EqualFold(strings.TrimSpace(item.Lang), lang) && strings.TrimSpace(item.SubtitleURL) != "" {
				chosen := item
				return &chosen
			}
		}
	}

	for _, item := range items {
		if strings.TrimSpace(item.SubtitleURL) != "" {
			chosen := item
			return &chosen
		}
	}

	return nil
}

func (s *Service) fetchSubtitleFile(ctx context.Context, subtitleURL string, sessdata string) (bilibiliSubtitleFile, error) {
	subtitleURL = strings.TrimSpace(subtitleURL)
	switch {
	case strings.HasPrefix(subtitleURL, "//"):
		subtitleURL = "https:" + subtitleURL
	case strings.HasPrefix(subtitleURL, "/"):
		subtitleURL = "https://api.bilibili.com" + subtitleURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, subtitleURL, nil)
	if err != nil {
		return bilibiliSubtitleFile{}, common.ErrInternal.WithErr(err)
	}

	addBilibiliHeaders(req, sessdata)

	var payload bilibiliSubtitleFile
	if err := s.doJSON(req, &payload); err != nil {
		return bilibiliSubtitleFile{}, err
	}
	return payload, nil
}

func (s *Service) doJSON(req *http.Request, dst any) error {
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return common.NewAppError(common.CodeBadRequest, "request to bilibili failed", http.StatusBadRequest).WithErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return common.NewAppError(common.CodeBadRequest, "bilibili request failed", http.StatusBadRequest)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return common.NewAppError(common.CodeBadRequest, "failed to decode bilibili response", http.StatusBadRequest).WithErr(err)
	}

	return nil
}

func addBilibiliHeaders(req *http.Request, sessdata string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	if strings.TrimSpace(sessdata) != "" {
		req.Header.Set("Cookie", "SESSDATA="+strings.TrimSpace(sessdata))
	}
}

func (s *Service) currentSessdata(ctx context.Context) string {
	if s == nil || s.bilibiliSessdataProvider == nil {
		return ""
	}

	return strings.TrimSpace(s.bilibiliSessdataProvider(ctx))
}

func buildSubtitleText(file bilibiliSubtitleFile) (string, int) {
	lines := make([]string, 0, len(file.Body))
	for _, item := range file.Body {
		line := strings.TrimSpace(item.Content)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), len(lines)
}

func summarizeHeuristically(meta BilibiliParseResult, transcript string) RecipeDraft {
	lines := collectCandidateLines(transcript, meta.Description)
	mainIngredients, secondaryIngredients := splitIngredientLines(extractIngredientLines(lines))
	steps := buildParsedSteps(extractStepLines(lines))

	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 {
		mainIngredients, secondaryIngredients = splitIngredientLines(fallbackIngredients(meta.Title))
	}
	if len(steps) == 0 {
		steps = fallbackSteps(meta.Title)
	}

	return RecipeDraft{
		Title:      firstNonEmpty(meta.Title, meta.Part, "B站视频菜谱草稿"),
		Ingredient: buildIngredientSummary(mainIngredients, meta.Title),
		Summary:    buildHeuristicSummary(steps),
		Link:       meta.Link,
		ImageURL:   strings.TrimSpace(meta.CoverURL),
		ImageURLs:  draftImageURLs(strings.TrimSpace(meta.CoverURL)),
		Note:       buildHeuristicNote(meta),
		ParsedContent: ParsedContent{
			MainIngredients:      mainIngredients,
			SecondaryIngredients: secondaryIngredients,
			Steps:                steps,
		},
	}
}

func collectCandidateLines(values ...string) []string {
	var lines []string
	for _, value := range values {
		for _, part := range strings.FieldsFunc(value, func(r rune) bool {
			return r == '\n' || r == '\r' || r == '。' || r == '；' || r == ';'
		}) {
			line := cleanCandidateLine(part)
			if line == "" {
				continue
			}
			lines = append(lines, line)
		}
	}
	return dedupeStrings(lines, 40)
}

func cleanCandidateLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, " ,，。！？!?:：()[]【】\"'")
	value = strings.ReplaceAll(value, "适量即可", "适量")
	value = strings.ReplaceAll(value, "适量就行", "适量")
	return strings.TrimSpace(value)
}

func extractIngredientLines(lines []string) []string {
	items := make([]string, 0, 8)
	for _, line := range lines {
		matched := ingredientUnitPattern.FindAllString(line, -1)
		matched = append(matched, ingredientLoosePattern.FindAllString(line, -1)...)
		if len(matched) == 0 {
			continue
		}

		for _, item := range matched {
			item = normalizeIngredientLine(item)
			if item == "" {
				continue
			}
			items = append(items, item)
		}
	}

	return dedupeStrings(items, 10)
}

func normalizeIngredientLine(value string) string {
	value = cleanCandidateLine(value)
	for _, prefix := range []string{"准备", "食材", "配料", "调料", "还有", "再来", "然后", "这里用到"} {
		value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
	}
	return ingredientSpacingPattern.ReplaceAllString(value, "$1 $2")
}

func extractStepLines(lines []string) []string {
	items := make([]string, 0, 8)
	for _, line := range lines {
		if len([]rune(line)) < 4 {
			continue
		}
		if !stepVerbPattern.MatchString(line) && !stepOrderPattern.MatchString(line) {
			continue
		}
		if strings.HasPrefix(line, "(") || strings.HasPrefix(line, "（") {
			continue
		}
		if strings.Contains(line, "背景音乐") {
			continue
		}

		line = normalizeStepLine(line)
		if line == "" {
			continue
		}
		items = append(items, line)
	}

	return dedupeStrings(items, 8)
}

func normalizeStepLine(value string) string {
	value = cleanCandidateLine(value)
	for _, prefix := range []string{"然后", "接着", "再把", "再来", "最后再"} {
		if strings.HasPrefix(value, prefix) {
			value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
		}
	}
	return value
}

func fallbackIngredients(title string) []string {
	mainIngredient := strings.TrimSpace(title)
	if mainIngredient == "" {
		mainIngredient = "主食材"
	}
	return []string{
		mainIngredient + " 1份",
		"常用调味料 适量",
	}
}

func fallbackSteps(title string) []ParsedStep {
	label := strings.TrimSpace(title)
	if label == "" {
		label = "这道菜"
	}
	return []ParsedStep{
		{Title: "确认食材", Detail: "先结合原视频确认 " + label + " 的主食材和用量。"},
		{Title: "整理步骤", Detail: "根据字幕里提到的顺序整理预处理、下锅和调味步骤。"},
		{Title: "补齐细节", Detail: "做完以后回看原视频，补齐火候和时间等细节。"},
	}
}

func buildIngredientSummary(ingredients []string, fallback string) string {
	names := make([]string, 0, len(ingredients))
	for _, ingredient := range ingredients {
		name := ingredientName(ingredient)
		if name == "" {
			continue
		}
		names = append(names, name)
	}

	names = dedupeStrings(names, 3)
	if len(names) == 0 {
		return strings.TrimSpace(fallback)
	}

	return strings.Join(names, "、")
}

func ingredientName(value string) string {
	value = strings.TrimSpace(value)
	value = ingredientSuffixPattern.ReplaceAllString(value, "")
	value = strings.TrimSpace(value)
	return strings.Trim(value, " ,，。")
}

func splitIngredientLines(lines []string) ([]string, []string) {
	cleaned := dedupeStrings(cleanLines(lines), 12)
	if len(cleaned) == 0 {
		return nil, nil
	}

	mainIngredients := make([]string, 0, 4)
	secondaryIngredients := make([]string, 0, len(cleaned))
	for _, line := range cleaned {
		label := ingredientName(line)
		if secondaryIngredientPattern.MatchString(label) && !secondaryIngredientExceptionPattern.MatchString(label) {
			secondaryIngredients = append(secondaryIngredients, line)
			continue
		}
		mainIngredients = append(mainIngredients, line)
	}

	if len(mainIngredients) == 0 {
		limit := 3
		if len(cleaned) < limit {
			limit = len(cleaned)
		}
		mainIngredients = append(mainIngredients, cleaned[:limit]...)
		secondaryIngredients = append([]string{}, cleaned[limit:]...)
	}

	return mainIngredients, secondaryIngredients
}

func buildParsedSteps(lines []string) []ParsedStep {
	items := make([]ParsedStep, 0, len(lines))
	for index, line := range dedupeStrings(cleanLines(lines), maxRawParsedSteps) {
		items = append(items, ParsedStep{
			Title:  inferParsedStepTitle(line, index),
			Detail: line,
		})
	}
	return compactParsedSteps(items)
}

func cleanParsedSteps(steps []ParsedStep) []ParsedStep {
	items := make([]ParsedStep, 0, len(steps))
	seen := make(map[string]struct{}, len(steps))
	for index, step := range steps {
		title := strings.TrimSpace(step.Title)
		detail := cleanCandidateLine(step.Detail)
		if detail == "" {
			detail = title
		}
		if detail == "" {
			continue
		}
		if title == "" {
			title = inferParsedStepTitle(detail, index)
		}
		key := title + "\x00" + detail
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, ParsedStep{
			Title:  title,
			Detail: detail,
		})
	}
	return compactParsedSteps(items)
}

func compactParsedSteps(steps []ParsedStep) []ParsedStep {
	if len(steps) <= maxParsedSteps {
		return append([]ParsedStep{}, steps...)
	}

	items := make([]ParsedStep, 0, maxParsedSteps)
	for index := 0; index < maxParsedSteps; index++ {
		start := index * len(steps) / maxParsedSteps
		end := (index + 1) * len(steps) / maxParsedSteps
		if start >= len(steps) {
			break
		}
		if end <= start {
			end = start + 1
		}
		if end > len(steps) {
			end = len(steps)
		}

		group := steps[start:end]
		title := strings.TrimSpace(group[0].Title)
		if title == "" {
			title = inferParsedStepTitle(group[0].Detail, index)
		}

		details := make([]string, 0, len(group))
		for _, step := range group {
			detail := cleanCandidateLine(step.Detail)
			if detail == "" {
				continue
			}
			details = append(details, detail)
		}
		if len(details) == 0 && title != "" {
			details = append(details, title)
		}

		items = append(items, ParsedStep{
			Title:  title,
			Detail: strings.Join(details, "；"),
		})
	}

	return items
}

func inferParsedStepTitle(detail string, index int) string {
	switch {
	case strings.Contains(detail, "焯水") || strings.Contains(detail, "汆水"):
		if strings.Contains(detail, "腥") || strings.Contains(detail, "浮沫") {
			return "焯水去腥"
		}
		return "焯水备用"
	case strings.Contains(detail, "腌"):
		return "腌制入味"
	case strings.Contains(detail, "糖色") || strings.Contains(detail, "冰糖"):
		return "炒糖上色"
	case strings.Contains(detail, "爆香") || strings.Contains(detail, "炒香"):
		return "炒香底料"
	case strings.Contains(detail, "切") || strings.Contains(detail, "改刀"):
		return "切配备料"
	case strings.Contains(detail, "收汁"):
		return "收汁出锅"
	case strings.Contains(detail, "炖") || strings.Contains(detail, "焖"):
		return "小火慢炖"
	case strings.Contains(detail, "蒸"):
		return "上锅蒸熟"
	case strings.Contains(detail, "炸"):
		return "炸至金黄"
	case strings.Contains(detail, "煎"):
		return "煎香上色"
	case strings.Contains(detail, "烤"):
		return "烤至上色"
	case strings.Contains(detail, "煮"):
		return "煮至入味"
	case strings.Contains(detail, "拌"):
		return "拌匀调味"
	case strings.Contains(detail, "炒") || strings.Contains(detail, "翻炒"):
		return "翻炒入味"
	case strings.Contains(detail, "出锅"):
		return "调味出锅"
	case index == 0:
		return "处理食材"
	default:
		return "继续烹饪"
	}
}

func buildHeuristicSummary(steps []ParsedStep) string {
	normalized := cleanParsedSteps(steps)
	if len(normalized) == 0 {
		return ""
	}

	first := normalized[0].Title
	second := ""
	for _, step := range normalized[1:] {
		if strings.TrimSpace(step.Title) == "" || step.Title == first {
			continue
		}
		second = step.Title
		break
	}

	switch {
	case first != "" && second != "":
		return normalizeRecipeSummary("先" + first + "，再" + second)
	case first != "":
		return normalizeRecipeSummary(first)
	default:
		return ""
	}
}

func buildHeuristicNote(meta BilibiliParseResult) string {
	base := "基于 B 站"
	if meta.SubtitleAvailable {
		base += "字幕"
	} else {
		base += "标题和简介"
	}
	base += "生成的 POC 草稿，建议做菜前再回看视频核对食材克数、火候和时长。"

	if meta.Part != "" && meta.Part != meta.Title {
		base += " 当前使用分 P：" + meta.Part + "。"
	}
	return base
}

func buildSummaryPromptRuleText() string {
	return "summary 字段用于详情页和美食库列表，必须写成“关键动作 + 结果”的一句中文短句，限制在 24 个汉字以内；不要重复标题里的菜名，不要写平台、图片数量、营销词或不确定信息。示例：番茄牛腩 -> 先焯水去腥，再慢炖至软烂；鸡蛋炸酱面 -> 先炒酱提香，再快速拌面出锅；港式干炒牛河 -> 猛火快炒，牛河更香更入味。"
}

func buildIngredientPromptRuleText() string {
	return "ingredient 只写 2 到 4 个最核心主料，用顿号连接；mainIngredients 只放主菜体或主食材及数量，不要把盐、生抽、料酒这类调味放进去；secondaryIngredients 统一承载配菜、香料和调味，顺序上先写配菜，再写调味，不要把土豆、洋葱、胡萝卜、青椒、香菇和盐、生抽、蚝油、料酒交错混排。"
}

func normalizeRecipeSummary(value string) string {
	summary := strings.TrimSpace(value)
	summary = strings.Trim(summary, "。；;、!！?？\"'")
	summary = summaryWhitespacePattern.ReplaceAllString(summary, "")
	if summary == "" {
		return ""
	}
	return truncateRunes(summary, 24)
}

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}

	items := []rune(value)
	if len(items) <= maxRunes {
		return value
	}
	return string(items[:maxRunes])
}

func (c *aiClient) summarize(ctx context.Context, result BilibiliParseResult) (RecipeDraft, error) {
	startedAt := time.Now()
	payload := openAIChatRequest{
		Model:       c.model,
		Temperature: 0.2,
		Messages: []openAIChatMessage{
			{
				Role:    "system",
				Content: "你是一个菜谱整理助手。请根据 B 站视频字幕和简介，提炼适合家庭复刻的菜谱草稿。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\",\"ingredient\":\"\",\"summary\":\"\",\"mainIngredients\":[],\"secondaryIngredients\":[],\"steps\":[{\"title\":\"\",\"detail\":\"\"}],\"note\":\"\"}。steps 必须返回 3 到 6 步；如果原始做法更细，请合并相邻动作，不要拆得过碎，也不要超过 6 步。每一步都要有简短 title 和完整 detail，尽量保留明确的食材名、用量、顺序、火候和动作；不确定的信息不要编造，可以在 note 里提醒用户回看原视频确认。 " + buildIngredientPromptRuleText() + " " + buildSummaryPromptRuleText(),
			},
			{
				Role:    "user",
				Content: buildAISummaryPrompt(result),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return RecipeDraft{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return RecipeDraft{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFromError(err), 0, err, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if strings.TrimSpace(string(data)) != "" {
			callErr := fmt.Errorf("ai request failed: %s", strings.TrimSpace(string(data)))
			c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
				"content_kind": "summary_bilibili",
			})
			return RecipeDraft{}, callErr
		}
		callErr := fmt.Errorf("ai request failed with status %d", resp.StatusCode)
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, callErr
	}

	var parsed openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, err, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		callErr := fmt.Errorf("ai error: %s", parsed.Error.Message)
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, callErr
	}
	if len(parsed.Choices) == 0 {
		callErr := fmt.Errorf("ai response contained no choices")
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, callErr
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimSpace(codeFencePattern.ReplaceAllString(content, "$1"))
	if content == "" {
		callErr := fmt.Errorf("ai response was empty")
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, callErr
	}

	var summary aiSummaryResponse
	if err := json.Unmarshal([]byte(content), &summary); err != nil {
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, err, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, err
	}

	parsedContent, err := summary.toParsedContent()
	if err != nil {
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, err, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, err
	}

	c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusSuccess, resp.StatusCode, nil, map[string]any{
		"content_kind": "summary_bilibili",
	})

	return RecipeDraft{
		Title:         summary.Title,
		Ingredient:    summary.Ingredient,
		Summary:       summary.Summary,
		Note:          summary.Note,
		ParsedContent: parsedContent,
	}, nil
}

func buildAISummaryPrompt(result BilibiliParseResult) string {
	transcript := result.SubtitleText
	truncated := false
	if len([]rune(transcript)) > defaultPromptCharLimit {
		transcript = string([]rune(transcript)[:defaultPromptCharLimit])
		truncated = true
	}

	var builder strings.Builder
	builder.WriteString("请整理这条 B 站视频里的菜谱信息。\n")
	builder.WriteString("标题: " + firstNonEmpty(result.Title, "未知标题") + "\n")
	if result.Part != "" {
		builder.WriteString("分P: " + result.Part + "\n")
	}
	if result.Author != "" {
		builder.WriteString("作者: " + result.Author + "\n")
	}
	if result.Description != "" {
		builder.WriteString("简介: " + result.Description + "\n")
	}
	builder.WriteString("摘要规则: " + buildSummaryPromptRuleText() + "\n")
	builder.WriteString("食材分组规则: " + buildIngredientPromptRuleText() + "\n")
	builder.WriteString("链接: " + result.Link + "\n")
	builder.WriteString("字幕语言: " + firstNonEmpty(result.SubtitleLanguage, "未知") + "\n")
	if truncated {
		builder.WriteString("注意: 字幕已截断为前 12000 个字符。\n")
	}
	builder.WriteString("字幕内容:\n")
	builder.WriteString(transcript)
	return builder.String()
}

func normalizeDraft(meta BilibiliParseResult, draft RecipeDraft) RecipeDraft {
	draft.Title = firstNonEmpty(strings.TrimSpace(draft.Title), meta.Title, "B站视频菜谱草稿")
	draft.Ingredient = firstNonEmpty(strings.TrimSpace(draft.Ingredient), strings.TrimSpace(meta.Title))
	draft.Link = meta.Link
	draft.ImageURL = firstNonEmpty(strings.TrimSpace(draft.ImageURL), strings.TrimSpace(meta.CoverURL))
	if len(draft.ImageURLs) == 0 {
		draft.ImageURLs = draftImageURLs(draft.ImageURL)
	}
	draft.Note = firstNonEmpty(strings.TrimSpace(draft.Note), "基于 B 站字幕生成的 AI 草稿，建议回看原视频补齐克数和火候。")
	draft.ParsedContent = normalizeParsedContentDraft(draft.ParsedContent)
	draft.Summary = normalizeRecipeSummary(draft.Summary)

	if (len(draft.ParsedContent.MainIngredients) == 0 && len(draft.ParsedContent.SecondaryIngredients) == 0) || len(draft.ParsedContent.Steps) == 0 {
		fallback := summarizeHeuristically(meta, meta.SubtitleText)
		if len(draft.ParsedContent.MainIngredients) == 0 && len(draft.ParsedContent.SecondaryIngredients) == 0 {
			draft.ParsedContent.MainIngredients = fallback.ParsedContent.MainIngredients
			draft.ParsedContent.SecondaryIngredients = fallback.ParsedContent.SecondaryIngredients
		}
		if len(draft.ParsedContent.Steps) == 0 {
			draft.ParsedContent.Steps = fallback.ParsedContent.Steps
		}
		if strings.TrimSpace(draft.Ingredient) == "" {
			draft.Ingredient = fallback.Ingredient
		}
		if strings.TrimSpace(draft.Summary) == "" {
			draft.Summary = fallback.Summary
		}
	}

	if strings.TrimSpace(draft.Ingredient) == "" {
		draft.Ingredient = buildIngredientSummary(draft.ParsedContent.MainIngredients, meta.Title)
	}
	if strings.TrimSpace(draft.Summary) == "" {
		draft.Summary = buildHeuristicSummary(draft.ParsedContent.Steps)
	}

	return draft
}

func normalizeParsedContentDraft(content ParsedContent) ParsedContent {
	mainIngredients := dedupeStrings(cleanLines(content.MainIngredients), 10)
	secondaryIngredients := dedupeStrings(cleanLines(content.SecondaryIngredients), 10)
	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 {
		mainIngredients, secondaryIngredients = splitIngredientLines(cleanLines(content.legacyIngredients))
	}

	steps := cleanParsedSteps(content.Steps)
	if len(steps) == 0 {
		steps = buildParsedSteps(cleanLines(content.legacySteps))
	}

	return ParsedContent{
		MainIngredients:      mainIngredients,
		SecondaryIngredients: secondaryIngredients,
		Steps:                steps,
	}
}

func draftImageURLs(values ...string) []string {
	items := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}
	return items
}

func cleanLines(lines []string) []string {
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		line = cleanCandidateLine(line)
		if line == "" {
			continue
		}
		items = append(items, line)
	}
	return items
}

func dedupeStrings(values []string, limit int) []string {
	seen := make(map[string]struct{}, len(values))
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		key := value
		if slices.ContainsFunc(items, func(existing string) bool {
			return strings.EqualFold(existing, value)
		}) {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		items = append(items, value)
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type previewTitleOutcome struct {
	Title  string
	Source string
}

func sanitizePreviewTitle(raw string) string {
	title := strings.TrimSpace(raw)
	if title == "" {
		return ""
	}

	if match := previewBracketPattern.FindStringSubmatch(title); len(match) == 2 {
		title = strings.TrimSpace(match[1])
	}

	title = strings.TrimSpace(previewPlatformPattern.ReplaceAllString(title, ""))
	title = strings.TrimSpace(previewShareSuffix.ReplaceAllString(title, ""))
	title = trimTrailingPreviewTag(title)
	title = strings.TrimSpace(previewWhitespacePattern.ReplaceAllString(title, " "))
	title = trimPreviewTitleNoise(title)
	title = choosePreviewTitleCandidate(title)
	title = strings.TrimSpace(strings.Trim(title, "[]【】"))
	title = strings.TrimSpace(strings.TrimRight(title, "。！!~～ "))
	return title
}

func (s *Service) finalizePreviewTitle(ctx context.Context, raw string) previewTitleOutcome {
	title := sanitizePreviewTitle(raw)
	outcome := previewTitleOutcome{
		Title:  title,
		Source: "",
	}
	if title == "" {
		return outcome
	}
	outcome.Source = "rule"
	if s == nil || !s.hasTitleAI(ctx) {
		return outcome
	}

	trackedCtx, _, finish := s.startTrackedJob(ctx, audit.SceneTitleRefine, raw, "preview_link", map[string]any{
		"raw_title": title,
	})
	refined, routeInfo, err := s.refineTitleWithAI(trackedCtx, raw)
	if err != nil {
		if finish != nil {
			_ = finish(trackedCtx, audit.JobResult{
				Status:        audit.JobStatusFallback,
				FinalProvider: "rule",
				FinalModel:    "",
				FallbackUsed:  true,
				ErrorMessage:  err.Error(),
				FinishedAt:    audit.NowRFC3339(),
				Meta: map[string]any{
					"result_source":    "rule",
					"route_strategy":   string(routeInfo.Strategy),
					"attempt_count":    routeInfo.AttemptCount,
					"started_provider": routeInfo.StartedProvider,
				},
			})
		}
		return outcome
	}

	refined = sanitizePreviewTitle(refined)
	if refined == "" {
		if finish != nil {
			_ = finish(trackedCtx, audit.JobResult{
				Status:        audit.JobStatusFallback,
				FinalProvider: "rule",
				FinalModel:    "",
				FallbackUsed:  true,
				ErrorMessage:  "title ai returned empty result",
				FinishedAt:    audit.NowRFC3339(),
				Meta: map[string]any{
					"result_source":    "rule",
					"route_strategy":   string(routeInfo.Strategy),
					"attempt_count":    routeInfo.AttemptCount,
					"started_provider": routeInfo.StartedProvider,
				},
			})
		}
		return outcome
	}

	aiScore := scorePreviewTitleCandidate(refined)
	ruleScore := scorePreviewTitleCandidate(title)
	if aiScore < ruleScore {
		if finish != nil {
			_ = finish(trackedCtx, audit.JobResult{
				Status:        audit.JobStatusFallback,
				FinalProvider: "rule",
				FinalModel:    "",
				FallbackUsed:  true,
				ErrorMessage:  fmt.Sprintf("ai title %q scored %d < rule title %q scored %d", refined, aiScore, title, ruleScore),
				FinishedAt:    audit.NowRFC3339(),
				Meta: map[string]any{
					"result_source":    "rule",
					"ai_title":         refined,
					"ai_score":         aiScore,
					"rule_score":       ruleScore,
					"route_strategy":   string(routeInfo.Strategy),
					"attempt_count":    routeInfo.AttemptCount,
					"started_provider": routeInfo.StartedProvider,
				},
			})
		}
		return outcome
	}

	outcome.Title = refined
	outcome.Source = "ai"
	if finish != nil {
		_ = finish(trackedCtx, audit.JobResult{
			Status:        audit.JobStatusSuccess,
			FinalProvider: firstNonEmpty(routeInfo.ProviderID, airouter.AdapterOpenAICompatible),
			FinalModel:    routeInfo.Model,
			FallbackUsed:  routeInfo.FallbackUsed,
			FinishedAt:    audit.NowRFC3339(),
			Meta: map[string]any{
				"result_source":    "ai",
				"route_strategy":   string(routeInfo.Strategy),
				"attempt_count":    routeInfo.AttemptCount,
				"started_provider": routeInfo.StartedProvider,
			},
		})
	}
	return outcome
}

func trimPreviewTitleNoise(title string) string {
	value := strings.TrimSpace(title)
	if value == "" {
		return ""
	}

	for _, pattern := range previewTitleNoisePatterns {
		if match := pattern.FindStringSubmatch(value); len(match) == 2 {
			candidate := strings.TrimSpace(match[1])
			if len([]rune(candidate)) >= 2 {
				value = candidate
				break
			}
		}
	}

	value = strings.TrimSpace(strings.TrimRight(value, "。！!~～ "))
	return value
}

func choosePreviewTitleCandidate(title string) string {
	value := strings.TrimSpace(title)
	if value == "" {
		return ""
	}

	candidates := collectPreviewTitleCandidates(value)
	best := value
	bestScore := scorePreviewTitleCandidate(value)
	bestLen := len([]rune(value))

	for _, candidate := range candidates {
		score := scorePreviewTitleCandidate(candidate)
		length := len([]rune(candidate))
		if score > bestScore || (score == bestScore && length < bestLen) {
			best = candidate
			bestScore = score
			bestLen = length
		}
	}

	return best
}

func isLowConfidencePreviewTitle(title string) bool {
	return scorePreviewTitleCandidate(title) < 5
}

func scorePreviewTitleCandidate(title string) int {
	value := strings.TrimSpace(title)
	if value == "" {
		return -100
	}
	runeCount := len([]rune(value))
	score := 0
	switch {
	case runeCount < 2:
		score -= 8
	case runeCount <= 12:
		score += 4
	case runeCount <= 16:
		score += 2
	case runeCount <= 20:
		score -= 1
	default:
		score -= 5
	}

	if previewDishPattern.MatchString(value) {
		score += 5
	}
	if previewLowConfidence.MatchString(value) {
		score -= 3
	}
	if previewNarrativePattern.MatchString(value) {
		score -= 4
	}
	if strings.Contains(value, "的") && previewDescriptorPattern.MatchString(value) {
		score -= 1
	}

	return score
}

func collectPreviewTitleCandidates(title string) []string {
	candidates := make([]string, 0, 8)
	appendCandidate := func(raw string) {
		candidate := strings.TrimSpace(raw)
		candidate = trimTrailingPreviewTag(candidate)
		candidate = strings.TrimSpace(strings.Trim(candidate, "[]【】"))
		candidate = strings.TrimSpace(strings.TrimRight(candidate, "。！!~～ "))
		candidate = trimPreviewTitleNoise(candidate)
		if len([]rune(candidate)) < 2 {
			return
		}
		if slices.Contains(candidates, candidate) {
			return
		}
		candidates = append(candidates, candidate)
	}

	appendCandidate(title)

	for _, segment := range previewSplitPattern.Split(title, -1) {
		appendCandidate(segment)
	}

	for _, candidate := range append([]string{}, candidates...) {
		if idx := strings.LastIndex(candidate, "的"); idx >= 0 && idx < len(candidate)-len("的") {
			appendCandidate(candidate[idx+len("的"):])
		}
	}

	return candidates
}

func trimTrailingPreviewTag(title string) string {
	value := strings.TrimSpace(title)
	lastBracket := strings.LastIndexAny(value, "【[")
	if lastBracket > 0 {
		value = strings.TrimSpace(value[:lastBracket])
	}
	return value
}

func (c *aiClient) refineTitle(ctx context.Context, rawTitle string) (string, error) {
	startedAt := time.Now()
	stream := c != nil && c.stream
	maxTokens := 64
	if c != nil && c.maxTokens > 0 {
		maxTokens = c.maxTokens
	}
	temperature := 0.0
	if c != nil {
		temperature = c.temperature
	}

	msgs := buildTitleRefineMessages(rawTitle)
	openAIMsgs := make([]openAIChatMessage, len(msgs))
	for i, m := range msgs {
		openAIMsgs[i] = openAIChatMessage{Role: m.Role, Content: m.Content}
	}

	payload := openAIChatRequest{
		Model:       c.model,
		Temperature: temperature,
		Stream:      &stream,
		MaxTokens:   &maxTokens,
		Messages:    openAIMsgs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFromError(err), 0, err, map[string]any{
			"content_kind": "title_refine",
		})
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if strings.TrimSpace(string(data)) != "" {
			callErr := fmt.Errorf("title ai request failed: %s", strings.TrimSpace(string(data)))
			c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
				"content_kind": "title_refine",
			})
			return "", callErr
		}
		callErr := fmt.Errorf("title ai request failed with status %d", resp.StatusCode)
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "title_refine",
		})
		return "", callErr
	}

	var parsed openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, err, map[string]any{
			"content_kind": "title_refine",
		})
		return "", err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		callErr := fmt.Errorf("title ai error: %s", parsed.Error.Message)
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "title_refine",
		})
		return "", callErr
	}
	if len(parsed.Choices) == 0 {
		callErr := fmt.Errorf("title ai response contained no choices")
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "title_refine",
		})
		return "", callErr
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimSpace(codeFencePattern.ReplaceAllString(content, "$1"))
	if content == "" {
		callErr := fmt.Errorf("title ai response was empty")
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "title_refine",
		})
		return "", callErr
	}

	var response struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, err, map[string]any{
			"content_kind": "title_refine",
		})
		return "", err
	}

	c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusSuccess, resp.StatusCode, nil, map[string]any{
		"content_kind": "title_refine",
	})
	return strings.TrimSpace(response.Title), nil
}
