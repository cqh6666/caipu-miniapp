package aialert

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
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
		Scene:        "summary",
		ProviderID:   "summary-main",
		ProviderName: "主节点",
		Model:        "gpt-test",
		ErrorType:    "timeout",
		ErrorMessage: "request timeout",
		RequestID:    "req-1",
		HTTPStatus:   504,
		OccurredAt:   "2026-04-15T08:00:00Z",
	}

	service.RecordFailure(context.Background(), event)
	service.RecordFailure(context.Background(), event)
	if len(sender.requests) != 0 {
		t.Fatalf("sender.requests = %d, want 0", len(sender.requests))
	}

	service.RecordFailure(context.Background(), event)
	if len(sender.requests) != 1 {
		t.Fatalf("sender.requests = %d, want 1", len(sender.requests))
	}

	service.RecordFailure(context.Background(), event)
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

	service.RecordFailure(context.Background(), event)
	service.RecordFailure(context.Background(), event)
	service.RecordFailure(context.Background(), event)
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
);`
	if _, err := db.Exec(statement); err != nil {
		t.Fatalf("db.Exec() error = %v", err)
	}
	return db
}
