package linkparse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

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
		callErr := sanitizedUpstreamError(
			common.CodeInternalServer,
			fmt.Sprintf("summary AI upstream returned status %d", resp.StatusCode),
			http.StatusBadGateway,
			string(data),
		)
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, callErr
	}

	var parsed openAIChatResponse
	if err := decodeBoundedUpstreamJSON(resp.Body, maxLinkparseAIResponseBytes, "summary AI upstream", &parsed); err != nil {
		c.logCall(ctx, startedAt, "/chat/completions", audit.CallStatusFailed, resp.StatusCode, err, map[string]any{
			"content_kind": "summary_bilibili",
		})
		return RecipeDraft{}, err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		callErr := sanitizedUpstreamError(common.CodeInternalServer, "summary AI upstream returned an error", http.StatusBadGateway, parsed.Error.Message)
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
