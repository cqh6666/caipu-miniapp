package linkparse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

var (
	bvidPattern          = regexp.MustCompile(`(?i)(BV[0-9A-Za-z]{10})`)
	avidPattern          = regexp.MustCompile(`(?i)(?:^|/|[?&])av([0-9]+)`)
	verifySubtitleProbes = []subtitleProbe{
		{BVID: "BV1frwnepEE7", CID: 27914735061},
		{BVID: "BV1gY411C7BY", CID: 1026481904},
		{BVID: "BV1Pw4m1k7pU", CID: 1621665057},
	}
)

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
