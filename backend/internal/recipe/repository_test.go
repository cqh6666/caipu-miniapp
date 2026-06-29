package recipe

import (
	"context"
	"database/sql"
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

func TestRepositoryListPendingAutoParseSkipsFutureRetry(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, link, parse_status, parse_requested_at, parse_next_attempt_at, created_by, updated_by, created_at, updated_at
) VALUES
  ('ready-now', '立即解析', 'https://www.xiaohongshu.com/explore/ready', 'pending', '2026-05-01T00:00:00Z', '', 1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z'),
  ('retry-ready', '重试到点', 'https://www.bilibili.com/video/BVready', 'pending', '2026-05-01T00:01:00Z', '2000-01-01T00:00:00Z', 1, 1, '2026-05-01T00:01:00Z', '2026-05-01T00:01:00Z'),
  ('retry-future', '稍后重试', 'https://www.bilibili.com/video/BVfuture', 'pending', '2026-05-01T00:02:00Z', '2999-01-01T00:00:00Z', 1, 1, '2026-05-01T00:02:00Z', '2026-05-01T00:02:00Z'),
  ('processing', '处理中', 'https://www.bilibili.com/video/BVprocessing', 'processing', '2026-05-01T00:03:00Z', '', 1, 1, '2026-05-01T00:03:00Z', '2026-05-01T00:03:00Z');
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	items, err := NewRepository(db).ListPendingAutoParse(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListPendingAutoParse() error = %v", err)
	}
	if got, want := len(items), 2; got != want {
		t.Fatalf("len(items) = %d, want %d: %#v", got, want, items)
	}
	if got, want := items[0].ID, "ready-now"; got != want {
		t.Fatalf("items[0].ID = %q, want %q", got, want)
	}
	if got, want := items[1].ID, "retry-ready"; got != want {
		t.Fatalf("items[1].ID = %q, want %q", got, want)
	}
}

func TestRepositoryMarkAutoParseProcessingTracksAttempt(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, parse_status, parse_attempts, parse_next_attempt_at, parse_last_error_type, parse_finished_at,
  created_by, updated_by, created_at, updated_at
) VALUES (
  'attempt-track', '番茄炒蛋', 'pending', 1, '2999-01-01T00:00:00Z', 'timeout', '2026-05-01T00:00:00Z',
  1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z'
);
`); err != nil {
		t.Fatalf("seed recipe error = %v", err)
	}

	marked, err := NewRepository(db).MarkAutoParseProcessing(
		context.Background(),
		"attempt-track",
		"xiaohongshu",
		"2026-05-01T00:02:00Z",
	)
	if err != nil {
		t.Fatalf("MarkAutoParseProcessing() error = %v", err)
	}
	if !marked {
		t.Fatal("MarkAutoParseProcessing() marked = false, want true")
	}

	assertAutoParseState(t, db, "attempt-track", ParseStatusProcessing, "xiaohongshu", "", "", "", "", "", "2026-05-01T00:02:00Z", 2)
}

func TestRepositoryMarkAutoParseRetryPendingStoresNextAttempt(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, parse_status, parse_attempts, parse_processing_started_at, parse_finished_at,
  created_by, updated_by, created_at, updated_at
) VALUES (
  'retry-pending', '小炒牛肉', 'processing', 1, '2026-05-01T00:00:00Z', '2026-05-01T00:01:00Z',
  1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z'
);
`); err != nil {
		t.Fatalf("seed recipe error = %v", err)
	}

	if err := NewRepository(db).MarkAutoParseRetryPending(
		context.Background(),
		"retry-pending",
		"bilibili",
		"upstream timed out",
		"timeout",
		"2026-05-01T00:06:00Z",
	); err != nil {
		t.Fatalf("MarkAutoParseRetryPending() error = %v", err)
	}

	assertAutoParseState(t, db, "retry-pending", ParseStatusPending, "bilibili", "upstream timed out", "", "", "2026-05-01T00:06:00Z", "timeout", "", 1)
}

func TestRepositoryRequeueStaleAutoParseOnlyTouchesExpiredProcessingJobs(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, parse_status, parse_error, parse_requested_at, parse_finished_at, parse_attempts,
  parse_processing_started_at, created_by, updated_by, created_at, updated_at, deleted_at
) VALUES
  ('stale-processing', '卡住任务', 'processing', 'upstream timeout', '2026-05-01T00:00:00Z', '2026-05-01T00:01:00Z', 2, '2026-05-01T00:00:00Z', 1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:01:00Z', NULL),
  ('fresh-processing', '新任务', 'processing', '', '2026-05-01T00:27:00Z', '', 1, '2026-05-01T00:27:00Z', 1, 1, '2026-05-01T00:26:00Z', '2026-05-01T00:27:00Z', NULL),
  ('old-pending', '已排队', 'pending', '', '2026-05-01T00:00:00Z', '', 1, '', 1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z', NULL),
  ('deleted-processing', '已删除', 'processing', '', '2026-05-01T00:00:00Z', '', 1, '2026-05-01T00:00:00Z', 1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z', '2026-05-01T00:05:00Z');
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	requeued, err := NewRepository(db).RequeueStaleAutoParse(
		context.Background(),
		"2026-05-01T00:20:00Z",
		"2026-05-01T00:30:00Z",
	)
	if err != nil {
		t.Fatalf("RequeueStaleAutoParse() error = %v", err)
	}
	if got, want := requeued, int64(1); got != want {
		t.Fatalf("RequeueStaleAutoParse() requeued %d rows, want %d", got, want)
	}

	assertAutoParseState(t, db, "stale-processing", ParseStatusPending, "", "", "2026-05-01T00:30:00Z", "", "", "", "", 2)
	assertAutoParseState(t, db, "fresh-processing", ParseStatusProcessing, "", "", "2026-05-01T00:27:00Z", "", "", "", "2026-05-01T00:27:00Z", 1)
	assertAutoParseState(t, db, "old-pending", ParseStatusPending, "", "", "2026-05-01T00:00:00Z", "", "", "", "", 1)
}

func TestRepositoryApplyAutoParseResultOnlyOverridesPlaceholderTitle(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '联调试吃厨房', 7, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z', 'custom');

INSERT INTO recipes (
  id, kitchen_id, title, title_source, parse_status, parse_attempts, parse_next_attempt_at, parse_last_error_type,
  parse_processing_started_at, created_by, updated_by, created_at, updated_at
) VALUES
  ('placeholder-title', 1, '猜测标题', 'placeholder', 'processing', 1, '2999-01-01T00:00:00Z', 'timeout', '2026-05-01T00:00:00Z', 7, 7, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z'),
  ('manual-title', 1, '用户标题', 'manual', 'processing', 1, '2999-01-01T00:00:00Z', 'timeout', '2026-05-01T00:00:00Z', 7, 7, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z');
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	repo := NewRepository(db)
	draft := Recipe{
		Title:      "模型标题",
		Ingredient: "鸡蛋",
		Summary:    "鲜香快手",
		ParsedContent: ParsedContent{
			MainIngredients: []string{"鸡蛋 2个"},
			Steps: []ParsedStep{
				{Title: "打蛋", Detail: "鸡蛋打散。"},
				{Title: "炒制", Detail: "热锅炒熟。"},
			},
		},
	}
	for _, recipeID := range []string{"placeholder-title", "manual-title"} {
		if err := repo.ApplyAutoParseResult(
			context.Background(),
			recipeID,
			"xiaohongshu:ai",
			"",
			"2026-05-01T00:05:00Z",
			draft,
		); err != nil {
			t.Fatalf("ApplyAutoParseResult(%s) error = %v", recipeID, err)
		}
	}

	assertRecipeTitleSource(t, db, "placeholder-title", "模型标题", TitleSourceParsed)
	assertRecipeTitleSource(t, db, "manual-title", "用户标题", TitleSourceManual)
	assertAutoParseState(t, db, "placeholder-title", ParseStatusDone, "xiaohongshu:ai", "", "", "2026-05-01T00:05:00Z", "", "", "", 1)
	assertAutoParseState(t, db, "manual-title", ParseStatusDone, "xiaohongshu:ai", "", "", "2026-05-01T00:05:00Z", "", "", "", 1)
}

func TestResolveUpdateTitleSourceKeepsEditedTitleManual(t *testing.T) {
	t.Parallel()

	got := resolveUpdateTitleSource(
		Recipe{
			Title:       "猜测标题",
			TitleSource: TitleSourcePlaceholder,
		},
		Recipe{
			Title: "用户改过的标题",
		},
		updateRecipeRequest{
			TitleSource: TitleSourcePlaceholder,
		},
	)
	if got != TitleSourceManual {
		t.Fatalf("resolveUpdateTitleSource() = %q, want %q", got, TitleSourceManual)
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

func assertAutoParseState(t *testing.T, db *sql.DB, recipeID, wantStatus, wantSource, wantError, wantRequestedAt, wantFinishedAt, wantNextAttemptAt, wantLastErrorType, wantProcessingStartedAt string, wantAttempts int) {
	t.Helper()

	var gotStatus string
	var gotSource string
	var gotError string
	var gotRequestedAt string
	var gotFinishedAt string
	var gotAttempts int
	var gotNextAttemptAt string
	var gotLastErrorType string
	var gotProcessingStartedAt string
	if err := db.QueryRow(`
SELECT parse_status, parse_source, parse_error, COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''),
       parse_attempts, parse_next_attempt_at, parse_last_error_type, parse_processing_started_at
FROM recipes
WHERE id = ?
`, recipeID).Scan(
		&gotStatus,
		&gotSource,
		&gotError,
		&gotRequestedAt,
		&gotFinishedAt,
		&gotAttempts,
		&gotNextAttemptAt,
		&gotLastErrorType,
		&gotProcessingStartedAt,
	); err != nil {
		t.Fatalf("query auto parse state for %s error = %v", recipeID, err)
	}

	if gotStatus != wantStatus {
		t.Fatalf("recipe %s parse_status = %q, want %q", recipeID, gotStatus, wantStatus)
	}
	if gotSource != wantSource {
		t.Fatalf("recipe %s parse_source = %q, want %q", recipeID, gotSource, wantSource)
	}
	if gotError != wantError {
		t.Fatalf("recipe %s parse_error = %q, want %q", recipeID, gotError, wantError)
	}
	if gotRequestedAt != wantRequestedAt {
		t.Fatalf("recipe %s parse_requested_at = %q, want %q", recipeID, gotRequestedAt, wantRequestedAt)
	}
	if gotFinishedAt != wantFinishedAt {
		t.Fatalf("recipe %s parse_finished_at = %q, want %q", recipeID, gotFinishedAt, wantFinishedAt)
	}
	if gotAttempts != wantAttempts {
		t.Fatalf("recipe %s parse_attempts = %d, want %d", recipeID, gotAttempts, wantAttempts)
	}
	if gotNextAttemptAt != wantNextAttemptAt {
		t.Fatalf("recipe %s parse_next_attempt_at = %q, want %q", recipeID, gotNextAttemptAt, wantNextAttemptAt)
	}
	if gotLastErrorType != wantLastErrorType {
		t.Fatalf("recipe %s parse_last_error_type = %q, want %q", recipeID, gotLastErrorType, wantLastErrorType)
	}
	if gotProcessingStartedAt != wantProcessingStartedAt {
		t.Fatalf("recipe %s parse_processing_started_at = %q, want %q", recipeID, gotProcessingStartedAt, wantProcessingStartedAt)
	}
}

func assertRecipeTitleSource(t *testing.T, db *sql.DB, recipeID, wantTitle, wantTitleSource string) {
	t.Helper()

	var gotTitle string
	var gotTitleSource string
	if err := db.QueryRow(`
SELECT title, title_source
FROM recipes
WHERE id = ?
`, recipeID).Scan(&gotTitle, &gotTitleSource); err != nil {
		t.Fatalf("query title source for %s error = %v", recipeID, err)
	}
	if gotTitle != wantTitle {
		t.Fatalf("recipe %s title = %q, want %q", recipeID, gotTitle, wantTitle)
	}
	if gotTitleSource != wantTitleSource {
		t.Fatalf("recipe %s title_source = %q, want %q", recipeID, gotTitleSource, wantTitleSource)
	}
}
