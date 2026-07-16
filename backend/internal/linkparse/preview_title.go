package linkparse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

var (
	previewBracketPattern     = regexp.MustCompile(`[【\[]([^】\]]+)[】\]]`)
	previewPlatformPattern    = regexp.MustCompile(`\s*-\s*(哔哩哔哩|小红书)\s*$`)
	previewShareSuffix        = regexp.MustCompile(`复制后打开【小红书】查看笔记!?`)
	previewWhitespacePattern  = regexp.MustCompile(`\s+`)
	previewSplitPattern       = regexp.MustCompile(`[!！?？~～|｜/·•,:，。；;、\s]+`)
	previewLowConfidence      = regexp.MustCompile(`(?i)(教程|做法|分享|来咯|来啦|来了|最好吃|就是这个味|超级软烂|超软烂|入口即化|香迷糊|巨好吃|真的绝了|一学就会|零失败|保姆级|超下饭|超级入味)`)
	previewNarrativePattern   = regexp.MustCompile(`(?i)(我做了|我家|我们家|拿手菜|私房菜|祖传|开店|饭店|餐馆|摆摊|多年|[0-9一二三四五六七八九十两]+年)`)
	previewDishPattern        = regexp.MustCompile(`(?i)(炖|炒|烧|煮|蒸|焖|拌|炸|卤|煎|烤|焗|煲|炝|凉拌|清蒸|红烧|糖醋|牛腩|牛肉|排骨|鸡翅|鸡腿|五花肉|里脊|番茄|西红柿|土豆|茄子|豆腐|虾|鱼|面|饭|粥|汤|蛋)`)
	previewDescriptorPattern  = regexp.MustCompile(`(?i)(鲜香|入味|浓稠|软烂|下饭|香辣|酸甜|麻辣|清爽|酥脆|嫩滑|家常|科学)`)
	previewTitleNoisePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(.+?)(?:最好吃的做法|家常做法|详细做法|做法分享|做法教程|做法来了|做法来咯|做法来啦|教程来咯|教程来啦|教程来了|教程分享|教程|做法).*$`),
		regexp.MustCompile(`(?i)(.+?)(?:就是这个味|超级软烂|超软烂|入口即化|香迷糊了?|巨好吃|好吃到哭|一学就会|零失败|保姆级|超下饭|真的绝了?|超级入味).*$`),
	}
)

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
		callErr := sanitizedUpstreamError(
			common.CodeInternalServer,
			fmt.Sprintf("title AI upstream returned status %d", resp.StatusCode),
			http.StatusBadGateway,
			string(data),
		)
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "title_refine",
		})
		return "", callErr
	}

	var parsed openAIChatResponse
	if err := decodeBoundedUpstreamJSON(resp.Body, maxLinkparseAIResponseBytes, "title AI upstream", &parsed); err != nil {
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, err, map[string]any{
			"content_kind": "title_refine",
		})
		return "", err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		callErr := sanitizedUpstreamError(common.CodeInternalServer, "title AI upstream returned an error", http.StatusBadGateway, parsed.Error.Message)
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
