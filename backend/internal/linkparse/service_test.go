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
}
