package recipecontent

import (
	"regexp"
	"strings"
)

const (
	MaxIngredientItems = 10
	MaxRawSteps        = 12
	MaxSteps           = 6
)

var (
	secondaryIngredientPattern          = regexp.MustCompile(`(?i)(常用配菜|基础调味|常用调味料|调味|葱|姜|蒜|香叶|桂皮|八角|花椒|胡椒|盐|糖|冰糖|白糖|红糖|生抽|老抽|蚝油|料酒|鸡精|味精|醋|陈醋|米醋|香醋|豆瓣酱|辣椒|小米椒|淀粉|清水|热水|食用油|香油|芝麻油|花椒粉|辣椒粉|五香粉|十三香|孜然|芝麻|香菜|葱花)`)
	secondaryIngredientExceptionPattern = regexp.MustCompile(`(?i)^(洋葱|红葱头|葱头)`)
	ingredientSuffixPattern             = regexp.MustCompile(`\s*(?:\d+(?:\.\d+)?\s*(?:g|kg|克|千克|ml|毫升|l|升|勺|汤匙|茶匙|匙|杯|个|颗|根|把|片|块|斤|两|袋|盒|碗)|半个|半颗|半根|半头|适量|少许)$`)
)

func Normalize(content Content) Content {
	mainIngredients := cleanNormalizedLines(content.MainIngredients, MaxIngredientItems)
	secondaryIngredients := cleanNormalizedLines(content.SecondaryIngredients, MaxIngredientItems)
	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 {
		mainIngredients, secondaryIngredients = SplitIngredientLines(content.legacyIngredients)
	}

	steps := CleanSteps(content.Steps)
	if len(steps) == 0 {
		steps = BuildSteps(content.legacySteps)
	}

	return Content{
		MainIngredients:      mainIngredients,
		SecondaryIngredients: secondaryIngredients,
		Steps:                steps,
	}
}

func CleanLines(lines []string) []string {
	items := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, exists := seen[line]; exists {
			continue
		}
		seen[line] = struct{}{}
		items = append(items, line)
	}
	return items
}

func SplitIngredientLines(lines []string) ([]string, []string) {
	cleaned := cleanNormalizedLines(lines, MaxIngredientItems)
	if len(cleaned) == 0 {
		return nil, nil
	}

	mainIngredients := make([]string, 0, 4)
	secondaryIngredients := make([]string, 0, len(cleaned))
	for _, line := range cleaned {
		if isSecondaryIngredientLine(line) {
			secondaryIngredients = append(secondaryIngredients, line)
			continue
		}
		mainIngredients = append(mainIngredients, line)
	}

	if len(mainIngredients) == 0 {
		limit := min(3, len(cleaned))
		mainIngredients = append(mainIngredients, cleaned[:limit]...)
		secondaryIngredients = append([]string{}, cleaned[limit:]...)
	}

	return mainIngredients, secondaryIngredients
}

func IngredientLabel(line string) string {
	label := strings.TrimSpace(line)
	label = ingredientSuffixPattern.ReplaceAllString(label, "")
	return strings.Trim(strings.TrimSpace(label), " ,，。")
}

func CleanSteps(steps []Step) []Step {
	items := make([]Step, 0, len(steps))
	seen := make(map[string]struct{}, len(steps))
	for index, step := range steps {
		title := strings.TrimSpace(step.Title)
		detail := cleanCandidateLine(step.Detail)
		if detail == "" {
			detail = title
		}
		if detail == "" {
			continue
		}
		if title == "" {
			title = InferStepTitle(detail, index)
		}
		key := title + "\x00" + detail
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, Step{Title: title, Detail: detail})
	}
	return CompactSteps(items)
}

func BuildSteps(lines []string) []Step {
	cleaned := cleanNormalizedLines(lines, MaxRawSteps)
	items := make([]Step, 0, len(cleaned))
	for index, line := range cleaned {
		detail := cleanCandidateLine(line)
		if detail == "" {
			continue
		}
		items = append(items, Step{
			Title:  InferStepTitle(detail, index),
			Detail: detail,
		})
	}
	return CompactSteps(items)
}

func CompactSteps(steps []Step) []Step {
	if len(steps) <= MaxSteps {
		return append([]Step{}, steps...)
	}

	items := make([]Step, 0, MaxSteps)
	for index := range MaxSteps {
		start := index * len(steps) / MaxSteps
		end := (index + 1) * len(steps) / MaxSteps
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
			title = InferStepTitle(group[0].Detail, index)
		}

		details := make([]string, 0, len(group))
		for _, step := range group {
			detail := cleanCandidateLine(step.Detail)
			if detail != "" {
				details = append(details, detail)
			}
		}
		if len(details) == 0 && title != "" {
			details = append(details, title)
		}

		items = append(items, Step{
			Title:  title,
			Detail: strings.Join(details, "；"),
		})
	}

	return items
}

func InferStepTitle(detail string, index int) string {
	switch {
	case strings.Contains(detail, "焯水") || strings.Contains(detail, "汆水"):
		if strings.Contains(detail, "腥") || strings.Contains(detail, "浮沫") {
			return "焯水去腥"
		}
		return "焯水备用"
	case strings.Contains(detail, "腌"):
		return "腌制入味"
	case strings.Contains(detail, "糖色") || strings.Contains(detail, "冰糖"):
		return "炒糖上色"
	case strings.Contains(detail, "爆香") || strings.Contains(detail, "炒香"):
		return "炒香底料"
	case strings.Contains(detail, "切") || strings.Contains(detail, "改刀"):
		return "切配备料"
	case strings.Contains(detail, "收汁"):
		return "收汁出锅"
	case strings.Contains(detail, "炖") || strings.Contains(detail, "焖"):
		return "小火慢炖"
	case strings.Contains(detail, "蒸"):
		return "上锅蒸熟"
	case strings.Contains(detail, "炸"):
		return "炸至金黄"
	case strings.Contains(detail, "煎"):
		return "煎香上色"
	case strings.Contains(detail, "烤"):
		return "烤至上色"
	case strings.Contains(detail, "煮"):
		return "煮至入味"
	case strings.Contains(detail, "拌"):
		return "拌匀调味"
	case strings.Contains(detail, "炒") || strings.Contains(detail, "翻炒"):
		return "翻炒入味"
	case strings.Contains(detail, "出锅"):
		return "调味出锅"
	case index == 0:
		return "处理食材"
	default:
		return "继续烹饪"
	}
}

func IngredientLines(content Content) []string {
	normalized := Normalize(content)
	return append(append([]string{}, normalized.MainIngredients...), normalized.SecondaryIngredients...)
}

func StepLines(content Content) []string {
	steps := Normalize(content).Steps
	items := make([]string, 0, len(steps))
	for _, step := range steps {
		items = append(items, step.Detail)
	}
	return items
}

func Equal(left, right Content) bool {
	left = Normalize(left)
	right = Normalize(right)
	return stringSlicesEqual(left.MainIngredients, right.MainIngredients) &&
		stringSlicesEqual(left.SecondaryIngredients, right.SecondaryIngredients) &&
		stepSlicesEqual(left.Steps, right.Steps)
}

func isSecondaryIngredientLine(line string) bool {
	label := IngredientLabel(line)
	return secondaryIngredientPattern.MatchString(label) && !secondaryIngredientExceptionPattern.MatchString(label)
}

func cleanNormalizedLines(lines []string, limit int) []string {
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		line = cleanCandidateLine(line)
		if line == "" {
			continue
		}
		duplicate := false
		for _, existing := range items {
			if strings.EqualFold(existing, line) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		items = append(items, line)
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items
}

func cleanCandidateLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, " ,，。！？!?:：()[]【】\"'")
	value = strings.ReplaceAll(value, "适量即可", "适量")
	value = strings.ReplaceAll(value, "适量就行", "适量")
	return strings.TrimSpace(value)
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func stepSlicesEqual(left, right []Step) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
