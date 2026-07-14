package recipe

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

var flowchartTipPattern = regexp.MustCompile(`(?i)(火候|口味|收汁|调味|腌|焯|大火|中火|小火|慢炖|软烂|嫩|脆|香|辣|酸甜|出汁|分钟|时间)`)

type flowchartPromptInput struct {
	Title                string       `json:"title"`
	Summary              string       `json:"summary"`
	MainIngredients      []string     `json:"mainIngredients"`
	SecondaryIngredients []string     `json:"secondaryIngredients"`
	Steps                []ParsedStep `json:"steps"`
	Keywords             []string     `json:"keywords,omitempty"`
	NoteTip              string       `json:"noteTip,omitempty"`
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

func buildFlowchartSourceHash(item Recipe) string {
	input, err := buildFlowchartPromptInput(item)
	if err != nil {
		return ""
	}
	return hashFlowchartPromptInput(input)
}

func floatPtr(value float64) *float64 {
	return &value
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
