package recipe

import "testing"

func TestNormalizeRecipeInputSupportsMultipleImages(t *testing.T) {
	t.Parallel()

	item, err := normalizeRecipeInput(
		"番茄牛腩",
		"牛腩",
		"酸甜浓汤，适合一锅炖",
		"",
		"",
		[]string{"https://img.example.com/1.jpg", "https://img.example.com/2.jpg"},
		"main",
		"wishlist",
		"",
		ParsedContent{},
	)
	if err != nil {
		t.Fatalf("normalizeRecipeInput returned error: %v", err)
	}

	if got, want := len(item.ImageURLs), 2; got != want {
		t.Fatalf("len(ImageURLs) = %d, want %d", got, want)
	}
	if got, want := item.ImageURL, "https://img.example.com/1.jpg"; got != want {
		t.Fatalf("ImageURL = %q, want %q", got, want)
	}
}

func TestNormalizeRecipeInputRejectsTooManyImages(t *testing.T) {
	t.Parallel()

	imageURLs := make([]string, 0, maxRecipeImages+1)
	for index := 0; index < maxRecipeImages+1; index += 1 {
		imageURLs = append(imageURLs, "https://img.example.com/test-"+string(rune('a'+index))+".jpg")
	}

	if _, err := normalizeRecipeInput("番茄牛腩", "", "", "", "", imageURLs, "main", "wishlist", "", ParsedContent{}); err == nil {
		t.Fatal("normalizeRecipeInput should reject too many images")
	}
}

func TestNormalizeRecipeInputRejectsSummaryLongerThanTwentyFourRunes(t *testing.T) {
	t.Parallel()

	if _, err := normalizeRecipeInput(
		"番茄牛腩",
		"牛腩",
		"先焯水去腥，再小火慢炖至软烂，周末一锅更省事也更适合反复做",
		"",
		"",
		nil,
		"main",
		"wishlist",
		"",
		ParsedContent{},
	); err == nil {
		t.Fatal("normalizeRecipeInput should reject long summary")
	}
}

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
		MainIngredients: []string{"牛腩 500克", "番茄 3个"},
		Steps: []ParsedStep{
			{Title: "焯水去腥", Detail: "牛腩焯水备用"},
			{Title: "小火慢炖", Detail: "番茄炒软后和牛腩一起炖煮"},
		},
	}

	if !hasUserProvidedParsedContent(content, "main", "番茄牛腩", "牛腩") {
		t.Fatal("manual parsed content should be treated as user-provided content")
	}
}

func TestNormalizeParsedContentKeepsMultiplePrimaryIngredients(t *testing.T) {
	t.Parallel()

	content := normalizeParsedContent(ParsedContent{
		legacyIngredients: []string{
			"牛腩 500克",
			"番茄 3个",
			"土豆 2个",
			"胡萝卜 1根",
			"洋葱 半个",
			"盐 3克",
			"生抽 1勺",
		},
	}, "main", "番茄牛腩", "牛腩")

	if got, want := len(content.MainIngredients), 5; got != want {
		t.Fatalf("len(MainIngredients) = %d, want %d (%#v)", got, want, content.MainIngredients)
	}
	if got, want := content.MainIngredients[4], "洋葱 半个"; got != want {
		t.Fatalf("MainIngredients[4] = %q, want %q", got, want)
	}
	if got, want := len(content.SecondaryIngredients), 2; got != want {
		t.Fatalf("len(SecondaryIngredients) = %d, want %d (%#v)", got, want, content.SecondaryIngredients)
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
