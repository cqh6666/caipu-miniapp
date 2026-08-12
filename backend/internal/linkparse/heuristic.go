package linkparse

import (
	"regexp"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/recipecontent"
)

var (
	stepVerbPattern          = regexp.MustCompile(`(切|洗|腌|拌|加|放|倒|下锅|翻炒|炒|煎|炸|蒸|煮|炖|焖|焯|烤|淋|撒|搅|收汁|出锅|开吃|冷藏|静置)`)
	stepOrderPattern         = regexp.MustCompile(`^(先|再|然后|接着|最后|随后|第一步|第二步|第三步|第四步)`)
	ingredientUnitPattern    = regexp.MustCompile(`[\p{Han}A-Za-z][\p{Han}A-Za-z0-9()（）-]{0,14}\s*\d+(?:\.\d+)?\s*(?:g|kg|克|千克|ml|毫升|l|升|勺|汤匙|茶匙|匙|杯|个|颗|根|把|片|块|斤|两|袋|盒|碗)`)
	ingredientLoosePattern   = regexp.MustCompile(`[\p{Han}A-Za-z][\p{Han}A-Za-z0-9()（）-]{0,14}\s*(?:适量|少许)`)
	ingredientSpacingPattern = regexp.MustCompile(`([\p{Han}A-Za-z])(\d)`)
	summaryWhitespacePattern = regexp.MustCompile(`\s+`)
)

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
	return recipecontent.IngredientLabel(value)
}

func splitIngredientLines(lines []string) ([]string, []string) {
	return recipecontent.SplitIngredientLines(lines)
}

func buildParsedSteps(lines []string) []ParsedStep {
	return recipecontent.BuildSteps(lines)
}

func cleanParsedSteps(steps []ParsedStep) []ParsedStep {
	return recipecontent.CleanSteps(steps)
}

func compactParsedSteps(steps []ParsedStep) []ParsedStep {
	return recipecontent.CompactSteps(steps)
}

func inferParsedStepTitle(detail string, index int) string {
	return recipecontent.InferStepTitle(detail, index)
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
