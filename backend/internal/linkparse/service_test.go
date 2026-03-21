package linkparse

import (
	"net/url"
	"strings"
	"testing"
)

func TestExtractInputURL(t *testing.T) {
	t.Parallel()

	got, err := extractInputURL("做个标记 https://www.bilibili.com/video/BV1xx411c7mD?p=2。")
	if err != nil {
		t.Fatalf("extractInputURL returned error: %v", err)
	}

	want := "https://www.bilibili.com/video/BV1xx411c7mD?p=2"
	if got != want {
		t.Fatalf("extractInputURL = %q, want %q", got, want)
	}
}

func TestParseVideoRef(t *testing.T) {
	t.Parallel()

	u, err := url.Parse("https://www.bilibili.com/video/BV1xx411c7mD?p=3")
	if err != nil {
		t.Fatalf("url.Parse returned error: %v", err)
	}

	ref, ok := parseVideoRef(u)
	if !ok {
		t.Fatal("parseVideoRef returned false")
	}
	if ref.BVID != "BV1xx411c7mD" {
		t.Fatalf("parseVideoRef BVID = %q", ref.BVID)
	}
	if ref.Page != 3 {
		t.Fatalf("parseVideoRef Page = %d", ref.Page)
	}
}

func TestBuildSubtitleText(t *testing.T) {
	t.Parallel()

	text, count := buildSubtitleText(bilibiliSubtitleFile{
		Body: []struct {
			From    float64 `json:"from"`
			To      float64 `json:"to"`
			Content string  `json:"content"`
		}{
			{Content: "准备牛肉 300克"},
			{Content: " "},
			{Content: "锅里加油翻炒"},
		},
	})

	if count != 2 {
		t.Fatalf("buildSubtitleText count = %d, want 2", count)
	}
	if text != "准备牛肉 300克\n锅里加油翻炒" {
		t.Fatalf("buildSubtitleText text = %q", text)
	}
}

func TestSummarizeHeuristically(t *testing.T) {
	t.Parallel()

	result := summarizeHeuristically(BilibiliParseResult{
		Title:             "土豆烧牛肉",
		Link:              "https://www.bilibili.com/video/BV1xx411c7mD",
		CoverURL:          "https://i0.hdslb.com/demo.jpg",
		SubtitleAvailable: true,
	}, "准备牛肉 300克\n土豆 2个\n先把牛肉切块\n锅里加油下锅翻炒\n再加入土豆焖煮二十分钟\n最后撒葱花出锅")

	if len(result.ParsedContent.MainIngredients) < 2 {
		t.Fatalf("main ingredients too short: %#v", result.ParsedContent.MainIngredients)
	}
	if len(result.ParsedContent.Steps) < 3 {
		t.Fatalf("steps too short: %#v", result.ParsedContent.Steps)
	}
	if result.Ingredient == "" {
		t.Fatal("ingredient summary is empty")
	}
	if result.Summary == "" {
		t.Fatal("recipe summary should not be empty in heuristic mode")
	}
	if result.ParsedContent.Steps[0].Title == "" || result.ParsedContent.Steps[0].Detail == "" {
		t.Fatalf("structured step missing title/detail: %#v", result.ParsedContent.Steps[0])
	}
	if result.Link == "" {
		t.Fatal("link should be preserved")
	}
	if got, want := result.ImageURL, "https://i0.hdslb.com/demo.jpg"; got != want {
		t.Fatalf("ImageURL = %q, want %q", got, want)
	}
	if got, want := len(result.ImageURLs), 1; got != want {
		t.Fatalf("len(ImageURLs) = %d, want %d", got, want)
	}
}

func TestSplitIngredientLinesKeepsMultiplePrimaryIngredients(t *testing.T) {
	t.Parallel()

	mainIngredients, secondaryIngredients := splitIngredientLines([]string{
		"牛腩 500克",
		"番茄 3个",
		"土豆 2个",
		"胡萝卜 1根",
		"洋葱 半个",
		"盐 3克",
		"生抽 1勺",
	})

	if got, want := len(mainIngredients), 5; got != want {
		t.Fatalf("len(mainIngredients) = %d, want %d (%#v)", got, want, mainIngredients)
	}
	if got, want := mainIngredients[4], "洋葱 半个"; got != want {
		t.Fatalf("mainIngredients[4] = %q, want %q", got, want)
	}
	if got, want := len(secondaryIngredients), 2; got != want {
		t.Fatalf("len(secondaryIngredients) = %d, want %d (%#v)", got, want, secondaryIngredients)
	}
}

func TestBuildIngredientPromptRuleTextMentionsSupportingThenSeasoning(t *testing.T) {
	t.Parallel()

	rule := buildIngredientPromptRuleText()
	if !strings.Contains(rule, "先写配菜，再写调味") {
		t.Fatalf("ingredient rule should mention ordering, got %q", rule)
	}
	if !strings.Contains(rule, "mainIngredients 只放主菜体或主食材") {
		t.Fatalf("ingredient rule should constrain main ingredients, got %q", rule)
	}
}

func TestSanitizePreviewTitle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"【香菜牛肉最好吃的做法~-哔哩哔哩】", "香菜牛肉"},
		{"番茄土豆炖牛腩教程来咯～超级软烂！", "番茄土豆炖牛腩"},
		{"  红烧牛腩 - 小红书  ", "红烧牛腩"},
		{"红烧牛腩 就是这个味！ 店里十几块一份", "红烧牛腩"},
		{"【【我做了20年的拿手菜，西红柿土豆炖牛腩】-哔哩哔哩】", "西红柿土豆炖牛腩"},
		{"【如何用科学做出超级浓稠，鲜香入味的番茄炖牛腩【解构家常菜】-哔哩哔哩】", "番茄炖牛腩"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := sanitizePreviewTitle(tc.input); got != tc.want {
				t.Fatalf("sanitizePreviewTitle(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
