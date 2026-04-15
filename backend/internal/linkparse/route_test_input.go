package linkparse

import "github.com/cqh6666/caipu-miniapp/backend/internal/airouter"

func BuildSummaryRouteTestInput() airouter.ChatCompletionInput {
	sample := XiaohongshuParseResult{
		Source:       "xiaohongshu",
		Link:         "https://www.xiaohongshu.com/explore/route-test-demo",
		CanonicalURL: "https://www.xiaohongshu.com/explore/route-test-demo",
		Title:        "番茄土豆炖牛腩教程来咯",
		Content:      "牛腩切块后冷水下锅焯水，番茄切块，土豆滚刀切块。锅里炒香番茄，加牛腩和热水炖煮，再下土豆焖到软烂收汁。",
		Transcript:   "",
		CoverURL:     "https://example.com/route-test-cover.jpg",
		Images:       []string{"https://example.com/route-test-cover.jpg"},
		Tags:         []string{"家常菜", "番茄牛腩", "炖菜"},
		Author:       "路由测试样例",
		NoteType:     "image",
	}
	return airouter.ChatCompletionInput{
		Messages:        buildXiaohongshuSummaryMessages(sample),
		Temperature:     floatPtr(0.2),
		MaxTokens:       intPtr(1024),
		ContentKind:     "route_test_real_summary",
		AdditionalMeta:  map[string]any{"sample_case": "xiaohongshu_recipe"},
		ValidateContent: func(content string) error { _, err := summaryDraftFromAIContent(content); return err },
	}
}

func BuildTitleRouteTestInput() airouter.ChatCompletionInput {
	return airouter.ChatCompletionInput{
		Messages:        buildTitleRefineMessages("番茄土豆炖牛腩教程来咯超级软烂"),
		ContentKind:     "route_test_real_title",
		AdditionalMeta:  map[string]any{"sample_case": "noisy_recipe_title"},
		ValidateContent: func(content string) error { _, err := parseTitleRefineContent(content); return err },
	}
}

func intPtr(value int) *int {
	return &value
}
