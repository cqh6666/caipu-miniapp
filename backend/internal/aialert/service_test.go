package aialert

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestServiceSendsAlertOncePerFailureStreak(t *testing.T) {
	t.Parallel()

	db := openAlertTestDB(t)
	repo := NewRepository(db)
	sender := &fakeSender{}
	service := NewService(repo, staticConfigProvider{
		Config: Config{
			Enabled:          true,
			FailureThreshold: 3,
			SMTPHost:         "smtp.qq.com",
			SMTPPort:         587,
			SMTPUsername:     "bot@qq.com",
			SMTPPassword:     "auth-code",
			ToEmails:         "ops@qq.com",
		},
	}, sender, slog.New(slog.NewTextHandler(io.Discard, nil)))

	event := Event{
		Scene:         "summary",
		ProviderID:    "summary-main",
		ProviderName:  "主节点",
		Model:         "gpt-test",
		ErrorType:     "timeout",
		ErrorMessage:  "request timeout",
		RequestID:     "req-1",
		HTTPStatus:    504,
		TriggerSource: "worker",
		TargetType:    "recipe",
		TargetID:      "recipe-123",
		OccurredAt:    "2026-04-15T08:00:00Z",
	}

	for attempt := 1; attempt <= 2; attempt++ {
		current := event
		current.RequestID = fmt.Sprintf("req-%d", attempt)
		current.ErrorMessage = fmt.Sprintf("request timeout %d", attempt)
		current.OccurredAt = fmt.Sprintf("2026-04-15T08:00:0%dZ", attempt)
		insertFailureCallLog(t, db, current)
		service.RecordFailure(context.Background(), current)
	}
	if len(sender.requests) != 0 {
		t.Fatalf("sender.requests = %d, want 0", len(sender.requests))
	}

	current := event
	current.RequestID = "req-3"
	current.ErrorMessage = "request timeout 3"
	current.OccurredAt = "2026-04-15T08:00:03Z"
	insertFailureCallLog(t, db, current)
	service.RecordFailure(context.Background(), current)
	if len(sender.requests) != 1 {
		t.Fatalf("sender.requests = %d, want 1", len(sender.requests))
	}
	if got := sender.requests[0].Subject; !strings.Contains(got, "做法总结") || !strings.Contains(got, "主节点(summary-main)") {
		t.Fatalf("sender.requests[0].Subject = %q, want scene/provider label", got)
	}
	body := sender.requests[0].Body
	for _, want := range []string{
		"触发来源: 后台 Worker",
		"目标对象: recipe / recipe-123",
		"最近 3 次失败摘要:",
		"req-3",
		"req-2",
		"req-1",
		"排查建议：",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("sender.requests[0].Body missing %q, body = %q", want, body)
		}
	}

	current = event
	current.RequestID = "req-4"
	current.ErrorMessage = "request timeout 4"
	current.OccurredAt = "2026-04-15T08:00:04Z"
	insertFailureCallLog(t, db, current)
	service.RecordFailure(context.Background(), current)
	if len(sender.requests) != 1 {
		t.Fatalf("sender.requests = %d, want still 1", len(sender.requests))
	}

	state, found, err := repo.GetState(context.Background(), "summary-main")
	if err != nil {
		t.Fatalf("repo.GetState() error = %v", err)
	}
	if !found {
		t.Fatalf("repo.GetState() found = false, want true")
	}
	if state.ConsecutiveFailures != 4 {
		t.Fatalf("state.ConsecutiveFailures = %d, want 4", state.ConsecutiveFailures)
	}
	if state.LastAlertedFailureCount != 3 {
		t.Fatalf("state.LastAlertedFailureCount = %d, want 3", state.LastAlertedFailureCount)
	}

	service.RecordSuccess(context.Background(), Event{
		Scene:      "summary",
		ProviderID: "summary-main",
		RequestID:  "req-2",
		OccurredAt: "2026-04-15T08:05:00Z",
	})

	state, found, err = repo.GetState(context.Background(), "summary-main")
	if err != nil {
		t.Fatalf("repo.GetState() after success error = %v", err)
	}
	if !found {
		t.Fatalf("repo.GetState() after success found = false, want true")
	}
	if state.ConsecutiveFailures != 0 {
		t.Fatalf("state.ConsecutiveFailures = %d, want 0", state.ConsecutiveFailures)
	}
	if state.LastAlertedFailureCount != 0 {
		t.Fatalf("state.LastAlertedFailureCount = %d, want 0", state.LastAlertedFailureCount)
	}

	for attempt := 5; attempt <= 7; attempt++ {
		current = event
		current.RequestID = fmt.Sprintf("req-%d", attempt)
		current.ErrorMessage = fmt.Sprintf("request timeout %d", attempt)
		current.OccurredAt = fmt.Sprintf("2026-04-15T08:00:%02dZ", attempt)
		insertFailureCallLog(t, db, current)
		service.RecordFailure(context.Background(), current)
	}
	if len(sender.requests) != 2 {
		t.Fatalf("sender.requests = %d, want 2 after new streak", len(sender.requests))
	}
}

type staticConfigProvider struct {
	Config Config
}

func (p staticConfigProvider) AIProviderAlert(context.Context) Config {
	return p.Config
}

type fakeSender struct {
	requests []SendRequest
}

func (f *fakeSender) Send(_ context.Context, request SendRequest) error {
	f.requests = append(f.requests, request)
	return nil
}

func openAlertTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	statement := `
CREATE TABLE ai_provider_alert_states (
	provider_id TEXT PRIMARY KEY,
	scene TEXT NOT NULL DEFAULT '',
	provider_name TEXT NOT NULL DEFAULT '',
	model TEXT NOT NULL DEFAULT '',
	consecutive_failures INTEGER NOT NULL DEFAULT 0,
	last_status TEXT NOT NULL DEFAULT '',
	last_error_type TEXT NOT NULL DEFAULT '',
	last_error_message TEXT NOT NULL DEFAULT '',
	last_http_status INTEGER NOT NULL DEFAULT 0,
	last_request_id TEXT NOT NULL DEFAULT '',
	last_failed_at TEXT NOT NULL DEFAULT '',
	last_recovered_at TEXT NOT NULL DEFAULT '',
	last_alerted_at TEXT NOT NULL DEFAULT '',
	last_alerted_failure_count INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE ai_call_logs (
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
);`
	if _, err := db.Exec(statement); err != nil {
		t.Fatalf("db.Exec() error = %v", err)
	}
	return db
}

func insertFailureCallLog(t *testing.T, db *sql.DB, event Event) {
	t.Helper()

	if _, err := db.Exec(`
INSERT INTO ai_call_logs (
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
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}', ?)
`,
		event.Scene,
		event.ProviderID,
		"/chat/completions",
		event.Model,
		"timeout",
		event.HTTPStatus,
		3000,
		event.ErrorType,
		event.ErrorMessage,
		event.RequestID,
		event.OccurredAt,
	); err != nil {
		t.Fatalf("insert ai_call_logs error = %v", err)
	}
}
