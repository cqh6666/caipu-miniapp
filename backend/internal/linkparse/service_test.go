package linkparse

import (
	"net/url"
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

	if len(result.ParsedContent.Ingredients) < 2 {
		t.Fatalf("ingredients too short: %#v", result.ParsedContent.Ingredients)
	}
	if len(result.ParsedContent.Steps) < 3 {
		t.Fatalf("steps too short: %#v", result.ParsedContent.Steps)
	}
	if result.Ingredient == "" {
		t.Fatal("ingredient summary is empty")
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

func TestSanitizePreviewTitle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"【香菜牛肉最好吃的做法~-哔哩哔哩】", "香菜牛肉最好吃的做法"},
		{"番茄土豆炖牛腩教程来咯～", "番茄土豆炖牛腩教程来咯"},
		{"  红烧牛腩 - 小红书  ", "红烧牛腩"},
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
