package recipe

import (
	"context"
	"database/sql"
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

	assertFlowchartState(t, db, "stale-processing", FlowchartStatusPending, "", "2026-03-25T00:30:00+08:00", "", "2026-03-25T00:30:00+08:00")
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

	assertFlowchartState(t, db, "missing-requested-at", FlowchartStatusPending, "", "2026-03-25T00:30:00+08:00", "", "2026-03-25T00:30:00+08:00")
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
  flowchart_status TEXT NOT NULL DEFAULT '',
  flowchart_error TEXT NOT NULL DEFAULT '',
  flowchart_requested_at TEXT,
  flowchart_finished_at TEXT,
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
