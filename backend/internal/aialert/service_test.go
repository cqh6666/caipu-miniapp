package aialert

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

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
	if sender.Count() != 0 {
		t.Fatalf("sender.requests = %d, want 0", sender.Count())
	}

	current := event
	current.RequestID = "req-3"
	current.ErrorMessage = "request timeout 3"
	current.OccurredAt = "2026-04-15T08:00:03Z"
	insertFailureCallLog(t, db, current)
	service.RecordFailure(context.Background(), current)
	if sender.Count() != 1 {
		t.Fatalf("sender.requests = %d, want 1", sender.Count())
	}
	requests := sender.Requests()
	if got := requests[0].Subject; !strings.Contains(got, "做法总结") || !strings.Contains(got, "主节点(summary-main)") {
		t.Fatalf("sender.requests[0].Subject = %q, want scene/provider label", got)
	}
	body := requests[0].Body
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
	if sender.Count() != 1 {
		t.Fatalf("sender.requests = %d, want still 1", sender.Count())
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
	if state.FailureStreakID != "" {
		t.Fatalf("state.FailureStreakID = %q, want empty after success", state.FailureStreakID)
	}

	for attempt := 5; attempt <= 7; attempt++ {
		current = event
		current.RequestID = fmt.Sprintf("req-%d", attempt)
		current.ErrorMessage = fmt.Sprintf("request timeout %d", attempt)
		current.OccurredAt = fmt.Sprintf("2026-04-15T08:00:%02dZ", attempt)
		insertFailureCallLog(t, db, current)
		service.RecordFailure(context.Background(), current)
	}
	if sender.Count() != 2 {
		t.Fatalf("sender.requests = %d, want 2 after new streak", sender.Count())
	}
}

func TestServiceConcurrentFailuresCreateAndSendOneDelivery(t *testing.T) {
	db := openAlertTestDB(t)
	repo := NewRepository(db)
	sender := &fakeSender{}
	service := NewService(repo, staticConfigProvider{Config: lifecycleConfig()}, sender, slog.New(slog.NewTextHandler(io.Discard, nil)))

	const concurrency = 20
	start := make(chan struct{})
	var wg sync.WaitGroup
	for index := 0; index < concurrency; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			service.RecordFailure(context.Background(), Event{
				Scene:        "summary",
				ProviderID:   "summary-concurrent",
				ProviderName: "并发节点",
				Model:        "gpt-test",
				ErrorType:    "timeout",
				ErrorMessage: "request timeout",
				RequestID:    fmt.Sprintf("req-concurrent-%d", index),
				HTTPStatus:   504,
			})
		}(index)
	}
	close(start)
	wg.Wait()

	if got := sender.Count(); got != 1 {
		t.Fatalf("sender.Count() = %d, want 1", got)
	}
	var deliveryCount, sentCount int
	if err := db.QueryRow(`SELECT COUNT(*), SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END) FROM ai_provider_alert_deliveries`).Scan(&deliveryCount, &sentCount); err != nil {
		t.Fatalf("query deliveries: %v", err)
	}
	if deliveryCount != 1 || sentCount != 1 {
		t.Fatalf("deliveries total/sent = %d/%d, want 1/1", deliveryCount, sentCount)
	}
}

func TestServiceFailedDeliveryReturnsToPendingAndRetries(t *testing.T) {
	db := openAlertTestDB(t)
	repo := NewRepository(db)
	sender := &failOnceSender{}
	service := NewService(repo, staticConfigProvider{Config: lifecycleConfig()}, sender, slog.New(slog.NewTextHandler(io.Discard, nil)))

	for attempt := 1; attempt <= 3; attempt++ {
		service.RecordFailure(context.Background(), Event{
			Scene:        "summary",
			ProviderID:   "summary-retry",
			ProviderName: "重试节点",
			Model:        "gpt-test",
			ErrorType:    "timeout",
			ErrorMessage: "request timeout",
			RequestID:    fmt.Sprintf("req-retry-%d", attempt),
			HTTPStatus:   504,
		})
	}

	var status string
	var attemptCount int
	var lastError string
	if err := db.QueryRow(`
SELECT status, attempt_count, last_error
FROM ai_provider_alert_deliveries
WHERE provider_id = 'summary-retry'
`).Scan(&status, &attemptCount, &lastError); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || attemptCount != 1 || !strings.Contains(lastError, "temporary SMTP failure") {
		t.Fatalf("delivery after failed send = status=%q attempts=%d error=%q", status, attemptCount, lastError)
	}
	if _, err := db.Exec(`
UPDATE ai_provider_alert_deliveries
SET available_at = '1970-01-01T00:00:00Z'
WHERE provider_id = 'summary-retry'
`); err != nil {
		t.Fatal(err)
	}
	dispatched, err := service.DispatchPending(context.Background(), 1)
	if err != nil {
		t.Fatalf("DispatchPending() error = %v", err)
	}
	if dispatched != 1 {
		t.Fatalf("DispatchPending() = %d, want 1", dispatched)
	}
	if err := db.QueryRow(`
SELECT status, attempt_count
FROM ai_provider_alert_deliveries
WHERE provider_id = 'summary-retry'
`).Scan(&status, &attemptCount); err != nil {
		t.Fatal(err)
	}
	if status != "sent" || attemptCount != 2 || sender.SuccessCount() != 1 {
		t.Fatalf("delivery after retry = status=%q attempts=%d successes=%d", status, attemptCount, sender.SuccessCount())
	}
	state, found, err := repo.GetState(context.Background(), "summary-retry")
	if err != nil || !found {
		t.Fatalf("GetState() found=%t error=%v", found, err)
	}
	if state.LastAlertedFailureCount != 3 || state.LastAlertedAt == "" {
		t.Fatalf("alerted state after retry = %#v", state)
	}
}

func TestDeliveryWorkerDispatchesQueuedAlertWithoutNewFailure(t *testing.T) {
	db := openAlertTestDB(t)
	repo := NewRepository(db)
	for attempt := 1; attempt <= 3; attempt++ {
		_, _, err := repo.RecordFailure(context.Background(), Event{
			Scene:        "summary",
			ProviderID:   "summary-worker",
			ProviderName: "Worker 节点",
			Model:        "gpt-test",
			ErrorType:    "timeout",
			RequestID:    fmt.Sprintf("req-worker-%d", attempt),
		}, 3)
		if err != nil {
			t.Fatalf("RecordFailure(%d) error = %v", attempt, err)
		}
	}

	sender := &fakeSender{}
	service := NewService(repo, staticConfigProvider{Config: lifecycleConfig()}, sender, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.deliveryWorker.interval = 5 * time.Millisecond
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Stop(stopCtx)
	})

	deadline := time.Now().Add(time.Second)
	for sender.Count() != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if sender.Count() != 1 {
		t.Fatalf("worker sender.Count() = %d, want 1", sender.Count())
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM ai_provider_alert_deliveries WHERE provider_id = 'summary-worker'`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "sent" {
		t.Fatalf("worker delivery status = %q, want sent", status)
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestRepositoryReclaimsExpiredDeliveryLease(t *testing.T) {
	db := openAlertTestDB(t)
	repo := NewRepository(db)
	for attempt := 1; attempt <= 3; attempt++ {
		if _, _, err := repo.RecordFailure(context.Background(), Event{
			Scene:      "summary",
			ProviderID: "summary-expired-claim",
			RequestID:  fmt.Sprintf("req-claim-%d", attempt),
		}, 3); err != nil {
			t.Fatal(err)
		}
	}
	first, found, err := repo.ClaimNextDelivery(context.Background(), time.Millisecond)
	if err != nil || !found {
		t.Fatalf("first ClaimNextDelivery() found=%t error=%v", found, err)
	}
	time.Sleep(3 * time.Millisecond)
	second, found, err := repo.ClaimNextDelivery(context.Background(), time.Second)
	if err != nil || !found {
		t.Fatalf("second ClaimNextDelivery() found=%t error=%v", found, err)
	}
	if first.EventID != second.EventID || first.ClaimToken == second.ClaimToken || second.AttemptCount != 2 {
		t.Fatalf("reclaimed delivery first=%#v second=%#v", first, second)
	}
}

func TestDeliveryWorkerStopCancelsInFlightSender(t *testing.T) {
	db := openAlertTestDB(t)
	repo := NewRepository(db)
	for attempt := 1; attempt <= 3; attempt++ {
		if _, _, err := repo.RecordFailure(context.Background(), Event{
			Scene:      "summary",
			ProviderID: "summary-stop",
			RequestID:  fmt.Sprintf("req-stop-%d", attempt),
		}, 3); err != nil {
			t.Fatal(err)
		}
	}
	sender := &contextBlockingSender{started: make(chan struct{})}
	service := NewService(repo, staticConfigProvider{Config: lifecycleConfig()}, sender, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.deliveryWorker.interval = time.Hour
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	select {
	case <-sender.started:
	case <-time.After(time.Second):
		t.Fatal("sender did not start")
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := service.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestServiceOverviewBuildsSortedSummary(t *testing.T) {
	t.Parallel()

	// 最近失败落在活跃窗口内，保证达到阈值的节点判定为 active（无 resolver 时按“仍在路由”兜底）。
	recentFailedAt := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339)

	db := openAlertTestDB(t)
	repo := NewRepository(db)
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
	}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	insertAlertState(t, db, State{
		ProviderID:          "provider-b",
		Scene:               "summary",
		ProviderName:        "主节点 B",
		Model:               "gpt-b",
		ConsecutiveFailures: 4,
		LastStatus:          "failed",
		LastErrorType:       "timeout",
		LastErrorMessage:    "request timeout",
		LastRequestID:       "req-b",
		LastFailedAt:        recentFailedAt,
		LastRecoveredAt:     "",
		LastAlertedAt:       "2026-04-25T09:05:00Z",
		UpdatedAt:           "2026-04-25T09:06:00Z",
	})
	insertAlertState(t, db, State{
		ProviderID:          "provider-a",
		Scene:               "title",
		ProviderName:        "主节点 A",
		Model:               "gpt-a",
		ConsecutiveFailures: 3,
		LastStatus:          "failed",
		LastErrorType:       "upstream",
		LastErrorMessage:    "upstream unavailable",
		LastRequestID:       "req-a",
		LastFailedAt:        recentFailedAt,
		LastRecoveredAt:     "",
		LastAlertedAt:       "2026-04-25T09:12:00Z",
		UpdatedAt:           "2026-04-25T09:03:00Z",
	})
	insertAlertState(t, db, State{
		ProviderID:          "provider-c",
		Scene:               "flowchart",
		ProviderName:        "备用节点 C",
		Model:               "gpt-c",
		ConsecutiveFailures: 2,
		LastStatus:          "failed",
		LastErrorType:       "network",
		LastErrorMessage:    "connection refused",
		LastRequestID:       "req-c",
		LastFailedAt:        recentFailedAt,
		LastRecoveredAt:     "",
		LastAlertedAt:       "2026-04-25T09:01:00Z",
		UpdatedAt:           "2026-04-25T09:10:00Z",
	})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}

	if !overview.Enabled {
		t.Fatal("overview.Enabled = false, want true")
	}
	if overview.FailureThreshold != 3 {
		t.Fatalf("overview.FailureThreshold = %d, want 3", overview.FailureThreshold)
	}
	if !overview.HasDeliveryConfig {
		t.Fatal("overview.HasDeliveryConfig = false, want true")
	}
	if overview.ActiveAlertCount != 2 {
		t.Fatalf("overview.ActiveAlertCount = %d, want 2", overview.ActiveAlertCount)
	}
	if overview.LatestAlertedAt != "2026-04-25T09:12:00Z" {
		t.Fatalf("overview.LatestAlertedAt = %q, want %q", overview.LatestAlertedAt, "2026-04-25T09:12:00Z")
	}
	if len(overview.Items) != 3 {
		t.Fatalf("len(overview.Items) = %d, want 3", len(overview.Items))
	}
	gotOrder := []string{
		overview.Items[0].ProviderID,
		overview.Items[1].ProviderID,
		overview.Items[2].ProviderID,
	}
	wantOrder := []string{"provider-b", "provider-a", "provider-c"}
	for index, want := range wantOrder {
		if gotOrder[index] != want {
			t.Fatalf("overview.Items[%d].ProviderID = %q, want %q (full order = %v)", index, gotOrder[index], want, gotOrder)
		}
	}
	if !overview.Items[0].ThresholdReached || !overview.Items[1].ThresholdReached {
		t.Fatalf("thresholdReached for first two items = %v, %v, want both true", overview.Items[0].ThresholdReached, overview.Items[1].ThresholdReached)
	}
	if overview.Items[2].ThresholdReached {
		t.Fatal("overview.Items[2].ThresholdReached = true, want false")
	}
}

func TestServiceOverviewMarksIncompleteDeliveryConfig(t *testing.T) {
	t.Parallel()

	db := openAlertTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo, staticConfigProvider{
		Config: Config{
			Enabled:          false,
			FailureThreshold: 4,
			SMTPUsername:     "bot@qq.com",
		},
	}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
	if overview.Enabled {
		t.Fatal("overview.Enabled = true, want false")
	}
	if overview.HasDeliveryConfig {
		t.Fatal("overview.HasDeliveryConfig = true, want false")
	}
	if overview.ActiveAlertCount != 0 {
		t.Fatalf("overview.ActiveAlertCount = %d, want 0", overview.ActiveAlertCount)
	}
}

type staticConfigProvider struct {
	Config Config
}

func (p staticConfigProvider) AIProviderAlert(context.Context) Config {
	return p.Config
}

type fakeSender struct {
	mu       sync.Mutex
	requests []SendRequest
}

type failOnceSender struct {
	mu        sync.Mutex
	attempts  int
	successes int
}

type contextBlockingSender struct {
	started chan struct{}
	once    sync.Once
}

func (s *contextBlockingSender) Send(ctx context.Context, _ SendRequest) error {
	s.once.Do(func() { close(s.started) })
	<-ctx.Done()
	return ctx.Err()
}

func (f *failOnceSender) Send(context.Context, SendRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts++
	if f.attempts == 1 {
		return errors.New("temporary SMTP failure")
	}
	f.successes++
	return nil
}

func (f *failOnceSender) SuccessCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.successes
}

func (f *fakeSender) Send(_ context.Context, request SendRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, request)
	return nil
}

func (f *fakeSender) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeSender) Requests() []SendRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]SendRequest(nil), f.requests...)
}

func openAlertTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	statement := `
CREATE TABLE ai_provider_alert_states (
	provider_id TEXT PRIMARY KEY,
	failure_streak_id TEXT NOT NULL DEFAULT '',
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
	archived_at TEXT NOT NULL DEFAULT '',
	archived_by TEXT NOT NULL DEFAULT '',
	archive_reason TEXT NOT NULL DEFAULT '',
	muted_until TEXT NOT NULL DEFAULT '',
	muted_by TEXT NOT NULL DEFAULT '',
	mute_reason TEXT NOT NULL DEFAULT '',
	last_config_changed_at TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE ai_provider_alert_deliveries (
	event_id TEXT PRIMARY KEY,
	failure_streak_id TEXT NOT NULL UNIQUE,
	provider_id TEXT NOT NULL,
	scene TEXT NOT NULL DEFAULT '',
	trigger_source TEXT NOT NULL DEFAULT '',
	target_type TEXT NOT NULL DEFAULT '',
	target_id TEXT NOT NULL DEFAULT '',
	request_id TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending',
	attempt_count INTEGER NOT NULL DEFAULT 0,
	claim_token TEXT NOT NULL DEFAULT '',
	claim_expires_at TEXT NOT NULL DEFAULT '',
	available_at TEXT NOT NULL,
	last_error TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	sent_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE ai_provider_alert_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	provider_id TEXT NOT NULL,
	scene TEXT NOT NULL DEFAULT '',
	event_type TEXT NOT NULL,
	reason TEXT NOT NULL DEFAULT '',
	operator_subject TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
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

func insertAlertState(t *testing.T, db *sql.DB, state State) {
	t.Helper()

	if _, err := db.Exec(`
INSERT INTO ai_provider_alert_states (
	provider_id,
	failure_streak_id,
	scene,
	provider_name,
	model,
	consecutive_failures,
	last_status,
	last_error_type,
	last_error_message,
	last_http_status,
	last_request_id,
	last_failed_at,
	last_recovered_at,
	last_alerted_at,
	last_alerted_failure_count,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		state.ProviderID,
		state.FailureStreakID,
		state.Scene,
		state.ProviderName,
		state.Model,
		state.ConsecutiveFailures,
		state.LastStatus,
		state.LastErrorType,
		state.LastErrorMessage,
		state.LastHTTPStatus,
		state.LastRequestID,
		state.LastFailedAt,
		state.LastRecoveredAt,
		state.LastAlertedAt,
		state.LastAlertedFailureCount,
		state.UpdatedAt,
	); err != nil {
		t.Fatalf("insert ai_provider_alert_states error = %v", err)
	}
}
