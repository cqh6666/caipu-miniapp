package recipe

import (
	"context"
	"testing"
)

func TestRepositoryListByKitchenIDFiltersTitleOrIngredientKeyword(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (id, kitchen_id, title, ingredient, summary, link, meal_type, status, note, ingredients_json, steps_json, created_by, updated_by, created_at, updated_at)
VALUES
  ('rec_title', 1, '鸡胸肉沙拉', '生菜', '', '', 'main', 'wishlist', '', '{}', '[]', 1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z'),
  ('rec_ingredient', 1, '清爽沙拉', '鸡胸肉、生菜', '', '', 'main', 'wishlist', '', '{}', '[]', 1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:01Z'),
  ('rec_structured', 1, '快手拌面', '', '', '', 'main', 'wishlist', '', '{"mainIngredients":["鸡胸肉","面条"]}', '[]', 1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:02Z'),
  ('rec_note_only', 1, '绿叶菜', '生菜', '', '', 'main', 'wishlist', '鸡胸肉也可以配', '{}', '[]', 1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:03Z');
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	items, err := NewRepository(db).ListByKitchenID(context.Background(), 1, ListFilter{
		TitleOrIngredientKeyword: "鸡胸肉",
	})
	if err != nil {
		t.Fatalf("ListByKitchenID() error = %v", err)
	}
	if got, want := len(items), 3; got != want {
		t.Fatalf("len(items) = %d, want %d: %#v", got, want, items)
	}
	ids := map[string]bool{}
	for _, item := range items {
		ids[item.ID] = true
	}
	for _, id := range []string{"rec_title", "rec_ingredient", "rec_structured"} {
		if !ids[id] {
			t.Fatalf("missing %s in %#v", id, ids)
		}
	}
	if ids["rec_note_only"] {
		t.Fatalf("note-only recipe should not match title/ingredient filter: %#v", ids)
	}
}

func TestResolveAutoParseImagesKeepsManualImagesAndAppendsParsedImages(t *testing.T) {
	t.Parallel()

	imageURL, imageURLs, imageMetas := resolveAutoParseImages(
		Recipe{
			Link: "https://www.xiaohongshu.com/explore/demo",
			ImageMetas: []RecipeImageMeta{
				{
					URL:         "https://cdn.example.com/manual-cover.jpg",
					SourceType:  RecipeImageSourceUser,
					OriginURL:   "https://cdn.example.com/manual-cover.jpg",
					ContentHash: "manual-cover",
				},
				{
					URL:         "https://cdn.example.com/manual-step.jpg",
					SourceType:  RecipeImageSourceUser,
					OriginURL:   "https://cdn.example.com/manual-step.jpg",
					ContentHash: "manual-step",
				},
			},
		},
		Recipe{
			ImageURLs: []string{
				"https://cdn.example.com/parsed-cover.jpg",
				"https://cdn.example.com/parsed-step.jpg",
			},
		},
	)

	if got, want := imageURL, "https://cdn.example.com/manual-cover.jpg"; got != want {
		t.Fatalf("imageURL = %q, want %q", got, want)
	}
	if got, want := len(imageURLs), 4; got != want {
		t.Fatalf("len(imageURLs) = %d, want %d", got, want)
	}
	if got, want := imageURLs[2], "https://cdn.example.com/parsed-cover.jpg"; got != want {
		t.Fatalf("imageURLs[2] = %q, want %q", got, want)
	}
	if got, want := imageMetas[2].SourceType, RecipeImageSourceParsed; got != want {
		t.Fatalf("imageMetas[2].SourceType = %q, want %q", got, want)
	}
	if got, want := imageMetas[2].SourceLink, "https://www.xiaohongshu.com/explore/demo"; got != want {
		t.Fatalf("imageMetas[2].SourceLink = %q, want %q", got, want)
	}
}

func TestResolveAutoParseImagesReplacesPreviousParsedImages(t *testing.T) {
	t.Parallel()

	_, imageURLs, imageMetas := resolveAutoParseImages(
		Recipe{
			Link: "https://www.bilibili.com/video/BV1demo",
			ImageMetas: []RecipeImageMeta{
				{
					URL:         "/uploads/2026/04/manual.jpg",
					SourceType:  RecipeImageSourceUser,
					OriginURL:   "/uploads/2026/04/manual.jpg",
					ContentHash: "manual",
				},
				{
					URL:         "/uploads/2026/04/old-parsed.jpg",
					SourceType:  RecipeImageSourceParsed,
					OriginURL:   "https://cdn.example.com/old-parsed.jpg",
					SourceLink:  "https://www.bilibili.com/video/BV1demo",
					ContentHash: "old-parsed",
				},
			},
		},
		Recipe{
			ImageURLs: []string{"https://cdn.example.com/new-parsed.jpg"},
		},
	)

	if got, want := len(imageURLs), 2; got != want {
		t.Fatalf("len(imageURLs) = %d, want %d", got, want)
	}
	if got, want := imageURLs[1], "https://cdn.example.com/new-parsed.jpg"; got != want {
		t.Fatalf("imageURLs[1] = %q, want %q", got, want)
	}
	if got, want := imageMetas[1].SourceType, RecipeImageSourceParsed; got != want {
		t.Fatalf("imageMetas[1].SourceType = %q, want %q", got, want)
	}
}

func TestDedupeRecipeImageMetasPrefersUserImageWhenHashesMatch(t *testing.T) {
	t.Parallel()

	imageMetas := dedupeRecipeImageMetas([]RecipeImageMeta{
		{
			URL:         "/uploads/2026/04/parsed.jpg",
			SourceType:  RecipeImageSourceParsed,
			ContentHash: "same-hash",
			OriginURL:   "https://cdn.example.com/parsed.jpg",
		},
		{
			URL:         "/uploads/2026/04/manual.jpg",
			SourceType:  RecipeImageSourceUser,
			ContentHash: "same-hash",
			OriginURL:   "/uploads/2026/04/manual.jpg",
		},
	})

	if got, want := len(imageMetas), 1; got != want {
		t.Fatalf("len(imageMetas) = %d, want %d", got, want)
	}
	if got, want := imageMetas[0].URL, "/uploads/2026/04/manual.jpg"; got != want {
		t.Fatalf("imageMetas[0].URL = %q, want %q", got, want)
	}
	if got, want := imageMetas[0].SourceType, RecipeImageSourceUser; got != want {
		t.Fatalf("imageMetas[0].SourceType = %q, want %q", got, want)
	}
}

func TestNonNullableTrimmedStringPreservesEmptyString(t *testing.T) {
	t.Parallel()

	if got := nonNullableTrimmedString("   "); got != "" {
		t.Fatalf("nonNullableTrimmedString returned %q, want empty string", got)
	}
}

func TestNonNullableTrimmedStringTrimsWhitespace(t *testing.T) {
	t.Parallel()

	if got, want := nonNullableTrimmedString("  酸甜浓汤  "), "酸甜浓汤"; got != want {
		t.Fatalf("nonNullableTrimmedString = %q, want %q", got, want)
	}
}
