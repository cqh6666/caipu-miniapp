package recipe

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRepositoryRequeueStaleFlowchartsOnlyTouchesExpiredProcessingJobs(t *testing.T) {
	db := openFlowchartTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, flowchart_status, flowchart_error, flowchart_requested_at, flowchart_finished_at, created_at, updated_at, deleted_at
) VALUES
  ('stale-processing', 'processing', 'upstream timeout', '2026-03-25T00:00:00+08:00', '2026-03-25T00:01:00+08:00', '2026-03-24T23:55:00+08:00', '2026-03-25T00:01:00+08:00', NULL),
  ('fresh-processing', 'processing', '', '2026-03-25T00:27:00+08:00', NULL, '2026-03-25T00:26:00+08:00', '2026-03-25T00:27:00+08:00', NULL),
  ('old-pending', 'pending', '', '2026-03-25T00:00:00+08:00', NULL, '2026-03-24T23:55:00+08:00', '2026-03-25T00:00:00+08:00', NULL),
  ('deleted-processing', 'processing', '', '2026-03-25T00:00:00+08:00', NULL, '2026-03-24T23:55:00+08:00', '2026-03-25T00:00:00+08:00', '2026-03-25T00:05:00+08:00');
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	repo := NewRepository(db)
	requeued, err := repo.RequeueStaleFlowcharts(
		context.Background(),
		"2026-03-25T00:20:00+08:00",
		"2026-03-25T00:30:00+08:00",
	)
	if err != nil {
		t.Fatalf("RequeueStaleFlowcharts() error = %v", err)
	}
	if got, want := requeued, int64(1); got != want {
		t.Fatalf("RequeueStaleFlowcharts() requeued %d rows, want %d", got, want)
	}

	assertFlowchartState(t, db, "stale-processing", FlowchartStatusPending, "", "2026-03-25T00:30:00+08:00", "", "2026-03-25T00:01:00+08:00")
	assertFlowchartState(t, db, "fresh-processing", FlowchartStatusProcessing, "", "2026-03-25T00:27:00+08:00", "", "2026-03-25T00:27:00+08:00")
	assertFlowchartState(t, db, "old-pending", FlowchartStatusPending, "", "2026-03-25T00:00:00+08:00", "", "2026-03-25T00:00:00+08:00")
}

func TestRepositoryRequeueStaleFlowchartsFallsBackToUpdatedAtWhenRequestedAtMissing(t *testing.T) {
	db := openFlowchartTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, flowchart_status, flowchart_error, flowchart_requested_at, flowchart_finished_at, created_at, updated_at, deleted_at
) VALUES
  ('missing-requested-at', 'processing', '', '', NULL, '2026-03-25T00:00:00+08:00', '2026-03-25T00:05:00+08:00', NULL);
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	repo := NewRepository(db)
	requeued, err := repo.RequeueStaleFlowcharts(
		context.Background(),
		"2026-03-25T00:20:00+08:00",
		"2026-03-25T00:30:00+08:00",
	)
	if err != nil {
		t.Fatalf("RequeueStaleFlowcharts() error = %v", err)
	}
	if got, want := requeued, int64(1); got != want {
		t.Fatalf("RequeueStaleFlowcharts() requeued %d rows, want %d", got, want)
	}

	assertFlowchartState(t, db, "missing-requested-at", FlowchartStatusPending, "", "2026-03-25T00:30:00+08:00", "", "2026-03-25T00:05:00+08:00")
}

func TestFlowchartWorkerEnqueueAutoCandidatesQueuesFirstEligibleRecipe(t *testing.T) {
	db := openFlowchartTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, meal_type, status, ingredients_json, steps_json, flowchart_status, flowchart_error, flowchart_finished_at, flowchart_image_url, parse_status, created_at, updated_at
) VALUES
  ('too-few-steps', '番茄牛腩', 'main', 'wishlist', '{"mainIngredients":["牛腩 500克"]}', '[{"title":"焯水","detail":"焯水去腥。"},{"title":"慢炖","detail":"小火慢炖。"}]', '', '', NULL, '', 'done', '2026-03-25T00:00:00+08:00', '2026-03-25T00:00:00+08:00'),
  ('failed-earlier', '糖醋里脊', 'main', 'wishlist', '{"mainIngredients":["里脊 300克"],"secondaryIngredients":["糖 适量"]}', '[{"title":"腌制","detail":"里脊抓匀腌制。"},{"title":"炸制","detail":"分次炸到定型。"},{"title":"挂汁","detail":"裹匀糖醋汁出锅。"}]', 'failed', 'upstream timeout', '2026-03-24T23:59:00+08:00', '', 'done', '2026-03-24T23:50:00+08:00', '2026-03-24T23:50:00+08:00'),
  ('eligible', '红烧排骨', 'main', 'wishlist', '{"mainIngredients":["排骨 500克"],"secondaryIngredients":["盐 适量"]}', '[{"title":"焯水","detail":"排骨焯水去腥。"},{"title":"炒糖","detail":"小火炒出糖色。"},{"title":"炖煮","detail":"加水炖至软烂。"}]', '', '', NULL, '', 'done', '2026-03-25T00:01:00+08:00', '2026-03-25T00:01:00+08:00'),
  ('has-flowchart', '葱油鸡', 'main', 'wishlist', '{"mainIngredients":["鸡 1只"]}', '[{"title":"处理","detail":"鸡肉擦干。"},{"title":"蒸熟","detail":"蒸到熟透。"},{"title":"淋油","detail":"热油激香。"}]', '', '', NULL, 'https://cdn.example.com/existing.png', 'done', '2026-03-25T00:02:00+08:00', '2026-03-25T00:02:00+08:00'),
  ('parse-running', '麻婆豆腐', 'main', 'wishlist', '{"mainIngredients":["豆腐 1盒"]}', '[{"title":"备料","detail":"豆腐切块。"},{"title":"煸香","detail":"炒香肉末豆瓣。"},{"title":"收汁","detail":"勾芡出锅。"}]', '', '', NULL, '', 'processing', '2026-03-25T00:03:00+08:00', '2026-03-25T00:03:00+08:00');
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	worker := &FlowchartWorker{
		logger:             slog.Default(),
		repo:               NewRepository(db),
		autoEnqueueEnabled: true,
		batchSize:          1,
	}

	worker.enqueueAutoCandidates(context.Background())

	assertFlowchartState(t, db, "too-few-steps", FlowchartStatusIdle, "", "", "", "2026-03-25T00:00:00+08:00")
	assertFlowchartState(t, db, "failed-earlier", FlowchartStatusFailed, "upstream timeout", "", "2026-03-24T23:59:00+08:00", "2026-03-24T23:50:00+08:00")
	assertFlowchartState(t, db, "has-flowchart", FlowchartStatusIdle, "", "", "", "2026-03-25T00:02:00+08:00")
	assertFlowchartState(t, db, "parse-running", FlowchartStatusIdle, "", "", "", "2026-03-25T00:03:00+08:00")

	var pendingCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM recipes WHERE flowchart_status = ?`, FlowchartStatusPending).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending flowcharts error = %v", err)
	}
	if got, want := pendingCount, 1; got != want {
		t.Fatalf("pending flowchart count = %d, want %d", got, want)
	}

	var requestedAt string
	var updatedAt string
	if err := db.QueryRow(`
SELECT COALESCE(flowchart_requested_at, ''), updated_at
FROM recipes
WHERE id = 'eligible'
`).Scan(&requestedAt, &updatedAt); err != nil {
		t.Fatalf("query eligible recipe error = %v", err)
	}
	if requestedAt == "" {
		t.Fatal("eligible recipe should have flowchart_requested_at after auto enqueue")
	}
	if got, want := updatedAt, "2026-03-25T00:01:00+08:00"; got != want {
		t.Fatalf("eligible recipe updated_at = %q, want %q", got, want)
	}
}

func TestFlowchartWorkerEnqueueAutoCandidatesRequeuesFailedRecipeWhenNoEligibleIdleCandidate(t *testing.T) {
	db := openFlowchartTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, meal_type, status, ingredients_json, steps_json, flowchart_status, flowchart_error, flowchart_finished_at, flowchart_image_url, parse_status, created_at, updated_at
) VALUES
  ('too-few-steps', '番茄牛腩', 'main', 'wishlist', '{"mainIngredients":["牛腩 500克"]}', '[{"title":"焯水","detail":"焯水去腥。"},{"title":"慢炖","detail":"小火慢炖。"}]', '', '', NULL, '', 'done', '2026-03-25T00:00:00+08:00', '2026-03-25T00:00:00+08:00'),
  ('failed-eligible', '清蒸鲈鱼', 'main', 'wishlist', '{"mainIngredients":["鲈鱼 1条"],"secondaryIngredients":["姜丝 适量"]}', '[{"title":"改刀","detail":"鱼身划刀方便入味。"},{"title":"铺料","detail":"盘底铺姜葱去腥。"},{"title":"蒸制","detail":"大火蒸到熟透后淋汁。"}]', 'failed', 'temporary upstream error', '2026-03-25T00:05:00+08:00', '', 'done', '2026-03-25T00:01:00+08:00', '2026-03-25T00:01:00+08:00');
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	worker := &FlowchartWorker{
		logger:             slog.Default(),
		repo:               NewRepository(db),
		autoEnqueueEnabled: true,
		batchSize:          1,
	}

	worker.enqueueAutoCandidates(context.Background())

	assertFlowchartState(t, db, "too-few-steps", FlowchartStatusIdle, "", "", "", "2026-03-25T00:00:00+08:00")

	var requestedAt string
	var updatedAt string
	if err := db.QueryRow(`
SELECT COALESCE(flowchart_requested_at, ''), updated_at
FROM recipes
WHERE id = 'failed-eligible'
`).Scan(&requestedAt, &updatedAt); err != nil {
		t.Fatalf("query failed-eligible recipe error = %v", err)
	}
	if requestedAt == "" {
		t.Fatal("failed recipe should have flowchart_requested_at after auto requeue")
	}
	if got, want := updatedAt, "2026-03-25T00:01:00+08:00"; got != want {
		t.Fatalf("failed recipe updated_at = %q, want %q", got, want)
	}

	assertFlowchartState(t, db, "failed-eligible", FlowchartStatusPending, "", requestedAt, "", "2026-03-25T00:01:00+08:00")
}

func TestRepositoryQueueFlowchartDoesNotTouchUpdatedAt(t *testing.T) {
	db := openFlowchartTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, meal_type, status, created_at, updated_at
) VALUES
  ('manual-queue', '红烧肉', 'main', 'wishlist', '2026-03-25T00:00:00+08:00', '2026-03-25T00:10:00+08:00');
`); err != nil {
		t.Fatalf("seed recipe error = %v", err)
	}

	repo := NewRepository(db)
	if err := repo.QueueFlowchart(context.Background(), "manual-queue", "2026-03-25T00:30:00+08:00"); err != nil {
		t.Fatalf("QueueFlowchart() error = %v", err)
	}

	assertFlowchartState(t, db, "manual-queue", FlowchartStatusPending, "", "2026-03-25T00:30:00+08:00", "", "2026-03-25T00:10:00+08:00")
}

func TestRepositoryApplyFlowchartResultDoesNotTouchUpdatedAt(t *testing.T) {
	db := openFlowchartTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (
  id, title, meal_type, status, flowchart_status, flowchart_requested_at, created_at, updated_at
) VALUES
  ('apply-result', '清炖牛腩', 'main', 'wishlist', 'processing', '2026-03-25T00:20:00+08:00', '2026-03-25T00:00:00+08:00', '2026-03-25T00:12:00+08:00');
`); err != nil {
		t.Fatalf("seed recipe error = %v", err)
	}

	repo := NewRepository(db)
	if err := repo.ApplyFlowchartResult(
		context.Background(),
		"apply-result",
		"https://cdn.example.com/flowchart-result.png",
		"flowchart-compat",
		"gpt-image-2-1536x1024",
		"source-hash",
		"2026-03-25T00:40:00+08:00",
	); err != nil {
		t.Fatalf("ApplyFlowchartResult() error = %v", err)
	}

	assertFlowchartState(t, db, "apply-result", FlowchartStatusDone, "", "2026-03-25T00:20:00+08:00", "2026-03-25T00:40:00+08:00", "2026-03-25T00:12:00+08:00")

	var imageURL string
	var provider string
	var model string
	var flowchartUpdatedAt sql.NullString
	if err := db.QueryRow(`
SELECT flowchart_image_url, flowchart_provider, flowchart_model, flowchart_updated_at
FROM recipes
WHERE id = 'apply-result'
`).Scan(&imageURL, &provider, &model, &flowchartUpdatedAt); err != nil {
		t.Fatalf("query apply-result recipe error = %v", err)
	}
	if got, want := imageURL, "https://cdn.example.com/flowchart-result.png"; got != want {
		t.Fatalf("flowchart_image_url = %q, want %q", got, want)
	}
	if got, want := provider, "flowchart-compat"; got != want {
		t.Fatalf("flowchart_provider = %q, want %q", got, want)
	}
	if got, want := model, "gpt-image-2-1536x1024"; got != want {
		t.Fatalf("flowchart_model = %q, want %q", got, want)
	}
	if got, want := flowchartUpdatedAt.String, "2026-03-25T00:40:00+08:00"; got != want {
		t.Fatalf("flowchart_updated_at = %q, want %q", got, want)
	}
}

func openFlowchartTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	if _, err := db.Exec(`
CREATE TABLE recipes (
  id TEXT PRIMARY KEY,
  kitchen_id INTEGER NOT NULL DEFAULT 1,
  title TEXT NOT NULL DEFAULT '',
  ingredient TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  link TEXT NOT NULL DEFAULT '',
  image_url TEXT NOT NULL DEFAULT '',
  image_urls_json TEXT NOT NULL DEFAULT '[]',
  image_meta_json TEXT NOT NULL DEFAULT '[]',
  flowchart_image_url TEXT NOT NULL DEFAULT '',
  flowchart_provider TEXT NOT NULL DEFAULT '',
  flowchart_model TEXT NOT NULL DEFAULT '',
  flowchart_updated_at TEXT,
  flowchart_source_hash TEXT NOT NULL DEFAULT '',
  flowchart_status TEXT NOT NULL DEFAULT '',
  flowchart_error TEXT NOT NULL DEFAULT '',
  flowchart_requested_at TEXT,
  flowchart_finished_at TEXT,
  meal_type TEXT NOT NULL DEFAULT 'main',
  status TEXT NOT NULL DEFAULT 'wishlist',
  note TEXT NOT NULL DEFAULT '',
  ingredients_json TEXT NOT NULL DEFAULT '{}',
  steps_json TEXT NOT NULL DEFAULT '[]',
  parse_status TEXT NOT NULL DEFAULT '',
  parse_source TEXT NOT NULL DEFAULT '',
  parse_error TEXT NOT NULL DEFAULT '',
  parse_requested_at TEXT NOT NULL DEFAULT '',
  parse_finished_at TEXT NOT NULL DEFAULT '',
  parsed_content_edited INTEGER NOT NULL DEFAULT 0,
  pinned_at TEXT NOT NULL DEFAULT '',
  created_by INTEGER NOT NULL DEFAULT 0,
  updated_by INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  deleted_at TEXT
);
`); err != nil {
		db.Close()
		t.Fatalf("create recipes table error = %v", err)
	}

	return db
}

func assertFlowchartState(t *testing.T, db *sql.DB, recipeID, wantStatus, wantError, wantRequestedAt, wantFinishedAt, wantUpdatedAt string) {
	t.Helper()

	var gotStatus string
	var gotError string
	var gotRequestedAt sql.NullString
	var gotFinishedAt sql.NullString
	var gotUpdatedAt string
	if err := db.QueryRow(`
SELECT flowchart_status, flowchart_error, flowchart_requested_at, flowchart_finished_at, updated_at
FROM recipes
WHERE id = ?
`, recipeID).Scan(&gotStatus, &gotError, &gotRequestedAt, &gotFinishedAt, &gotUpdatedAt); err != nil {
		t.Fatalf("query recipe %s error = %v", recipeID, err)
	}

	if gotStatus != wantStatus {
		t.Fatalf("recipe %s flowchart_status = %q, want %q", recipeID, gotStatus, wantStatus)
	}
	if gotError != wantError {
		t.Fatalf("recipe %s flowchart_error = %q, want %q", recipeID, gotError, wantError)
	}
	if got, want := gotRequestedAt.String, wantRequestedAt; got != want {
		t.Fatalf("recipe %s flowchart_requested_at = %q, want %q", recipeID, got, want)
	}
	if got, want := gotFinishedAt.String, wantFinishedAt; got != want {
		t.Fatalf("recipe %s flowchart_finished_at = %q, want %q", recipeID, got, want)
	}
	if gotUpdatedAt != wantUpdatedAt {
		t.Fatalf("recipe %s updated_at = %q, want %q", recipeID, gotUpdatedAt, wantUpdatedAt)
	}
}
