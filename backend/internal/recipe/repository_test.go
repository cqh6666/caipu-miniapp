package recipe

import (
	"context"
	"database/sql"
	"errors"
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

	claimToken, marked, err := NewRepository(db).MarkAutoParseProcessing(
		context.Background(),
		"attempt-track",
		"xiaohongshu",
		"2026-05-01T00:02:00Z",
		"2026-05-01T00:12:00Z",
	)
	if err != nil {
		t.Fatalf("MarkAutoParseProcessing() error = %v", err)
	}
	if !marked {
		t.Fatal("MarkAutoParseProcessing() marked = false, want true")
	}
	if claimToken == "" {
		t.Fatal("MarkAutoParseProcessing() claim token = empty")
	}

	assertAutoParseState(t, db, "attempt-track", ParseStatusProcessing, "xiaohongshu", "", "", "", "", "", "2026-05-01T00:02:00Z", 2)
}

func TestRepositoryMarkAutoParseRetryPendingStoresNextAttempt(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, parse_status, parse_attempts, parse_finished_at,
  created_by, updated_by, created_at, updated_at
) VALUES (
  'retry-pending', '小炒牛肉', 'pending', 0, '2026-05-01T00:01:00Z',
  1, 1, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z'
);
`); err != nil {
		t.Fatalf("seed recipe error = %v", err)
	}

	repo := NewRepository(db)
	claimToken, marked, err := repo.MarkAutoParseProcessing(
		context.Background(),
		"retry-pending",
		"bilibili",
		"2026-05-01T00:02:00Z",
		"2026-05-01T00:12:00Z",
	)
	if err != nil || !marked {
		t.Fatalf("MarkAutoParseProcessing() marked=%t error=%v", marked, err)
	}
	if err := repo.MarkAutoParseRetryPending(
		context.Background(),
		"retry-pending",
		claimToken,
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
  ('placeholder-title', 1, '猜测标题', 'placeholder', 'pending', 0, '2999-01-01T00:00:00Z', 'timeout', '', 7, 7, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z'),
  ('manual-title', 1, '用户标题', 'manual', 'pending', 0, '2999-01-01T00:00:00Z', 'timeout', '', 7, 7, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z');
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
		claimToken, marked, err := repo.MarkAutoParseProcessing(
			context.Background(),
			recipeID,
			"xiaohongshu",
			"2026-05-01T00:01:00Z",
			"2026-05-01T00:11:00Z",
		)
		if err != nil || !marked {
			t.Fatalf("MarkAutoParseProcessing(%s) marked=%t error=%v", recipeID, marked, err)
		}
		if err := repo.ApplyAutoParseResult(
			context.Background(),
			recipeID,
			claimToken,
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

func TestRepositoryAutoParseClaimRejectsReclaimedWorkerResult(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()
	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '并发测试厨房', 7, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z', 'custom');
INSERT INTO recipes (id, kitchen_id, title, parse_status, created_by, updated_by, created_at, updated_at)
VALUES ('reclaimed-parse', 1, '原始标题', 'pending', 7, 7, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z');
`); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	claimA, marked, err := repo.MarkAutoParseProcessing(context.Background(), "reclaimed-parse", "bilibili", "2026-05-01T00:01:00Z", "2026-05-01T00:02:00Z")
	if err != nil || !marked {
		t.Fatalf("claim A marked=%t error=%v", marked, err)
	}
	if _, err := repo.RequeueStaleAutoParse(context.Background(), "2026-05-01T00:00:00Z", "2026-05-01T00:03:00Z"); err != nil {
		t.Fatal(err)
	}
	claimB, marked, err := repo.MarkAutoParseProcessing(context.Background(), "reclaimed-parse", "bilibili", "2026-05-01T00:04:00Z", "2026-05-01T00:14:00Z")
	if err != nil || !marked {
		t.Fatalf("claim B marked=%t error=%v", marked, err)
	}
	if err := repo.MarkAutoParseFailed(context.Background(), "reclaimed-parse", claimA, "bilibili", "old worker", "upstream", "2026-05-01T00:05:00Z"); !errors.Is(err, ErrStaleJobResult) {
		t.Fatalf("claim A failure update error=%v, want stale result", err)
	}
	if err := repo.ApplyAutoParseResult(context.Background(), "reclaimed-parse", claimB, "bilibili:ai", "", "2026-05-01T00:06:00Z", Recipe{
		Title:         "新执行者标题",
		ParsedContent: ParsedContent{Steps: []ParsedStep{{Title: "步骤一", Detail: "完成"}}},
	}); err != nil {
		t.Fatalf("claim B apply error=%v", err)
	}
	assertRecipeTitleSource(t, db, "reclaimed-parse", "原始标题", TitleSourceManual)
	assertAutoParseState(t, db, "reclaimed-parse", ParseStatusDone, "bilibili:ai", "", "2026-05-01T00:03:00Z", "2026-05-01T00:06:00Z", "", "", "", 2)
}

func TestRepositoryAutoParsePreservesHumanUpdateDuringClaim(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()
	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '人工编辑厨房', 7, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z', 'custom');
INSERT INTO recipes (
  id, kitchen_id, title, title_source, meal_type, status, parse_status,
  created_by, updated_by, created_at, updated_at
) VALUES (
  'human-edit-parse', 1, '占位标题', 'placeholder', 'main', 'wishlist', 'pending',
  7, 7, '2026-05-01T00:00:00Z', '2026-05-01T00:00:00Z'
);
`); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	claim, marked, err := repo.MarkAutoParseProcessing(context.Background(), "human-edit-parse", "bilibili", "2026-05-01T00:01:00Z", "2026-05-01T00:11:00Z")
	if err != nil || !marked {
		t.Fatalf("claim marked=%t error=%v", marked, err)
	}
	item, err := repo.FindByID(context.Background(), "human-edit-parse")
	if err != nil {
		t.Fatal(err)
	}
	item.Title = "用户刚保存的标题"
	item.TitleSource = TitleSourceManual
	item.UpdatedBy = 7
	item.UpdatedAt = "2026-05-01T00:02:00Z"
	if _, err := repo.Update(context.Background(), item); err != nil {
		t.Fatalf("human Update error=%v", err)
	}
	err = repo.ApplyAutoParseResult(context.Background(), "human-edit-parse", claim, "bilibili:ai", "", "2026-05-01T00:03:00Z", Recipe{
		Title:         "模型覆盖标题",
		ParsedContent: ParsedContent{Steps: []ParsedStep{{Title: "模型步骤", Detail: "不应保存"}}},
	})
	if !errors.Is(err, ErrAutoParseContentChanged) {
		t.Fatalf("ApplyAutoParseResult error=%v, want content conflict", err)
	}
	assertRecipeTitleSource(t, db, "human-edit-parse", "用户刚保存的标题", TitleSourceManual)
	assertAutoParseState(t, db, "human-edit-parse", ParseStatusFailed, "bilibili", "菜谱已被人工修改，自动解析结果未应用", "", "2026-05-01T00:03:00Z", "", "content_conflict", "", 1)
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
