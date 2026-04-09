package audit

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

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

func TestServiceOverviewAndTrendsHandlePositiveDurations(t *testing.T) {
	t.Parallel()

	db := openAuditTestDB(t)
	service := NewService(db, slog.New(slog.NewTextHandler(io.Discard, nil)))

	bucketTime := time.Now().UTC().Truncate(time.Hour).Add(-2 * time.Hour)
	jobStartedAtA := bucketTime.Add(15 * time.Minute).Format(time.RFC3339)
	jobStartedAtB := bucketTime.Add(35 * time.Minute).Format(time.RFC3339)
	callCreatedAtA := bucketTime.Add(5 * time.Minute).Format(time.RFC3339)
	callCreatedAtB := bucketTime.Add(40 * time.Minute).Format(time.RFC3339)

	if _, err := db.Exec(`
INSERT INTO ai_job_runs (
	scene,
	target_type,
	target_id,
	trigger_source,
	status,
	final_provider,
	final_model,
	fallback_used,
	error_message,
	request_id,
	started_at,
	finished_at,
	duration_ms,
	meta_json
) VALUES
	(?, ?, ?, ?, ?, ?, ?, 0, '', ?, ?, ?, ?, '{}'),
	(?, ?, ?, ?, ?, ?, ?, 0, '', ?, ?, ?, ?, '{}')
`,
		SceneParseSummary, "recipe", "rec_1", "worker", JobStatusSuccess, "openai-compatible", "gpt-test", "req-1", jobStartedAtA, jobStartedAtA, 1000,
		SceneParseSummary, "recipe", "rec_2", "worker", JobStatusSuccess, "openai-compatible", "gpt-test", "req-2", jobStartedAtB, jobStartedAtB, 2000,
	); err != nil {
		t.Fatalf("insert ai_job_runs returned error: %v", err)
	}

	if _, err := db.Exec(`
INSERT INTO ai_call_logs (
	job_run_id,
	scene,
	provider,
	endpoint,
	model,
	status,
	http_status,
	latency_ms,
	error_type,
	error_message,
	request_id,
	meta_json,
	created_at
) VALUES
	(1, ?, ?, ?, ?, ?, 200, 120, '', '', ?, '{}', ?),
	(2, ?, ?, ?, ?, ?, 200, 240, '', '', ?, '{}', ?)
`,
		SceneParseSummary, "openai-compatible", "/chat/completions", "gpt-test", CallStatusSuccess, "req-1", callCreatedAtA,
		SceneParseSummary, "openai-compatible", "/chat/completions", "gpt-test", CallStatusSuccess, "req-2", callCreatedAtB,
	); err != nil {
		t.Fatalf("insert ai_call_logs returned error: %v", err)
	}

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}
	if overview.TaskTotal != 2 {
		t.Fatalf("overview.TaskTotal = %d, want 2", overview.TaskTotal)
	}
	if overview.APITotal != 2 {
		t.Fatalf("overview.APITotal = %d, want 2", overview.APITotal)
	}
	if overview.AvgDurationMS != 1500 {
		t.Fatalf("overview.AvgDurationMS = %d, want 1500", overview.AvgDurationMS)
	}

	trends, err := service.Trends(context.Background(), "24h")
	if err != nil {
		t.Fatalf("Trends returned error: %v", err)
	}
	if len(trends) != 1 {
		t.Fatalf("len(trends) = %d, want 1", len(trends))
	}

	expectedBucket := bucketTime.Format(time.RFC3339)
	if trends[0].Bucket != expectedBucket {
		t.Fatalf("trends[0].Bucket = %q, want %q", trends[0].Bucket, expectedBucket)
	}
	if trends[0].Label != bucketTime.Format("01-02 15:04") {
		t.Fatalf("trends[0].Label = %q, want %q", trends[0].Label, bucketTime.Format("01-02 15:04"))
	}
	if trends[0].AvgDurationMS != 1500 {
		t.Fatalf("trends[0].AvgDurationMS = %d, want 1500", trends[0].AvgDurationMS)
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
