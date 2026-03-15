package linkparse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type xiaohongshuClient struct {
	baseURL  string
	provider string
	apiKey   string
	client   *http.Client
}

type xhsSidecarParseRequest struct {
	Input         string `json:"input"`
	Provider      string `json:"provider,omitempty"`
	IncludeImages bool   `json:"includeImages"`
	IncludeDebug  bool   `json:"includeDebug"`
}

type xhsSidecarParseResponse struct {
	OK                bool   `json:"ok"`
	Platform          string `json:"platform"`
	ProviderRequested string `json:"providerRequested"`
	ProviderUsed      string `json:"providerUsed"`
	Normalized        struct {
		ShareURL     string `json:"shareUrl"`
		CanonicalURL string `json:"canonicalUrl"`
		NoteID       string `json:"noteId"`
		XSECToken    string `json:"xsecToken"`
	} `json:"normalized"`
	Note struct {
		Title    string   `json:"title"`
		Content  string   `json:"content"`
		Tags     []string `json:"tags"`
		Images   []string `json:"images"`
		Videos   []string `json:"videos"`
		CoverURL string   `json:"coverUrl"`
		Author   struct {
			Name      string `json:"name"`
			AvatarURL string `json:"avatarUrl"`
		} `json:"author"`
		NoteType  string `json:"noteType"`
		Likes     int64  `json:"likes"`
		Comments  int64  `json:"comments"`
		Favorites int64  `json:"favorites"`
	} `json:"note"`
	Error *struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Retryable bool   `json:"retryable"`
	} `json:"error,omitempty"`
	Warnings []string `json:"warnings"`
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
		result, err := s.ParseXiaohongshu(ctx, rawInput)
		if err != nil {
			return RecipeParseOutcome{}, err
		}
		return RecipeParseOutcome{
			Source:      result.Source,
			SummaryMode: result.SummaryMode,
			RecipeDraft: result.RecipeDraft,
		}, nil
	default:
		return RecipeParseOutcome{}, common.NewAppError(common.CodeBadRequest, "unsupported auto-parse link", http.StatusBadRequest)
	}
}

func (s *Service) ParseXiaohongshu(ctx context.Context, rawInput string) (XiaohongshuParseResult, error) {
	if s == nil || s.xhs == nil {
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeInternalServer, "xiaohongshu sidecar is not configured", http.StatusInternalServerError)
	}

	inputURL, err := extractSupportedURL(rawInput)
	if err != nil {
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, "invalid xiaohongshu url", http.StatusBadRequest)
	}
	if !SupportsXiaohongshuURL(inputURL) {
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, "invalid xiaohongshu url", http.StatusBadRequest)
	}

	payload := xhsSidecarParseRequest{
		Input:         rawInput,
		Provider:      s.xhs.provider,
		IncludeImages: true,
		IncludeDebug:  false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return XiaohongshuParseResult{}, common.ErrInternal.WithErr(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.xhs.baseURL+"/v1/parse/xiaohongshu", bytes.NewReader(body))
	if err != nil {
		return XiaohongshuParseResult{}, common.ErrInternal.WithErr(err)
	}

	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(s.xhs.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(s.xhs.apiKey))
	}

	resp, err := s.xhs.client.Do(req)
	if err != nil {
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, "request to xiaohongshu sidecar failed", http.StatusBadRequest).WithErr(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = "xiaohongshu sidecar request failed"
		}
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, message, http.StatusBadRequest)
	}

	var parsed xhsSidecarParseResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, "failed to decode xiaohongshu sidecar response", http.StatusBadRequest).WithErr(err)
	}
	if !parsed.OK {
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, strings.TrimSpace(parsed.Error.Message), http.StatusBadRequest)
		}
		return XiaohongshuParseResult{}, common.NewAppError(common.CodeBadRequest, "xiaohongshu parse failed", http.StatusBadRequest)
	}

	result := XiaohongshuParseResult{
		Source:            "xiaohongshu",
		Link:              firstNonEmpty(parsed.Normalized.ShareURL, inputURL),
		CanonicalURL:      firstNonEmpty(parsed.Normalized.CanonicalURL, inputURL),
		ProviderRequested: firstNonEmpty(parsed.ProviderRequested, s.xhs.provider, "auto"),
		ProviderUsed:      strings.TrimSpace(parsed.ProviderUsed),
		Title:             strings.TrimSpace(parsed.Note.Title),
		Content:           strings.TrimSpace(parsed.Note.Content),
		CoverURL:          strings.TrimSpace(parsed.Note.CoverURL),
		Images:            dedupeStrings(cleanLines(parsed.Note.Images), 12),
		Videos:            dedupeStrings(cleanLines(parsed.Note.Videos), 4),
		Tags:              dedupeStrings(cleanLines(parsed.Note.Tags), 12),
		Author:            strings.TrimSpace(parsed.Note.Author.Name),
		NoteType:          strings.TrimSpace(parsed.Note.NoteType),
		NoteID:            strings.TrimSpace(parsed.Normalized.NoteID),
		XSECToken:         strings.TrimSpace(parsed.Normalized.XSECToken),
		Warnings:          parsed.Warnings,
	}

	if s.ai != nil {
		draft, err := s.ai.summarizeXiaohongshu(ctx, result)
		if err == nil {
			result.SummaryMode = "ai"
			result.RecipeDraft = normalizeXiaohongshuDraft(result, draft)
			return result, nil
		}
		result.Warnings = append(result.Warnings, "AI 总结暂时不可用，已回退到规则总结。")
	}

	result.SummaryMode = "heuristic"
	result.RecipeDraft = summarizeXiaohongshuHeuristically(result)
	return result, nil
}

func summarizeXiaohongshuHeuristically(meta XiaohongshuParseResult) RecipeDraft {
	lines := collectCandidateLines(meta.Content, strings.Join(meta.Tags, "\n"))
	ingredients := extractIngredientLines(lines)
	steps := extractStepLines(lines)

	if len(ingredients) == 0 {
		ingredients = fallbackIngredients(meta.Title)
	}
	if len(steps) == 0 {
		steps = []string{
			"先结合小红书原文确认这道菜的主食材和用量。",
			"按原文提到的顺序整理预处理、调味和烹饪步骤。",
			"做之前建议回看原链接，补齐克数、火候和时间等细节。",
		}
	}

	return RecipeDraft{
		Title:      firstNonEmpty(meta.Title, "小红书图文菜谱草稿"),
		Ingredient: buildIngredientSummary(ingredients, meta.Title),
		Link:       firstNonEmpty(meta.CanonicalURL, meta.Link),
		Note:       buildXiaohongshuHeuristicNote(meta),
		ParsedContent: ParsedContent{
			Ingredients: ingredients,
			Steps:       steps,
		},
	}
}

func buildXiaohongshuHeuristicNote(meta XiaohongshuParseResult) string {
	base := "基于小红书图文正文生成的草稿，建议做菜前回看原笔记核对食材克数、火候和图片里的细节。"
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
				Content: "你是一个菜谱整理助手。请根据小红书图文笔记正文、标签和图片描述线索，提炼适合家庭复刻的菜谱草稿。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\",\"ingredient\":\"\",\"ingredients\":[],\"steps\":[],\"note\":\"\"}。ingredients 和 steps 各返回 2 到 8 条，尽量保留明确的食材名、用量、顺序、火候和动作；不确定的信息不要编造，可以在 note 里提醒用户回看原笔记和配图确认。",
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

func buildXiaohongshuAISummaryPrompt(result XiaohongshuParseResult) string {
	content := result.Content
	if len([]rune(content)) > defaultPromptCharLimit {
		content = string([]rune(content)[:defaultPromptCharLimit])
	}

	var builder strings.Builder
	builder.WriteString("请整理这条小红书图文笔记里的菜谱信息。\n")
	builder.WriteString("标题: " + firstNonEmpty(result.Title, "未知标题") + "\n")
	if result.Author != "" {
		builder.WriteString("作者: " + result.Author + "\n")
	}
	builder.WriteString("链接: " + firstNonEmpty(result.CanonicalURL, result.Link) + "\n")
	if len(result.Tags) > 0 {
		builder.WriteString("标签: " + strings.Join(result.Tags, "、") + "\n")
	}
	if len(result.Images) > 0 {
		builder.WriteString(fmt.Sprintf("图片数量: %d\n", len(result.Images)))
	}
	builder.WriteString("正文内容:\n")
	builder.WriteString(content)
	return builder.String()
}

func normalizeXiaohongshuDraft(meta XiaohongshuParseResult, draft RecipeDraft) RecipeDraft {
	draft.Title = firstNonEmpty(strings.TrimSpace(draft.Title), meta.Title, "小红书图文菜谱草稿")
	draft.Ingredient = firstNonEmpty(strings.TrimSpace(draft.Ingredient), strings.TrimSpace(meta.Title))
	draft.Link = firstNonEmpty(meta.CanonicalURL, meta.Link)
	draft.Note = firstNonEmpty(strings.TrimSpace(draft.Note), "基于小红书图文内容生成的 AI 草稿，建议回看原笔记和配图补齐克数、火候和时间。")
	draft.ParsedContent.Ingredients = dedupeStrings(cleanLines(draft.ParsedContent.Ingredients), 10)
	draft.ParsedContent.Steps = dedupeStrings(cleanLines(draft.ParsedContent.Steps), 8)

	if len(draft.ParsedContent.Ingredients) == 0 || len(draft.ParsedContent.Steps) == 0 {
		fallback := summarizeXiaohongshuHeuristically(meta)
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
