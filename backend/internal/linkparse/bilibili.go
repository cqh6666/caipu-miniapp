package linkparse

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/securehttp"
)

var bilibiliInputDomains = []string{"bilibili.com", "b23.tv", "bili2233.cn"}

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

	result, err := s.fetchBilibiliViaSidecar(ctx, rawInput, bilibiliFetchOptions{IncludeTranscript: true})
	if err != nil {
		finishResult(BilibiliParseResult{}, airouter.ChatCompletionResult{}, err)
		return BilibiliParseResult{}, err
	}

	if result.SubtitleText == "" {
		result.SummaryMode = "heuristic"
		result.RecipeDraft = summarizeHeuristically(result, "")
		result.Warnings = append(result.Warnings, "当前视频没有可直接访问的字幕，已使用标题和简介生成降级草稿。")
		finishResult(result, airouter.ChatCompletionResult{}, nil)
		return result, nil
	}

	var summaryRoute airouter.ChatCompletionResult
	if s.hasSummaryAI(ctx) {
		draft, routeInfo, summaryErr := s.summarizeBilibiliDraft(ctx, result)
		summaryRoute = routeInfo
		if summaryErr == nil {
			result.SummaryMode = "ai"
			result.RecipeDraft = normalizeDraft(result, draft)
			finishResult(result, routeInfo, nil)
			return result, nil
		}
		result.Warnings = append(result.Warnings, buildAISummaryFallbackWarning(summaryErr))
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
	sidecar := s.sidecarFor(ctx)
	if sidecar == nil {
		return sidecarUnavailableError()
	}
	return sidecar.verifyBilibiliSession(ctx, sessdata)
}

func extractInputURL(rawInput string) (string, error) {
	value, err := extractSupportedURL(rawInput)
	if err != nil {
		return "", common.NewAppError(common.CodeBadRequest, "invalid bilibili url", http.StatusBadRequest)
	}

	parsedURL, err := url.Parse(value)
	if err != nil || securehttp.ValidateURL(parsedURL) != nil || !isResolvableBilibiliHost(parsedURL.Hostname()) {
		return "", common.NewAppError(common.CodeBadRequest, "invalid bilibili url", http.StatusBadRequest)
	}
	return parsedURL.String(), nil
}

func isResolvableBilibiliHost(host string) bool {
	return securehttp.HostMatches(host, bilibiliInputDomains...)
}

func (s *Service) currentSessdata(ctx context.Context) string {
	if s == nil || s.bilibiliSessdataProvider == nil {
		return ""
	}
	return strings.TrimSpace(s.bilibiliSessdataProvider(ctx))
}
