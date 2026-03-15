package recipe

import "testing"

func TestHasUserProvidedParsedContentTreatsFallbackAsEmpty(t *testing.T) {
	t.Parallel()

	fallback := normalizeParsedContent(ParsedContent{}, "main", "番茄牛腩", "牛腩")
	if hasUserProvidedParsedContent(fallback, "main", "番茄牛腩", "牛腩") {
		t.Fatal("fallback parsed content should not be treated as user-provided content")
	}
}

func TestHasUserProvidedParsedContentTreatsLegacyFrontendFallbackAsEmpty(t *testing.T) {
	t.Parallel()

	fallback := legacyFrontendFallbackParsedContent("main", "番茄牛腩", "牛腩")
	if hasUserProvidedParsedContent(fallback, "main", "番茄牛腩", "牛腩") {
		t.Fatal("legacy frontend fallback parsed content should not be treated as user-provided content")
	}
}

func TestHasUserProvidedParsedContentRecognizesManualContent(t *testing.T) {
	t.Parallel()

	content := ParsedContent{
		Ingredients: []string{"牛腩 500克", "番茄 3个"},
		Steps:       []string{"牛腩焯水备用", "番茄炒软后和牛腩一起炖煮"},
	}

	if !hasUserProvidedParsedContent(content, "main", "番茄牛腩", "牛腩") {
		t.Fatal("manual parsed content should be treated as user-provided content")
	}
}

func TestApplyCreateParseStateQueuesSupportedBilibiliLink(t *testing.T) {
	t.Parallel()

	item := Recipe{}
	req := createRecipeRequest{
		Title:      "番茄牛腩",
		Ingredient: "",
		Link:       "https://www.bilibili.com/video/BV1aWCEYHErc",
		MealType:   "main",
		ParsedContent: normalizeParsedContent(
			ParsedContent{},
			"main",
			"番茄牛腩",
			"",
		),
	}

	applyCreateParseState(&item, req, "2026-03-14T00:00:00Z")

	if item.ParseStatus != ParseStatusPending {
		t.Fatalf("ParseStatus = %q, want %q", item.ParseStatus, ParseStatusPending)
	}
}

func TestApplyCreateParseStateQueuesLegacyFrontendFallbackBilibiliLink(t *testing.T) {
	t.Parallel()

	item := Recipe{}
	req := createRecipeRequest{
		Title:         "番茄牛腩",
		Ingredient:    "",
		Link:          "https://www.bilibili.com/video/BV1aWCEYHErc",
		MealType:      "main",
		ParsedContent: legacyFrontendFallbackParsedContent("main", "番茄牛腩", ""),
	}

	applyCreateParseState(&item, req, "2026-03-14T00:00:00Z")

	if item.ParseStatus != ParseStatusPending {
		t.Fatalf("ParseStatus = %q, want %q", item.ParseStatus, ParseStatusPending)
	}
}

func TestApplyCreateParseStateQueuesSupportedXiaohongshuLink(t *testing.T) {
	t.Parallel()

	item := Recipe{}
	req := createRecipeRequest{
		Title:      "番茄牛腩",
		Ingredient: "",
		Link:       "https://www.xiaohongshu.com/explore/68abcd1234",
		MealType:   "main",
		ParsedContent: normalizeParsedContent(
			ParsedContent{},
			"main",
			"番茄牛腩",
			"",
		),
	}

	applyCreateParseState(&item, req, "2026-03-15T00:00:00Z")

	if item.ParseStatus != ParseStatusPending {
		t.Fatalf("ParseStatus = %q, want %q", item.ParseStatus, ParseStatusPending)
	}
	if item.ParseSource != "xiaohongshu" {
		t.Fatalf("ParseSource = %q, want %q", item.ParseSource, "xiaohongshu")
	}
}
