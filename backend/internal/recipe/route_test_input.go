package recipe

import (
	"fmt"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
)

func BuildFlowchartRouteTestInput() airouter.ChatCompletionInput {
	item := Recipe{
		Title:   "番茄土豆炖牛腩",
		Summary: "先焯水去腥，再慢炖至软烂",
		Note:    "最后开盖大火收汁",
		ParsedContent: ParsedContent{
			MainIngredients:      []string{"牛腩 500 克", "番茄 3 个", "土豆 2 个"},
			SecondaryIngredients: []string{"姜片", "葱段", "盐", "生抽"},
			Steps: []ParsedStep{
				{Title: "处理牛腩", Detail: "牛腩切块后冷水下锅焯水，捞出洗净备用。"},
				{Title: "炒出番茄汁", Detail: "番茄切块下锅翻炒到出汁，加入姜葱炒香。"},
				{Title: "下锅炖煮", Detail: "放入牛腩和热水，大火煮开后转小火慢炖。"},
				{Title: "加入土豆", Detail: "牛腩炖到七成熟时放入土豆块继续焖煮。"},
				{Title: "调味收汁", Detail: "加入盐和生抽调味，最后开盖把汤汁收浓。"},
			},
		},
	}
	input, _ := buildFlowchartPromptInput(item)
	prompt := buildFlowchartPrompt(input)
	return airouter.ChatCompletionInput{
		Messages: []airouter.ChatMessage{
			{
				Role:    "system",
				Content: "你是一个料理流程图生成助手。请严格按用户要求生成手绘风格料理流程信息图，不要输出额外解释。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature:    floatPtr(0.4),
		ContentKind:    "route_test_real_flowchart",
		AdditionalMeta: map[string]any{"sample_case": "recipe_flowchart"},
		ValidateContent: func(content string) error {
			if extractFlowchartImageURL(content) == "" {
				return fmt.Errorf("flowchart generation did not return an image")
			}
			return nil
		},
	}
}
