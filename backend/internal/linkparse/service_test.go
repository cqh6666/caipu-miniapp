package linkparse

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type fakeAIRouter struct {
	available map[airouter.Scene]bool
	content   string
	err       error
	input     airouter.ChatCompletionInput
	scene     airouter.Scene
}

func (f *fakeAIRouter) IsSceneAvailable(_ context.Context, scene airouter.Scene) bool {
	return f != nil && f.available[scene]
}

func (f *fakeAIRouter) RouteChat(_ context.Context, scene airouter.Scene, input airouter.ChatCompletionInput) (airouter.ChatCompletionResult, error) {
	f.scene = scene
	f.input = input
	result := airouter.ChatCompletionResult{
		Content:      f.content,
		ProviderID:   "fake-provider",
		Model:        "fake-model",
		AttemptCount: 1,
	}
	if f.err != nil {
		return result, f.err
	}
	if input.ValidateContent != nil {
		if err := input.ValidateContent(f.content); err != nil {
			return result, err
		}
	}
	return result, nil
}

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

func TestSupportedPlatformHostsRequireDNSLabelBoundary(t *testing.T) {
	tests := []struct {
		url       string
		supported bool
	}{
		{url: "https://www.bilibili.com/video/BV1xx411c7mD", supported: true},
		{url: "https://b23.tv/abc", supported: true},
		{url: "https://bilibili.com.attacker.example/video/BV1xx411c7mD", supported: false},
		{url: "https://user@bilibili.com/video/BV1xx411c7mD", supported: false},
		{url: "ftp://bilibili.com/video/BV1xx411c7mD", supported: false},
	}
	for _, tt := range tests {
		if got := SupportsBilibiliURL(tt.url); got != tt.supported {
			t.Fatalf("SupportsBilibiliURL(%q) = %t, want %t", tt.url, got, tt.supported)
		}
	}
	if SupportsXiaohongshuURL("https://xiaohongshu.com.attacker.example/explore/1") {
		t.Fatal("xiaohongshu fake suffix must be rejected")
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

func TestCleanParsedStepsCompactsToSix(t *testing.T) {
	t.Parallel()

	steps := make([]ParsedStep, 0, 8)
	for index := 1; index <= 8; index++ {
		steps = append(steps, ParsedStep{
			Title:  fmt.Sprintf("第%d步", index),
			Detail: fmt.Sprintf("第%d步处理", index),
		})
	}

	got := cleanParsedSteps(steps)
	if len(got) != 6 {
		t.Fatalf("len(cleanParsedSteps) = %d, want 6", len(got))
	}
	if !strings.Contains(got[5].Detail, "第7步处理") || !strings.Contains(got[5].Detail, "第8步处理") {
		t.Fatalf("last compacted step should preserve trailing actions, got %#v", got[5])
	}
}

func TestBuildParsedStepsCompactsToSix(t *testing.T) {
	t.Parallel()

	lines := []string{
		"第1步处理",
		"第2步处理",
		"第3步处理",
		"第4步处理",
		"第5步处理",
		"第6步处理",
		"第7步处理",
		"第8步处理",
	}

	got := buildParsedSteps(lines)
	if len(got) != 6 {
		t.Fatalf("len(buildParsedSteps) = %d, want 6", len(got))
	}
	if !strings.Contains(got[5].Detail, "第7步处理") || !strings.Contains(got[5].Detail, "第8步处理") {
		t.Fatalf("last compacted step should preserve trailing actions, got %#v", got[5])
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

func TestFinalizePreviewTitleDefaultsToRuleSource(t *testing.T) {
	t.Parallel()

	service := &Service{}

	got := service.finalizePreviewTitle(context.Background(), "【蒜香排骨】")
	if got.Title != "蒜香排骨" {
		t.Fatalf("Title = %q, want %q", got.Title, "蒜香排骨")
	}
	if got.Source != "rule" {
		t.Fatalf("Source = %q, want %q", got.Source, "rule")
	}
}

func TestSummarizeBilibiliDraftUsesRouter(t *testing.T) {
	t.Parallel()

	router := &fakeAIRouter{
		available: map[airouter.Scene]bool{airouter.SceneSummary: true},
		content:   `{"title":"番茄牛腩","ingredient":"牛腩、番茄","summary":"炖至软烂。","mainIngredients":["牛腩 500克","番茄 3个"],"secondaryIngredients":["盐 3克"],"steps":[{"title":"焯水","detail":"牛腩冷水下锅焯水。"},{"title":"炒香","detail":"番茄炒出汁。"},{"title":"炖煮","detail":"加入牛腩炖至软烂。"}],"note":"回看原视频确认火候。"}`,
	}
	service := &Service{aiRouter: router}
	draft, routeResult, err := service.summarizeBilibiliDraft(context.Background(), BilibiliParseResult{
		Title:        "番茄牛腩教程",
		SubtitleText: "牛腩焯水，番茄炒出汁后炖煮。",
	})
	if err != nil {
		t.Fatalf("summarizeBilibiliDraft() error = %v", err)
	}
	if draft.Title != "番茄牛腩" || len(draft.ParsedContent.Steps) != 3 {
		t.Fatalf("draft = %#v", draft)
	}
	if routeResult.ProviderID != "fake-provider" || router.scene != airouter.SceneSummary || router.input.ContentKind != "summary_bilibili" {
		t.Fatalf("unexpected route result/input: result=%#v scene=%q input=%#v", routeResult, router.scene, router.input)
	}
}

func TestSummaryAndTitleRequireRouter(t *testing.T) {
	t.Parallel()

	service := &Service{}
	_, _, summaryErr := service.summarizeBilibiliDraft(context.Background(), BilibiliParseResult{})
	_, _, titleErr := service.refineTitleWithAI(context.Background(), "番茄牛腩")
	for name, err := range map[string]error{"summary": summaryErr, "title": titleErr} {
		var appErr *common.AppError
		if !errors.As(err, &appErr) || appErr.HTTPStatus != 503 {
			t.Fatalf("%s error = %v, want 503 app error", name, err)
		}
	}
}

func TestFinalizePreviewTitleMarksAISourceWhenRefinedTitleWins(t *testing.T) {
	t.Parallel()

	router := &fakeAIRouter{
		available: map[airouter.Scene]bool{airouter.SceneTitle: true},
		content:   `{"title":"蒜香排骨"}`,
	}
	service := &Service{aiRouter: router}

	got := service.finalizePreviewTitle(context.Background(), "【蒜香排骨最好吃的做法~-哔哩哔哩】")
	if got.Title != "蒜香排骨" {
		t.Fatalf("Title = %q, want %q", got.Title, "蒜香排骨")
	}
	if got.Source != "ai" {
		t.Fatalf("Source = %q, want %q", got.Source, "ai")
	}
	if router.scene != airouter.SceneTitle || router.input.ContentKind != "title_refine" {
		t.Fatalf("unexpected route input: scene=%q input=%#v", router.scene, router.input)
	}
}

func TestFinalizePreviewTitleFallsBackToRuleWhenAIRequestFails(t *testing.T) {
	t.Parallel()

	service := &Service{aiRouter: &fakeAIRouter{
		available: map[airouter.Scene]bool{airouter.SceneTitle: true},
		err:       errors.New("boom"),
	}}

	got := service.finalizePreviewTitle(context.Background(), "【蒜香排骨】")
	if got.Title != "蒜香排骨" {
		t.Fatalf("Title = %q, want %q", got.Title, "蒜香排骨")
	}
	if got.Source != "rule" {
		t.Fatalf("Source = %q, want %q", got.Source, "rule")
	}
}

func TestFinalizePreviewTitleFallsBackToRuleWhenAIScoreLower(t *testing.T) {
	t.Parallel()

	service := &Service{aiRouter: &fakeAIRouter{
		available: map[airouter.Scene]bool{airouter.SceneTitle: true},
		content:   `{"title":"厨房日记"}`,
	}}

	got := service.finalizePreviewTitle(context.Background(), "【蒜香排骨】做法分享")
	if got.Title != "蒜香排骨" {
		t.Fatalf("Title = %q, want %q", got.Title, "蒜香排骨")
	}
	if got.Source != "rule" {
		t.Fatalf("Source = %q, want %q", got.Source, "rule")
	}
}

func TestFinalizePreviewTitleFallsBackToRuleWhenAIReturnsEmpty(t *testing.T) {
	t.Parallel()

	service := &Service{aiRouter: &fakeAIRouter{
		available: map[airouter.Scene]bool{airouter.SceneTitle: true},
		content:   `{"title":""}`,
	}}

	got := service.finalizePreviewTitle(context.Background(), "【红烧排骨】")
	if got.Title != "红烧排骨" {
		t.Fatalf("Title = %q, want %q", got.Title, "红烧排骨")
	}
	if got.Source != "rule" {
		t.Fatalf("Source = %q, want %q", got.Source, "rule")
	}
}
