package recipe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

var (
	flowchartMarkdownImagePattern = regexp.MustCompile(`!\[[^\]]*\]\((https?://[^\s)]+)\)`)
	flowchartPlainURLPattern      = regexp.MustCompile(`https?://[^\s)]+`)
	flowchartTipPattern           = regexp.MustCompile(`(?i)(火候|口味|收汁|调味|腌|焯|大火|中火|小火|慢炖|软烂|嫩|脆|香|辣|酸甜|出汁|分钟|时间)`)
)

type FlowchartOptions struct {
	BaseURL             string
	APIKey              string
	Model               string
	Timeout             time.Duration
	RuntimeConfigLoader RuntimeConfigLoader
	Tracker             audit.Tracker
}

type FlowchartGenerator struct {
	defaultConfig FlowchartRuntimeConfig
	configLoader  RuntimeConfigLoader
	tracker       audit.Tracker
	uploader      *upload.Service
}

type FlowchartResult struct {
	ImageURL   string
	SourceHash string
	Provider   string
	Model      string
}

type flowchartClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
	tracker    audit.Tracker
}

type RuntimeConfigLoader func(context.Context) FlowchartRuntimeConfig

type FlowchartRuntimeConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

type flowchartPromptInput struct {
	Title                string       `json:"title"`
	Summary              string       `json:"summary"`
	MainIngredients      []string     `json:"mainIngredients"`
	SecondaryIngredients []string     `json:"secondaryIngredients"`
	Steps                []ParsedStep `json:"steps"`
	Keywords             []string     `json:"keywords,omitempty"`
	NoteTip              string       `json:"noteTip,omitempty"`
}

type flowchartChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []flowchartChatMessage `json:"messages"`
	Temperature float64                `json:"temperature"`
}

type flowchartChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type flowchartChatResponse struct {
	Choices []struct {
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewFlowchartGenerator(opts FlowchartOptions, uploader *upload.Service) *FlowchartGenerator {
	if uploader == nil {
		return nil
	}

	return &FlowchartGenerator{
		defaultConfig: FlowchartRuntimeConfig{
			BaseURL: strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
			APIKey:  strings.TrimSpace(opts.APIKey),
			Model:   strings.TrimSpace(opts.Model),
			Timeout: opts.Timeout,
		},
		configLoader: opts.RuntimeConfigLoader,
		tracker:      opts.Tracker,
		uploader:     uploader,
	}
}

func (g *FlowchartGenerator) IsConfigured() bool {
	if g == nil || g.uploader == nil {
		return false
	}
	if strings.TrimSpace(g.defaultConfig.Model) != "" && strings.TrimSpace(g.defaultConfig.BaseURL) != "" {
		return true
	}
	return g.configLoader != nil
}

func (g *FlowchartGenerator) Generate(ctx context.Context, item Recipe) (FlowchartResult, error) {
	client := g.clientFor(ctx)
	if client == nil || !g.IsConfigured() {
		return FlowchartResult{}, common.NewAppError(common.CodeInternalServer, "flowchart generation is not configured", http.StatusServiceUnavailable)
	}

	input, err := buildFlowchartPromptInput(item)
	if err != nil {
		return FlowchartResult{}, err
	}

	content, err := client.generate(ctx, buildFlowchartPrompt(input))
	if err != nil {
		return FlowchartResult{}, err
	}

	remoteURL := extractFlowchartImageURL(content)
	if remoteURL == "" {
		return FlowchartResult{}, common.NewAppError(common.CodeInternalServer, "flowchart generation did not return an image", http.StatusBadGateway)
	}

	image, err := g.uploader.SaveRemoteImage(ctx, remoteURL)
	if err != nil {
		return FlowchartResult{}, err
	}

	return FlowchartResult{
		ImageURL:   image.URL,
		SourceHash: hashFlowchartPromptInput(input),
		Provider:   "openai-compatible",
		Model:      client.model,
	}, nil
}

func (g *FlowchartGenerator) clientFor(ctx context.Context) *flowchartClient {
	if g == nil {
		return nil
	}
	cfg := g.defaultConfig
	if g.configLoader != nil {
		runtimeCfg := g.configLoader(ctx)
		if strings.TrimSpace(runtimeCfg.BaseURL) != "" {
			cfg.BaseURL = strings.TrimSpace(runtimeCfg.BaseURL)
		}
		if strings.TrimSpace(runtimeCfg.APIKey) != "" {
			cfg.APIKey = strings.TrimSpace(runtimeCfg.APIKey)
		}
		if strings.TrimSpace(runtimeCfg.Model) != "" {
			cfg.Model = strings.TrimSpace(runtimeCfg.Model)
		}
		if runtimeCfg.Timeout > 0 {
			cfg.Timeout = runtimeCfg.Timeout
		}
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.Model) == "" {
		return nil
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 45 * time.Second
	}
	return &flowchartClient{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		apiKey:     strings.TrimSpace(cfg.APIKey),
		model:      strings.TrimSpace(cfg.Model),
		httpClient: &http.Client{Timeout: cfg.Timeout},
		tracker:    g.tracker,
	}
}

func (c *flowchartClient) generate(ctx context.Context, prompt string) (string, error) {
	startedAt := time.Now()
	logCall := func(status string, httpStatus int, err error) {
		if c == nil || c.tracker == nil {
			return
		}
		jobCtx, ok := audit.CurrentJobContext(ctx)
		if !ok || jobCtx.JobRunID <= 0 {
			return
		}
		_ = c.tracker.LogCall(ctx, audit.CallLogInput{
			JobRunID:     jobCtx.JobRunID,
			Scene:        jobCtx.Scene,
			Provider:     "openai-compatible",
			Endpoint:     "/chat/completions",
			Model:        c.model,
			Status:       status,
			HTTPStatus:   httpStatus,
			LatencyMS:    time.Since(startedAt).Milliseconds(),
			ErrorType:    audit.ErrorTypeFromError(err),
			ErrorMessage: flowchartErrorMessage(err),
			RequestID:    common.RequestID(ctx),
			Meta: map[string]any{
				"content_kind": "flowchart",
			},
		})
	}

	payload := flowchartChatRequest{
		Model:       c.model,
		Temperature: 0.4,
		Messages: []flowchartChatMessage{
			{
				Role:    "system",
				Content: "你是一个料理流程图生成助手。请严格按用户要求生成手绘风格料理流程信息图，不要输出额外解释。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logCall(audit.CallStatusFailed, 0, err)
		return "", common.ErrInternal.WithErr(fmt.Errorf("marshal flowchart request: %w", err))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		logCall(audit.CallStatusFailed, 0, err)
		return "", common.ErrInternal.WithErr(fmt.Errorf("build flowchart request: %w", err))
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		callErr := newFlowchartRequestError(err, c.httpClient.Timeout)
		logCall(audit.CallStatusFromError(err), 0, callErr)
		return "", callErr
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = fmt.Sprintf("flowchart request failed with status %d", resp.StatusCode)
		}
		callErr := common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return "", callErr
	}

	var parsed flowchartChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		cause := truncateString(flowchartErrorCause(err), 160)
		if cause == "" {
			cause = "decode failed"
		}
		callErr := common.NewAppError(
			common.CodeInternalServer,
			"invalid flowchart response: "+cause,
			http.StatusBadGateway,
		).WithErr(err)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return "", callErr
	}
	if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
		callErr := common.NewAppError(common.CodeInternalServer, strings.TrimSpace(parsed.Error.Message), http.StatusBadGateway)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return "", callErr
	}
	if len(parsed.Choices) == 0 {
		callErr := common.NewAppError(common.CodeInternalServer, "flowchart response contained no choices", http.StatusBadGateway)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return "", callErr
	}

	content := extractFlowchartMessageContent(parsed.Choices[0].Message.Content)
	if content == "" {
		callErr := common.NewAppError(common.CodeInternalServer, "flowchart response was empty", http.StatusBadGateway)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return "", callErr
	}

	logCall(audit.CallStatusSuccess, resp.StatusCode, nil)

	return content, nil
}

func newFlowchartRequestError(err error, timeout time.Duration) error {
	cause := flowchartErrorCause(err)
	if isFlowchartTimeoutError(err) {
		message := "流程图生成超时，上游生图响应较慢"
		if timeout > 0 {
			message = fmt.Sprintf("%s（已等待 %s）", message, timeout.Round(time.Second))
		}
		if cause != "" {
			message += ": " + cause
		}
		return common.NewAppError(common.CodeInternalServer, message, http.StatusBadGateway).WithErr(err)
	}

	if cause == "" {
		cause = "unknown error"
	}

	return common.NewAppError(
		common.CodeInternalServer,
		"flowchart request failed: "+truncateString(cause, 180),
		http.StatusBadGateway,
	).WithErr(err)
}

func isFlowchartTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func flowchartErrorCause(err error) string {
	if err == nil {
		return ""
	}

	var appErr *common.AppError
	if errors.As(err, &appErr) {
		parts := make([]string, 0, 2)
		message := strings.TrimSpace(appErr.Message)
		if message != "" {
			parts = append(parts, message)
		}
		if appErr.Err != nil {
			cause := deepestError(appErr.Err)
			if cause != "" && (message == "" || !strings.Contains(message, cause)) {
				parts = append(parts, cause)
			}
		}
		return strings.Join(parts, ": ")
	}

	return deepestError(err)
}

func flowchartErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func deepestError(err error) string {
	if err == nil {
		return ""
	}

	current := err
	for {
		next := errors.Unwrap(current)
		if next == nil {
			break
		}
		current = next
	}

	return strings.TrimSpace(current.Error())
}

func buildFlowchartPromptInput(item Recipe) (flowchartPromptInput, error) {
	if !canGenerateFlowchartForRecipe(item) {
		return flowchartPromptInput{}, common.NewAppError(common.CodeBadRequest, "please complete key recipe steps before generating a flowchart", http.StatusBadRequest)
	}

	mainIngredients := cleanLines(item.ParsedContent.MainIngredients)
	secondaryIngredients := cleanLines(item.ParsedContent.SecondaryIngredients)
	steps := compactFlowchartSteps(cleanParsedSteps(item.ParsedContent.Steps))
	summary := buildEffectiveFlowchartSummary(item, steps)
	noteTip := extractFlowchartNoteTip(item.Note)

	input := flowchartPromptInput{
		Title:                strings.TrimSpace(item.Title),
		Summary:              summary,
		MainIngredients:      mainIngredients,
		SecondaryIngredients: secondaryIngredients,
		Steps:                steps,
		NoteTip:              noteTip,
	}
	input.Keywords = buildFlowchartKeywords(input)
	return input, nil
}

func buildFlowchartPrompt(input flowchartPromptInput) string {
	var builder strings.Builder

	if len(input.Steps) == 6 {
		builder.WriteString("请根据输入内容生成一张简洁的卡通手绘流程信息图：\n\n")
		builder.WriteString("- 横版 16:9 构图。\n")
		builder.WriteString("- 所有图像和文字都使用手绘风格，不要出现写实元素。\n")
		builder.WriteString("- 只保留大标题、副标题、6 个步骤卡片和连接箭头。\n")
		builder.WriteString("- 布局使用 2 x 3 网格：第一行 3 步，第二行 3 步。\n")
		builder.WriteString("- 不要单独绘制食材清单大区块。\n")
		builder.WriteString("- 不要在底部添加任何关键词标签、贴纸、吊牌、徽章或角标。\n")
		builder.WriteString("- 每个步骤卡片里的插画必须严格对应当前步骤的动作、锅内状态和食材变化。\n")
		builder.WriteString("- 每个步骤只保留一行文案，不要重复排版。\n")
		builder.WriteString("- 语言使用中文。\n\n")
		builder.WriteString("输入内容：\n")
		builder.WriteString("菜名：" + input.Title + "\n")
		builder.WriteString("副标题：" + fallbackPromptText(input.Summary, "可为空") + "\n")
		builder.WriteString("主料：" + fallbackPromptText(strings.Join(input.MainIngredients, "、"), "可为空") + "\n")
		builder.WriteString("辅料：" + fallbackPromptText(strings.Join(input.SecondaryIngredients, "、"), "可为空") + "\n")
		builder.WriteString("步骤：\n")
		for index, step := range input.Steps {
			builder.WriteString(fmt.Sprintf("%d. %s\n", index+1, truncateString(strings.TrimSpace(step.Title), 12)))
		}
		builder.WriteString("步骤对应动作说明：\n")
		for index, step := range input.Steps {
			builder.WriteString(fmt.Sprintf("%d. %s\n", index+1, truncateString(strings.TrimSpace(step.Detail), 48)))
		}
		if len(input.Keywords) > 0 {
			builder.WriteString("可选关键词：" + strings.Join(input.Keywords, "、") + "\n")
		}
		if input.NoteTip != "" {
			builder.WriteString("补充提示：" + input.NoteTip + "\n")
		}
		builder.WriteString("\n希望画面：\n")
		builder.WriteString("- 标题为“" + input.Title + "流程图”\n")
		builder.WriteString("- 6 个步骤模块用 2 x 3 布局排开\n")
		builder.WriteString("- 箭头按阅读顺序连接步骤\n")
		builder.WriteString("- 每步一个对应动作的手绘小插画\n")
		builder.WriteString("- 整体像清爽的料理步骤卡\n")
		return builder.String()
	}

	builder.WriteString("请根据输入内容生成一张更简洁的卡通手绘流程信息图：\n\n")
	builder.WriteString("- 横版 16:9 构图。\n")
	builder.WriteString("- 整体要简洁，重点突出步骤，不要堆太多说明文字。\n")
	builder.WriteString("- 所有图像和文字都使用手绘风格，不要出现写实元素。\n")
	builder.WriteString("- 只保留大标题、副标题、步骤卡片和连接箭头。\n")
	builder.WriteString("- 不要单独绘制食材清单大区块。\n")
	builder.WriteString("- 关键词仅用于帮助理解内容，不要渲染成底部标签、贴纸、吊牌、徽章或角标。\n")
	builder.WriteString("- 每个步骤卡片里的插画必须严格对应当前步骤的动作、锅内状态和食材变化。\n")
	builder.WriteString("- 步骤卡片按从左到右顺序排列，编号与步骤内容一一对应。\n")
	builder.WriteString("- 每一步文案用简短中文，尽量控制在一两行内。\n")
	builder.WriteString("- 语言使用中文。\n\n")
	builder.WriteString("输入内容：\n")
	builder.WriteString("菜名：" + input.Title + "\n")
	builder.WriteString("副标题：" + fallbackPromptText(input.Summary, "可为空") + "\n")
	builder.WriteString("主料：" + fallbackPromptText(strings.Join(input.MainIngredients, "、"), "可为空") + "\n")
	builder.WriteString("辅料：" + fallbackPromptText(strings.Join(input.SecondaryIngredients, "、"), "可为空") + "\n")
	builder.WriteString("步骤：\n")
	for index, step := range input.Steps {
		builder.WriteString(fmt.Sprintf("%d. %s：%s\n", index+1, truncateString(strings.TrimSpace(step.Title), 12), truncateString(strings.TrimSpace(step.Detail), 42)))
	}
	if len(input.Keywords) > 0 {
		builder.WriteString("可选关键词：" + strings.Join(input.Keywords, "、") + "\n")
	}
	if input.NoteTip != "" {
		builder.WriteString("补充提示：" + input.NoteTip + "\n")
	}
	builder.WriteString("\n希望画面：\n")
	builder.WriteString("- 标题为“" + input.Title + "流程图”\n")
	builder.WriteString(fmt.Sprintf("- 用 %d 个步骤模块展示流程\n", len(input.Steps)))
	builder.WriteString("- 箭头连接步骤\n")
	builder.WriteString("- 每步一个对应动作的手绘小插画\n")
	builder.WriteString("- 不要底部关键词标签\n")
	builder.WriteString("- 整体像清爽、易读的料理信息卡\n")
	return builder.String()
}

func compactFlowchartSteps(steps []ParsedStep) []ParsedStep {
	if len(steps) <= 6 {
		return append([]ParsedStep{}, steps...)
	}

	limit := 6
	items := make([]ParsedStep, 0, limit)
	for index := 0; index < limit; index++ {
		start := index * len(steps) / limit
		end := (index + 1) * len(steps) / limit
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
			detail := strings.TrimSpace(step.Detail)
			if detail == "" {
				continue
			}
			details = append(details, detail)
		}
		items = append(items, ParsedStep{
			Title:  truncateString(title, 12),
			Detail: truncateString(strings.Join(details, "；"), 72),
		})
	}

	return items
}

func buildEffectiveFlowchartSummary(item Recipe, steps []ParsedStep) string {
	summary := strings.TrimSpace(item.Summary)
	if summary != "" {
		return truncateString(summary, 24)
	}
	if len(steps) == 0 {
		return ""
	}

	first := strings.TrimSpace(steps[0].Title)
	second := ""
	for _, step := range steps[1:] {
		if title := strings.TrimSpace(step.Title); title != "" && title != first {
			second = title
			break
		}
	}

	switch {
	case first != "" && second != "":
		return truncateString("先"+first+"，再"+second, 24)
	case first != "":
		return truncateString(first, 24)
	default:
		return ""
	}
}

func extractFlowchartNoteTip(note string) string {
	candidates := strings.FieldsFunc(strings.TrimSpace(note), func(r rune) bool {
		return r == '\n' || r == '。' || r == '！' || r == '!' || r == '；' || r == ';'
	})
	for _, candidate := range candidates {
		text := strings.TrimSpace(candidate)
		if text == "" {
			continue
		}
		if flowchartTipPattern.MatchString(text) {
			return truncateString(text, 48)
		}
	}
	return ""
}

func buildFlowchartKeywords(input flowchartPromptInput) []string {
	items := make([]string, 0, 6)
	seen := make(map[string]struct{}, 6)
	appendKeyword := func(value string) {
		value = ingredientLabelFromLine(value)
		value = truncateString(strings.TrimSpace(value), 8)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}

	for _, ingredient := range input.MainIngredients {
		appendKeyword(ingredient)
		if len(items) >= 4 {
			break
		}
	}
	for _, ingredient := range input.SecondaryIngredients {
		appendKeyword(ingredient)
		if len(items) >= 5 {
			break
		}
	}
	for _, step := range input.Steps {
		appendKeyword(step.Title)
		if len(items) >= 6 {
			break
		}
	}

	return items
}

func extractFlowchartMessageContent(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		items := make([]string, 0, len(parts))
		for _, part := range parts {
			if strings.TrimSpace(part.Text) == "" {
				continue
			}
			items = append(items, strings.TrimSpace(part.Text))
		}
		return strings.TrimSpace(strings.Join(items, "\n"))
	}

	return strings.TrimSpace(string(raw))
}

func extractFlowchartImageURL(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	if matches := flowchartMarkdownImagePattern.FindStringSubmatch(content); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	for _, candidate := range flowchartPlainURLPattern.FindAllString(content, -1) {
		candidate = strings.TrimSpace(strings.TrimRight(candidate, "])}>.,;!\"'"))
		if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
			return candidate
		}
	}

	return ""
}

func buildFlowchartSourceHash(item Recipe) string {
	input, err := buildFlowchartPromptInput(item)
	if err != nil {
		return ""
	}
	return hashFlowchartPromptInput(input)
}

func hashFlowchartPromptInput(input flowchartPromptInput) string {
	body, err := json.Marshal(input)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func fallbackPromptText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
