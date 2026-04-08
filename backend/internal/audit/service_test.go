package audit

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"
)

func TestServiceStartJobLogCallAndQuery(t *testing.T) {
	t.Parallel()

	db := openAuditTestDB(t)
	service := NewService(db, slog.New(slog.NewTextHandler(io.Discard, nil)))

	jobID, finish, err := service.StartJob(context.Background(), JobInput{
		Scene:         SceneParseSummary,
		TargetType:    "recipe",
		TargetID:      "rec_123",
		TriggerSource: "worker",
		RequestID:     "req-1",
		Meta: map[string]any{
			"platform": "bilibili",
		},
	})
	if err != nil {
		t.Fatalf("StartJob returned error: %v", err)
	}
	if jobID <= 0 {
		t.Fatalf("jobID = %d, want > 0", jobID)
	}

	if err := service.LogCall(context.Background(), CallLogInput{
		JobRunID:   jobID,
		Scene:      SceneParseSummary,
		Provider:   "openai-compatible",
		Endpoint:   "/chat/completions",
		Model:      "gpt-test",
		Status:     CallStatusSuccess,
		HTTPStatus: 200,
		LatencyMS:  123,
		RequestID:  "req-1",
	}); err != nil {
		t.Fatalf("LogCall returned error: %v", err)
	}

	if err := finish(context.Background(), JobResult{
		Status:        JobStatusSuccess,
		FinalProvider: "openai-compatible",
		FinalModel:    "gpt-test",
	}); err != nil {
		t.Fatalf("finish returned error: %v", err)
	}

	jobs, err := service.ListJobs(context.Background(), JobListFilter{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("ListJobs returned error: %v", err)
	}
	if jobs.Total != 1 || len(jobs.Items) != 1 {
		t.Fatalf("jobs = %+v, want 1 item", jobs)
	}
	if jobs.Items[0].Status != JobStatusSuccess {
		t.Fatalf("status = %q, want %q", jobs.Items[0].Status, JobStatusSuccess)
	}

	calls, err := service.ListCalls(context.Background(), CallListFilter{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("ListCalls returned error: %v", err)
	}
	if calls.Total != 1 || len(calls.Items) != 1 {
		t.Fatalf("calls = %+v, want 1 item", calls)
	}
	if calls.Items[0].Provider != "openai-compatible" {
		t.Fatalf("provider = %q, want %q", calls.Items[0].Provider, "openai-compatible")
	}

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}
	if overview.TaskTotal != 1 {
		t.Fatalf("overview.TaskTotal = %d, want 1", overview.TaskTotal)
	}
	if overview.APITotal != 1 {
		t.Fatalf("overview.APITotal = %d, want 1", overview.APITotal)
	}
}

func openAuditTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	statements := []string{
		`CREATE TABLE ai_job_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scene TEXT NOT NULL,
			target_type TEXT NOT NULL DEFAULT '',
			target_id TEXT NOT NULL DEFAULT '',
			trigger_source TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			final_provider TEXT NOT NULL DEFAULT '',
			final_model TEXT NOT NULL DEFAULT '',
			fallback_used INTEGER NOT NULL DEFAULT 0,
			error_message TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL DEFAULT '',
			started_at TEXT NOT NULL,
			finished_at TEXT NOT NULL DEFAULT '',
			duration_ms INTEGER NOT NULL DEFAULT 0,
			meta_json TEXT NOT NULL DEFAULT '{}'
		);`,
		`CREATE TABLE ai_call_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_run_id INTEGER,
			scene TEXT NOT NULL,
			provider TEXT NOT NULL DEFAULT '',
			endpoint TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			http_status INTEGER NOT NULL DEFAULT 0,
			latency_ms INTEGER NOT NULL DEFAULT 0,
			error_type TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL DEFAULT '',
			meta_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL
		);`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("Exec(%q) returned error: %v", statement, err)
		}
	}

	return db
}
