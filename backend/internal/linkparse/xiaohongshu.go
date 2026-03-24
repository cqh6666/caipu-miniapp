package linkparse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type xhsFetchOptions struct {
	IncludeTranscript bool
}

func (s *Service) ParseRecipeLink(ctx context.Context, rawInput string) (RecipeParseOutcome, error) {
	switch DetectParsePlatform(rawInput) {
	case "bilibili":
		result, err := s.ParseBilibili(ctx, rawInput)
		if err != nil {
			return RecipeParseOutcome{}, err
		}
		return RecipeParseOutcome{
			Source:      result.Source,
			SummaryMode: result.SummaryMode,
			RecipeDraft: result.RecipeDraft,
		}, nil
	case "xiaohongshu":
		result, err := s.parseXiaohongshu(ctx, rawInput, xhsFetchOptions{IncludeTranscript: true})
		if err != nil {
			return RecipeParseOutcome{}, err
		}
		return RecipeParseOutcome{
			Source:       result.Source,
			SourceDetail: strings.TrimSpace(result.NoteType),
			SummaryMode:  result.SummaryMode,
			RecipeDraft:  result.RecipeDraft,
		}, nil
	default:
		return RecipeParseOutcome{}, common.NewAppError(common.CodeBadRequest, "unsupported auto-parse link", http.StatusBadRequest)
	}
}

func (s *Service) PreviewXiaohongshu(ctx context.Context, rawInput string) (LinkPreviewResult, error) {
	result, err := s.fetchXiaohongshu(ctx, rawInput, xhsFetchOptions{})
	if err != nil {
		return LinkPreviewResult{}, err
	}

	return LinkPreviewResult{
		Platform:     "xiaohongshu",
		Link:         result.Link,
		CanonicalURL: firstNonEmpty(result.CanonicalURL, result.Link),
		Title:        s.finalizePreviewTitle(ctx, result.Title),
		CoverURL:     firstNonEmpty(result.CoverURL, firstImage(result.Images)),
		ImageURLs:    preferredXiaohongshuImages(result),
		ProviderUsed: result.ProviderUsed,
		Warnings:     result.Warnings,
	}, nil
}

func (s *Service) ParseXiaohongshu(ctx context.Context, rawInput string) (XiaohongshuParseResult, error) {
	return s.parseXiaohongshu(ctx, rawInput, xhsFetchOptions{})
}

func (s *Service) parseXiaohongshu(ctx context.Context, rawInput string, opts xhsFetchOptions) (XiaohongshuParseResult, error) {
	result, err := s.fetchXiaohongshu(ctx, rawInput, opts)
	if err != nil {
		if !opts.IncludeTranscript || !shouldRetryXiaohongshuWithoutTranscript(err) {
			return XiaohongshuParseResult{}, err
		}

		fallback, fallbackErr := s.fetchXiaohongshu(ctx, rawInput, xhsFetchOptions{})
		if fallbackErr != nil {
			return XiaohongshuParseResult{}, err
		}

		fallback.TranscriptStatus = "failed"
		fallback.TranscriptError = "小红书视频转写超时，已回退为仅解析图文内容。"
		fallback.Warnings = append(fallback.Warnings, fallback.TranscriptError)
		result = fallback
	}

	if s.ai != nil {
		draft, err := s.ai.summarizeXiaohongshu(ctx, result)
		if err == nil {
			result.SummaryMode = "ai"
			result.RecipeDraft = normalizeXiaohongshuDraft(result, draft)
			return result, nil
		}
		result.Warnings = append(result.Warnings, "AI 总结暂时不可用，已回退到规则整理并生成一句话重点。")
	}

	result.SummaryMode = "heuristic"
	result.RecipeDraft = summarizeXiaohongshuHeuristically(result)
	return result, nil
}

func (s *Service) fetchXiaohongshu(ctx context.Context, rawInput string, opts xhsFetchOptions) (XiaohongshuParseResult, error) {
	if s == nil || s.sidecar == nil {
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeInternalServer, "linkparse sidecar is not configured", http.StatusInternalServerError)
	}

	inputURL, err := extractSupportedURL(rawInput)
	if err != nil {
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, "invalid xiaohongshu url", http.StatusBadRequest)
	}
	if !SupportsXiaohongshuURL(inputURL) {
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, "invalid xiaohongshu url", http.StatusBadRequest)
	}

	parsed, err := s.sidecar.parse(ctx, "/v1/parse/xiaohongshu", sidecarParseRequest{
		Input:             rawInput,
		IncludeDebug:      false,
		IncludeTranscript: opts.IncludeTranscript,
	}, nil)
	if err != nil {
		if isLinkparseSidecarTimeout(err) {
			return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, "xiaohongshu sidecar timed out", http.StatusBadRequest).WithErr(err)
		}
		var appErr *common.AppError
		if errors.As(err, &appErr) {
			return XiaohongshuParseResult{}, err
		}
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, "request to xiaohongshu sidecar failed", http.StatusBadRequest).WithErr(err)
	}

	result := XiaohongshuParseResult{
		Source:            "xiaohongshu",
		Link:              firstNonEmpty(parsed.Normalized.ShareURL, inputURL),
		CanonicalURL:      firstNonEmpty(parsed.Normalized.CanonicalURL, inputURL),
		ProviderRequested: firstNonEmpty(parsed.ProviderRequested, "auto"),
		ProviderUsed:      strings.TrimSpace(parsed.ProviderUsed),
		Title:             strings.TrimSpace(parsed.Content.Title),
		Content:           strings.TrimSpace(parsed.Content.Body),
		Transcript:        strings.TrimSpace(parsed.Content.Transcript),
		TranscriptStatus:  strings.TrimSpace(parsed.Content.TranscriptStatus),
		TranscriptError:   strings.TrimSpace(parsed.Content.TranscriptError),
		CoverURL:          normalizeXiaohongshuMediaURL(parsed.Content.CoverURL),
		Images:            normalizeXiaohongshuMediaURLs(parsed.Content.Images, 12),
		Videos:            normalizeXiaohongshuMediaURLs(parsed.Content.Videos, 4),
		Tags:              dedupeStrings(cleanLines(parsed.Content.Tags), 12),
		Author:            strings.TrimSpace(parsed.Content.Author.Name),
		NoteType:          strings.TrimSpace(parsed.Content.ContentType),
		NoteID:            strings.TrimSpace(parsed.Normalized.ID),
		XSECToken:         strings.TrimSpace(parsed.Normalized.XSECToken),
		Warnings:          parsed.Warnings,
	}
	return result, nil
}

func shouldRetryXiaohongshuWithoutTranscript(err error) bool {
	return isLinkparseSidecarTimeout(err)
}

func isLinkparseSidecarTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err)
}

func summarizeXiaohongshuHeuristically(meta XiaohongshuParseResult) RecipeDraft {
	lines := collectCandidateLines(meta.Content, strings.Join(meta.Tags, "\n"), meta.Transcript)
	mainIngredients, secondaryIngredients := splitIngredientLines(extractIngredientLines(lines))
	steps := buildParsedSteps(extractStepLines(lines))

	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 {
		mainIngredients, secondaryIngredients = splitIngredientLines(fallbackIngredients(meta.Title))
	}
	if len(steps) == 0 {
		steps = []ParsedStep{
			{Title: "确认食材", Detail: "先结合小红书原文确认这道菜的主食材和用量。"},
			{Title: "整理步骤", Detail: "按原文提到的顺序整理预处理、调味和烹饪步骤。"},
			{Title: "补齐细节", Detail: "做之前建议回看原链接，补齐克数、火候和时间等细节。"},
		}
	}

	return RecipeDraft{
		Title:      firstNonEmpty(meta.Title, "小红书菜谱草稿"),
		Ingredient: buildIngredientSummary(mainIngredients, meta.Title),
		Summary:    buildHeuristicSummary(steps),
		Link:       firstNonEmpty(meta.CanonicalURL, meta.Link),
		ImageURL:   firstNonEmpty(strings.TrimSpace(meta.CoverURL), firstImage(meta.Images)),
		ImageURLs:  preferredXiaohongshuImages(meta),
		Note:       buildXiaohongshuHeuristicNote(meta),
		ParsedContent: ParsedContent{
			MainIngredients:      mainIngredients,
			SecondaryIngredients: secondaryIngredients,
			Steps:                steps,
		},
	}
}

func buildXiaohongshuHeuristicNote(meta XiaohongshuParseResult) string {
	base := "基于小红书笔记内容生成的草稿，建议做菜前回看原笔记核对食材克数、火候和图片里的细节。"
	if strings.TrimSpace(meta.Transcript) != "" {
		base = "基于小红书正文和视频转写生成的草稿，建议做菜前回看原笔记核对食材克数、火候和图片里的细节。"
	} else if meta.TranscriptStatus == "failed" {
		base += " 当前视频口播未成功转写，步骤细节建议回看原视频确认。"
	}
	if strings.TrimSpace(meta.ProviderUsed) != "" {
		base += " 当前解析策略：" + strings.TrimSpace(meta.ProviderUsed) + "。"
	}
	return base
}

func (c *aiClient) summarizeXiaohongshu(ctx context.Context, result XiaohongshuParseResult) (RecipeDraft, error) {
	payload := openAIChatRequest{
		Model:       c.model,
		Temperature: 0.2,
		Messages: []openAIChatMessage{
			{
				Role:    "system",
				Content: "你是一个菜谱整理助手。请根据小红书图文笔记正文、标签和图片描述线索，提炼适合家庭复刻的菜谱草稿。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\",\"ingredient\":\"\",\"summary\":\"\",\"mainIngredients\":[],\"secondaryIngredients\":[],\"steps\":[{\"title\":\"\",\"detail\":\"\"}],\"note\":\"\"}。steps 必须返回 3 到 6 步；如果原始做法更细，请合并相邻动作，不要拆得过碎，也不要超过 6 步。每一步都要有简短 title 和完整 detail，尽量保留明确的食材名、用量、顺序、火候和动作；不确定的信息不要编造，可以在 note 里提醒用户回看原笔记和配图确认。 " + buildIngredientPromptRuleText() + " " + buildSummaryPromptRuleText(),
			},
			{
				Role:    "user",
				Content: buildXiaohongshuAISummaryPrompt(result),
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

func buildXiaohongshuAISummaryPrompt(result XiaohongshuParseResult) string {
	content := result.Content
	if len([]rune(content)) > defaultPromptCharLimit {
		content = string([]rune(content)[:defaultPromptCharLimit])
	}
	transcript := result.Transcript
	if len([]rune(transcript)) > defaultPromptCharLimit {
		transcript = string([]rune(transcript)[:defaultPromptCharLimit])
	}

	var builder strings.Builder
	builder.WriteString("请整理这条小红书笔记里的菜谱信息；如果存在视频转写，请优先参考口播里的食材、调味和步骤细节。\n")
	builder.WriteString("标题: " + firstNonEmpty(result.Title, "未知标题") + "\n")
	if result.Author != "" {
		builder.WriteString("作者: " + result.Author + "\n")
	}
	builder.WriteString("摘要规则: " + buildSummaryPromptRuleText() + "\n")
	builder.WriteString("食材分组规则: " + buildIngredientPromptRuleText() + "\n")
	builder.WriteString("链接: " + firstNonEmpty(result.CanonicalURL, result.Link) + "\n")
	if len(result.Tags) > 0 {
		builder.WriteString("标签: " + strings.Join(result.Tags, "、") + "\n")
	}
	if len(result.Images) > 0 {
		builder.WriteString(fmt.Sprintf("图片数量: %d\n", len(result.Images)))
	}
	builder.WriteString("正文内容:\n")
	builder.WriteString(content)
	if strings.TrimSpace(transcript) != "" {
		builder.WriteString("\n视频转写:\n")
		builder.WriteString(transcript)
	}
	return builder.String()
}

func normalizeXiaohongshuDraft(meta XiaohongshuParseResult, draft RecipeDraft) RecipeDraft {
	draft.Title = firstNonEmpty(strings.TrimSpace(draft.Title), meta.Title, "小红书菜谱草稿")
	draft.Ingredient = firstNonEmpty(strings.TrimSpace(draft.Ingredient), strings.TrimSpace(meta.Title))
	draft.Link = firstNonEmpty(meta.CanonicalURL, meta.Link)
	draft.ImageURL = firstNonEmpty(strings.TrimSpace(draft.ImageURL), strings.TrimSpace(meta.CoverURL), firstImage(meta.Images))
	if len(draft.ImageURLs) == 0 {
		draft.ImageURLs = preferredXiaohongshuImages(meta)
	}
	draft.Note = firstNonEmpty(strings.TrimSpace(draft.Note), "基于小红书笔记内容生成的 AI 草稿，建议回看原笔记补齐克数、火候和时间。")
	draft.ParsedContent = normalizeParsedContentDraft(draft.ParsedContent)
	draft.Summary = normalizeRecipeSummary(draft.Summary)

	if (len(draft.ParsedContent.MainIngredients) == 0 && len(draft.ParsedContent.SecondaryIngredients) == 0) || len(draft.ParsedContent.Steps) == 0 {
		fallback := summarizeXiaohongshuHeuristically(meta)
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

func preferredXiaohongshuImages(meta XiaohongshuParseResult) []string {
	images := draftImageURLs(meta.Images...)
	if len(images) > 0 {
		return images
	}
	return draftImageURLs(strings.TrimSpace(meta.CoverURL))
}

func firstImage(images []string) string {
	if len(images) == 0 {
		return ""
	}
	return strings.TrimSpace(images[0])
}

func normalizeXiaohongshuMediaURLs(values []string, limit int) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeXiaohongshuMediaURL(value)
		if normalized == "" {
			continue
		}
		items = append(items, normalized)
	}
	return dedupeStrings(items, limit)
}

func normalizeXiaohongshuMediaURL(value string) string {
	raw := strings.TrimSpace(value)
	switch {
	case strings.HasPrefix(raw, "//"):
		return "https:" + raw
	case strings.HasPrefix(raw, "http://"):
		return "https://" + strings.TrimPrefix(raw, "http://")
	default:
		return raw
	}
}
