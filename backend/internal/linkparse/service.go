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

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const (
	defaultHTTPTimeout     = 15 * time.Second
	defaultPromptCharLimit = 12000
)

var (
	bvidPattern              = regexp.MustCompile(`(?i)(BV[0-9A-Za-z]{10})`)
	avidPattern              = regexp.MustCompile(`(?i)(?:^|/|[?&])av([0-9]+)`)
	firstURLPattern          = regexp.MustCompile(`https?://[^\s]+`)
	stepVerbPattern          = regexp.MustCompile(`(切|洗|腌|拌|加|放|倒|下锅|翻炒|炒|煎|炸|蒸|煮|炖|焖|焯|烤|淋|撒|搅|收汁|出锅|开吃|冷藏|静置)`)
	stepOrderPattern         = regexp.MustCompile(`^(先|再|然后|接着|最后|随后|第一步|第二步|第三步|第四步)`)
	ingredientUnitPattern    = regexp.MustCompile(`[\p{Han}A-Za-z][\p{Han}A-Za-z0-9()（）-]{0,14}\s*\d+(?:\.\d+)?\s*(?:g|kg|克|千克|ml|毫升|l|升|勺|汤匙|茶匙|匙|杯|个|颗|根|把|片|块|斤|两|袋|盒|碗)`)
	ingredientLoosePattern   = regexp.MustCompile(`[\p{Han}A-Za-z][\p{Han}A-Za-z0-9()（）-]{0,14}\s*(?:适量|少许)`)
	ingredientSpacingPattern = regexp.MustCompile(`([\p{Han}A-Za-z])(\d)`)
	codeFencePattern         = regexp.MustCompile("(?s)^```(?:json)?\\s*(.*?)\\s*```$")
	previewBracketPattern    = regexp.MustCompile(`[【\[]([^】\]]+)[】\]]`)
	previewPlatformPattern   = regexp.MustCompile(`\s*-\s*(哔哩哔哩|小红书)\s*$`)
	previewShareSuffix       = regexp.MustCompile(`复制后打开【小红书】查看笔记!?`)
	previewWhitespacePattern = regexp.MustCompile(`\s+`)
	previewSplitPattern      = regexp.MustCompile(`[!！?？~～|｜/·•,:，。；;、\s]+`)
	previewLowConfidence     = regexp.MustCompile(`(?i)(教程|做法|分享|来咯|来啦|来了|最好吃|就是这个味|超级软烂|超软烂|入口即化|香迷糊|巨好吃|真的绝了|一学就会|零失败|保姆级|超下饭|超级入味)`)
	previewNarrativePattern  = regexp.MustCompile(`(?i)(我做了|我家|我们家|拿手菜|私房菜|祖传|开店|饭店|餐馆|摆摊|多年|[0-9一二三四五六七八九十两]+年)`)
	previewDishPattern       = regexp.MustCompile(`(?i)(炖|炒|烧|煮|蒸|焖|拌|炸|卤|煎|烤|焗|煲|炝|凉拌|清蒸|红烧|糖醋|牛腩|牛肉|排骨|鸡翅|鸡腿|五花肉|里脊|番茄|西红柿|土豆|茄子|豆腐|虾|鱼|面|饭|粥|汤|蛋)`)
	previewDescriptorPattern = regexp.MustCompile(`(?i)(鲜香|入味|浓稠|软烂|下饭|香辣|酸甜|麻辣|清爽|酥脆|嫩滑|家常|科学)`)
	verifySubtitleProbes     = []subtitleProbe{
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
	AITitleModel             string
	AITitleTimeout           time.Duration
	BilibiliSessdataProvider func(context.Context) string
	XHSSidecarEnabled        bool
	XHSSidecarBaseURL        string
	XHSSidecarTimeout        time.Duration
	XHSSidecarProvider       string
	XHSSidecarAPIKey         string
	HTTPClient               *http.Client
	AIHTTPClient             *http.Client
	ResolveURLClient         *http.Client
}

type Service struct {
	httpClient               *http.Client
	resolveURLClient         *http.Client
	ai                       *aiClient
	titleAI                  *aiClient
	xhs                      *xiaohongshuClient
	bilibiliSessdataProvider func(context.Context) string
}

type aiClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
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
	Title       string   `json:"title"`
	Ingredient  string   `json:"ingredient"`
	Ingredients []string `json:"ingredients"`
	Steps       []string `json:"steps"`
	Note        string   `json:"note"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
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

	var xhs *xiaohongshuClient
	if opts.XHSSidecarEnabled && strings.TrimSpace(opts.XHSSidecarBaseURL) != "" {
		xhsHTTPClient := &http.Client{Timeout: defaultHTTPTimeout}
		if opts.XHSSidecarTimeout > 0 {
			xhsHTTPClient.Timeout = opts.XHSSidecarTimeout
		}

		xhs = &xiaohongshuClient{
			baseURL:  strings.TrimRight(strings.TrimSpace(opts.XHSSidecarBaseURL), "/"),
			provider: firstNonEmpty(strings.TrimSpace(opts.XHSSidecarProvider), "auto"),
			apiKey:   strings.TrimSpace(opts.XHSSidecarAPIKey),
			client:   xhsHTTPClient,
		}
	}

	var summaryAI *aiClient
	if strings.TrimSpace(opts.AIModel) != "" {
		aiHTTPClient := opts.AIHTTPClient
		if aiHTTPClient == nil {
			timeout := opts.AITimeout
			if timeout <= 0 {
				timeout = 30 * time.Second
			}
			aiHTTPClient = &http.Client{Timeout: timeout}
		}

		baseURL := strings.TrimRight(strings.TrimSpace(opts.AIBaseURL), "/")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}

		summaryAI = &aiClient{
			baseURL:    baseURL,
			apiKey:     strings.TrimSpace(opts.AIAPIKey),
			model:      strings.TrimSpace(opts.AIModel),
			httpClient: aiHTTPClient,
		}
	}

	var titleAI *aiClient
	if opts.AITitleEnabled && strings.TrimSpace(opts.AITitleModel) != "" {
		titleHTTPClient := &http.Client{Timeout: 3 * time.Second}
		if opts.AITitleTimeout > 0 {
			titleHTTPClient.Timeout = opts.AITitleTimeout
		}

		baseURL := strings.TrimRight(strings.TrimSpace(opts.AIBaseURL), "/")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}

		titleAI = &aiClient{
			baseURL:    baseURL,
			apiKey:     strings.TrimSpace(opts.AIAPIKey),
			model:      strings.TrimSpace(opts.AITitleModel),
			httpClient: titleHTTPClient,
		}
	}

	return &Service{
		httpClient:               httpClient,
		resolveURLClient:         resolveURLClient,
		ai:                       summaryAI,
		titleAI:                  titleAI,
		xhs:                      xhs,
		bilibiliSessdataProvider: opts.BilibiliSessdataProvider,
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

	return LinkPreviewResult{
		Platform:     "bilibili",
		Link:         ref.URL,
		CanonicalURL: ref.URL,
		Title:        s.finalizePreviewTitle(ctx, firstNonEmpty(view.Data.Title, page.Part)),
		CoverURL:     strings.TrimSpace(view.Data.Pic),
		ImageURLs:    draftImageURLs(strings.TrimSpace(view.Data.Pic)),
		Warnings:     warnings,
	}, nil
}

func (s *Service) ParseBilibili(ctx context.Context, rawInput string) (BilibiliParseResult, error) {
	inputURL, err := extractInputURL(rawInput)
	if err != nil {
		return BilibiliParseResult{}, err
	}

	ref, warnings, err := s.resolveVideoRef(ctx, inputURL)
	if err != nil {
		return BilibiliParseResult{}, err
	}

	sessdata := s.currentSessdata(ctx)

	view, err := s.fetchView(ctx, ref, sessdata)
	if err != nil {
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
		return BilibiliParseResult{}, err
	}

	selectedSubtitle := selectSubtitle(subtitles)
	if selectedSubtitle == nil {
		result.SummaryMode = "heuristic"
		result.RecipeDraft = summarizeHeuristically(result, "")
		result.Warnings = append(result.Warnings, "当前视频没有可直接访问的字幕，已使用标题和简介生成降级草稿。")
		return result, nil
	}

	subtitleFile, err := s.fetchSubtitleFile(ctx, selectedSubtitle.SubtitleURL, sessdata)
	if err != nil {
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
		return result, nil
	}

	if s.ai != nil {
		draft, err := s.ai.summarize(ctx, result)
		if err == nil {
			result.SummaryMode = "ai"
			result.RecipeDraft = normalizeDraft(result, draft)
			return result, nil
		}
		result.Warnings = append(result.Warnings, "AI 总结暂时不可用，已回退到规则总结。")
	}

	result.SummaryMode = "heuristic"
	result.RecipeDraft = summarizeHeuristically(result, result.SubtitleText)
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
	ingredients := extractIngredientLines(lines)
	steps := extractStepLines(lines)

	if len(ingredients) == 0 {
		ingredients = fallbackIngredients(meta.Title)
	}
	if len(steps) == 0 {
		steps = fallbackSteps(meta.Title)
	}

	return RecipeDraft{
		Title:      firstNonEmpty(meta.Title, meta.Part, "B站视频菜谱草稿"),
		Ingredient: buildIngredientSummary(ingredients, meta.Title),
		Link:       meta.Link,
		ImageURL:   strings.TrimSpace(meta.CoverURL),
		ImageURLs:  draftImageURLs(strings.TrimSpace(meta.CoverURL)),
		Note:       buildHeuristicNote(meta),
		ParsedContent: ParsedContent{
			Ingredients: ingredients,
			Steps:       steps,
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

func fallbackSteps(title string) []string {
	label := strings.TrimSpace(title)
	if label == "" {
		label = "这道菜"
	}
	return []string{
		"先结合原视频确认 " + label + " 的主食材和用量。",
		"根据字幕里提到的顺序整理预处理、下锅和调味步骤。",
		"做完以后回看原视频，补齐火候和时间等细节。",
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
	value = regexp.MustCompile(`\s*\d+(?:\.\d+)?\s*(?:g|kg|克|千克|ml|毫升|l|升|勺|汤匙|茶匙|匙|杯|个|颗|根|把|片|块|斤|两|袋|盒|碗)$`).ReplaceAllString(value, "")
	value = regexp.MustCompile(`\s*(?:适量|少许)$`).ReplaceAllString(value, "")
	value = strings.TrimSpace(value)
	return strings.Trim(value, " ,，。")
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

func (c *aiClient) summarize(ctx context.Context, result BilibiliParseResult) (RecipeDraft, error) {
	payload := openAIChatRequest{
		Model:       c.model,
		Temperature: 0.2,
		Messages: []openAIChatMessage{
			{
				Role:    "system",
				Content: "你是一个菜谱整理助手。请根据 B 站视频字幕和简介，提炼适合家庭复刻的菜谱草稿。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\",\"ingredient\":\"\",\"ingredients\":[],\"steps\":[],\"note\":\"\"}。ingredients 和 steps 各返回 2 到 8 条，尽量保留明确的食材名、用量、顺序、火候和动作；不确定的信息不要编造，可以在 note 里提醒用户回看原视频确认。",
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
		return RecipeDraft{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if strings.TrimSpace(string(data)) != "" {
			return RecipeDraft{}, fmt.Errorf("ai request failed: %s", strings.TrimSpace(string(data)))
		}
		return RecipeDraft{}, fmt.Errorf("ai request failed with status %d", resp.StatusCode)
	}

	var parsed openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return RecipeDraft{}, err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return RecipeDraft{}, fmt.Errorf("ai error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return RecipeDraft{}, fmt.Errorf("ai response contained no choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimSpace(codeFencePattern.ReplaceAllString(content, "$1"))
	if content == "" {
		return RecipeDraft{}, fmt.Errorf("ai response was empty")
	}

	var summary aiSummaryResponse
	if err := json.Unmarshal([]byte(content), &summary); err != nil {
		return RecipeDraft{}, err
	}

	return RecipeDraft{
		Title:      summary.Title,
		Ingredient: summary.Ingredient,
		Note:       summary.Note,
		ParsedContent: ParsedContent{
			Ingredients: summary.Ingredients,
			Steps:       summary.Steps,
		},
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
	draft.ParsedContent.Ingredients = dedupeStrings(cleanLines(draft.ParsedContent.Ingredients), 10)
	draft.ParsedContent.Steps = dedupeStrings(cleanLines(draft.ParsedContent.Steps), 8)

	if len(draft.ParsedContent.Ingredients) == 0 || len(draft.ParsedContent.Steps) == 0 {
		fallback := summarizeHeuristically(meta, meta.SubtitleText)
		if len(draft.ParsedContent.Ingredients) == 0 {
			draft.ParsedContent.Ingredients = fallback.ParsedContent.Ingredients
		}
		if len(draft.ParsedContent.Steps) == 0 {
			draft.ParsedContent.Steps = fallback.ParsedContent.Steps
		}
		if strings.TrimSpace(draft.Ingredient) == "" {
			draft.Ingredient = fallback.Ingredient
		}
	}

	return draft
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

func (s *Service) finalizePreviewTitle(ctx context.Context, raw string) string {
	title := sanitizePreviewTitle(raw)
	if title == "" || s == nil || s.titleAI == nil || !isLowConfidencePreviewTitle(title) {
		return title
	}

	refined, err := s.titleAI.refineTitle(ctx, raw, title)
	if err != nil {
		return title
	}

	refined = sanitizePreviewTitle(refined)
	if refined == "" {
		return title
	}
	if scorePreviewTitleCandidate(refined) >= scorePreviewTitleCandidate(title) {
		return refined
	}
	return title
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

func (c *aiClient) refineTitle(ctx context.Context, rawTitle, currentTitle string) (string, error) {
	payload := openAIChatRequest{
		Model:       c.model,
		Temperature: 0,
		Messages: []openAIChatMessage{
			{
				Role: "system",
				Content: "你是一个菜谱标题清洗助手。请从原始分享标题里提取最适合作为菜谱名的核心菜名。" +
					"必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\"}。" +
					"不要返回平台词、教程词、营销词、口感修饰、系列名。标题尽量 3 到 12 个汉字，最长不超过 14 个字。",
			},
			{
				Role: "user",
				Content: "原始标题: " + strings.TrimSpace(rawTitle) + "\n" +
					"当前规则结果: " + strings.TrimSpace(currentTitle) + "\n" +
					"请只提取核心菜名。",
			},
		},
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
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		if strings.TrimSpace(string(data)) != "" {
			return "", fmt.Errorf("title ai request failed: %s", strings.TrimSpace(string(data)))
		}
		return "", fmt.Errorf("title ai request failed with status %d", resp.StatusCode)
	}

	var parsed openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("title ai error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("title ai response contained no choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimSpace(codeFencePattern.ReplaceAllString(content, "$1"))
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
