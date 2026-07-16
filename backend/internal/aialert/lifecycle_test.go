package aialert

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

type fakeResolver struct {
	statuses map[string]ProviderRuntimeStatus
	err      error
}

func (f fakeResolver) ResolveProviderStatuses(context.Context) (map[string]ProviderRuntimeStatus, error) {
	return f.statuses, f.err
}

type fakeRetester struct {
	outcome    ProviderRetestOutcome
	found      bool
	err        error
	calledWith string
}

func (f *fakeRetester) RetestProvider(_ context.Context, providerID string) (ProviderRetestOutcome, bool, error) {
	f.calledWith = providerID
	return f.outcome, f.found, f.err
}

func lifecycleConfig() Config {
	return Config{
		Enabled:           true,
		FailureThreshold:  3,
		ActiveWindowHours: 72,
		SMTPHost:          "smtp.qq.com",
		SMTPPort:          587,
		SMTPUsername:      "bot@qq.com",
		SMTPPassword:      "auth-code",
		FromEmail:         "bot@qq.com",
		ToEmails:          "ops@qq.com",
	}
}

func findItem(items []OverviewItem, providerID string) (OverviewItem, bool) {
	for _, item := range items {
		if item.ProviderID == providerID {
			return item, true
		}
	}
	return OverviewItem{}, false
}

func TestOverviewDerivesStatuses(t *testing.T) {
	t.Parallel()

	db := openAlertTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo, staticConfigProvider{Config: lifecycleConfig()}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	recent := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339)
	old := time.Now().UTC().Add(-100 * time.Hour).Format(time.RFC3339)
	recovered := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)

	insertAlertState(t, db, State{ProviderID: "p-active", Scene: "summary", ConsecutiveFailures: 3, LastStatus: "failed", LastFailedAt: recent})
	insertAlertState(t, db, State{ProviderID: "p-stale-old", Scene: "title", ConsecutiveFailures: 5, LastStatus: "failed", LastFailedAt: old})
	insertAlertState(t, db, State{ProviderID: "p-below", Scene: "flowchart", ConsecutiveFailures: 1, LastStatus: "failed", LastFailedAt: recent})
	insertAlertState(t, db, State{ProviderID: "p-recovered", Scene: "summary", ConsecutiveFailures: 0, LastStatus: "success", LastRecoveredAt: recovered})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}

	cases := map[string]AlertStatus{
		"p-active":    StatusActive,
		"p-stale-old": StatusStale,
		"p-below":     StatusNormal,
		"p-recovered": StatusRecovered,
	}
	for id, want := range cases {
		item, ok := findItem(overview.Items, id)
		if !ok {
			t.Fatalf("item %q missing", id)
		}
		if item.AlertStatus != want {
			t.Fatalf("item %q status = %q, want %q (reason=%q)", id, item.AlertStatus, want, item.StatusReason)
		}
	}

	if overview.ActiveAlertCount != 1 {
		t.Fatalf("ActiveAlertCount = %d, want 1", overview.ActiveAlertCount)
	}
	if overview.StaleAlertCount != 1 {
		t.Fatalf("StaleAlertCount = %d, want 1", overview.StaleAlertCount)
	}
	if overview.RecoveredCount != 1 {
		t.Fatalf("RecoveredCount = %d, want 1", overview.RecoveredCount)
	}
	if overview.ReviewAlertCount != 1 {
		t.Fatalf("ReviewAlertCount = %d, want 1", overview.ReviewAlertCount)
	}

	active, _ := findItem(overview.Items, "p-active")
	if active.ActiveUntil == "" {
		t.Fatal("active item ActiveUntil should be set")
	}
	if !active.CanMute || !active.CanArchive {
		t.Fatalf("active item should allow mute+archive, got mute=%v archive=%v", active.CanMute, active.CanArchive)
	}
}

func TestOverviewResolverDowngradesDisabledProvider(t *testing.T) {
	t.Parallel()

	db := openAlertTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo, staticConfigProvider{Config: lifecycleConfig()}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	recent := time.Now().UTC().Add(-10 * time.Minute).Format(time.RFC3339)
	insertAlertState(t, db, State{ProviderID: "p-on", Scene: "summary", ConsecutiveFailures: 4, LastStatus: "failed", LastFailedAt: recent})
	insertAlertState(t, db, State{ProviderID: "p-off", Scene: "summary", ConsecutiveFailures: 4, LastStatus: "failed", LastFailedAt: recent})
	insertAlertState(t, db, State{ProviderID: "p-deleted", Scene: "summary", ConsecutiveFailures: 4, LastStatus: "failed", LastFailedAt: recent})

	service.SetProviderStatusResolver(fakeResolver{statuses: map[string]ProviderRuntimeStatus{
		"p-on":  {Enabled: true, InEffectiveRoute: true, ProviderName: "在线主节点", Model: "gpt-x"},
		"p-off": {Enabled: false, InEffectiveRoute: false},
		// p-deleted 不在 map 中，视为已删除。
	}})

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}

	on, _ := findItem(overview.Items, "p-on")
	if on.AlertStatus != StatusActive {
		t.Fatalf("p-on status = %q, want active", on.AlertStatus)
	}
	if on.ProviderName != "在线主节点" || on.Model != "gpt-x" {
		t.Fatalf("p-on should adopt runtime name/model, got %q/%q", on.ProviderName, on.Model)
	}

	off, _ := findItem(overview.Items, "p-off")
	if off.AlertStatus != StatusStale {
		t.Fatalf("p-off status = %q, want stale", off.AlertStatus)
	}

	deleted, _ := findItem(overview.Items, "p-deleted")
	if deleted.AlertStatus != StatusStale {
		t.Fatalf("p-deleted status = %q, want stale", deleted.AlertStatus)
	}
	if deleted.CanRetest {
		t.Fatal("deleted provider should not be retestable")
	}
	if !deleted.CanArchive {
		t.Fatal("deleted provider should still be archivable")
	}
}

func TestArchiveMuteUnmuteLifecycle(t *testing.T) {
	t.Parallel()

	db := openAlertTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo, staticConfigProvider{Config: lifecycleConfig()}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	recent := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	insertAlertState(t, db, State{ProviderID: "p1", Scene: "summary", ConsecutiveFailures: 3, LastStatus: "failed", LastFailedAt: recent})

	// 归档。
	res, err := service.Archive(context.Background(), "p1", "admin", "节点已下线")
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}
	item, _ := findItem(res.Overview.Items, "p1")
	if item.AlertStatus != StatusArchived {
		t.Fatalf("after archive status = %q, want archived", item.AlertStatus)
	}
	if res.Overview.ActiveAlertCount != 0 {
		t.Fatalf("after archive ActiveAlertCount = %d, want 0", res.Overview.ActiveAlertCount)
	}

	// 新失败自动解除归档并重新计数为 active。
	service.RecordFailure(context.Background(), Event{Scene: "summary", ProviderID: "p1", ErrorType: "timeout", OccurredAt: recent})
	overview, _ := service.Overview(context.Background())
	item, _ = findItem(overview.Items, "p1")
	if item.AlertStatus != StatusActive {
		t.Fatalf("after new failure status = %q, want active", item.AlertStatus)
	}
	if item.ArchivedAt != "" {
		t.Fatalf("archived_at should be cleared, got %q", item.ArchivedAt)
	}

	// 静默 -> muted。
	res, err = service.Mute(context.Background(), "p1", "admin", 24, "上游限流")
	if err != nil {
		t.Fatalf("Mute() error = %v", err)
	}
	item, _ = findItem(res.Overview.Items, "p1")
	if item.AlertStatus != StatusMuted {
		t.Fatalf("after mute status = %q, want muted", item.AlertStatus)
	}
	if !item.CanUnmute {
		t.Fatal("muted item should allow unmute")
	}
	if res.Overview.MutedAlertCount != 1 || res.Overview.ActiveAlertCount != 0 {
		t.Fatalf("after mute counts muted=%d active=%d, want 1/0", res.Overview.MutedAlertCount, res.Overview.ActiveAlertCount)
	}

	// 解除静默 -> 回到 active。
	res, err = service.Unmute(context.Background(), "p1", "admin")
	if err != nil {
		t.Fatalf("Unmute() error = %v", err)
	}
	item, _ = findItem(res.Overview.Items, "p1")
	if item.AlertStatus != StatusActive {
		t.Fatalf("after unmute status = %q, want active", item.AlertStatus)
	}
}

func TestRetestRecoversAndFails(t *testing.T) {
	t.Parallel()

	db := openAlertTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo, staticConfigProvider{Config: lifecycleConfig()}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	recent := time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339)
	insertAlertState(t, db, State{ProviderID: "p1", FailureStreakID: "streak-p1", Scene: "summary", ConsecutiveFailures: 4, LastStatus: "failed", LastFailedAt: recent})

	// 复测成功 -> recovered。
	service.SetProviderRetester(&fakeRetester{outcome: ProviderRetestOutcome{OK: true, RequestID: "rq-ok"}, found: true})
	res, err := service.Retest(context.Background(), "p1", "admin")
	if err != nil {
		t.Fatalf("Retest() error = %v", err)
	}
	if !res.OK {
		t.Fatalf("retest ok = false, want true (%s)", res.Message)
	}
	item, _ := findItem(res.Overview.Items, "p1")
	if item.AlertStatus != StatusRecovered {
		t.Fatalf("after retest success status = %q, want recovered", item.AlertStatus)
	}
	state, _, _ := repo.GetState(context.Background(), "p1")
	if state.ConsecutiveFailures != 0 {
		t.Fatalf("after retest success consecutive = %d, want 0", state.ConsecutiveFailures)
	}
	if state.FailureStreakID != "" {
		t.Fatalf("after retest success failure streak = %q, want empty", state.FailureStreakID)
	}

	// 再造一个失败节点，复测失败 -> 计数不变，仍 active。
	insertAlertState(t, db, State{ProviderID: "p2", Scene: "title", ConsecutiveFailures: 3, LastStatus: "failed", LastFailedAt: recent})
	service.SetProviderRetester(&fakeRetester{outcome: ProviderRetestOutcome{OK: false, ErrorType: "timeout", ErrorMessage: "still timing out"}, found: true})
	res, err = service.Retest(context.Background(), "p2", "admin")
	if err != nil {
		t.Fatalf("Retest() error = %v", err)
	}
	if res.OK {
		t.Fatal("retest ok = true, want false")
	}
	state, _, _ = repo.GetState(context.Background(), "p2")
	if state.ConsecutiveFailures != 3 {
		t.Fatalf("after retest failure consecutive = %d, want 3 (unchanged)", state.ConsecutiveFailures)
	}

	// 已删除节点。
	insertAlertState(t, db, State{ProviderID: "p3", Scene: "flowchart", ConsecutiveFailures: 3, LastStatus: "failed", LastFailedAt: recent})
	service.SetProviderRetester(&fakeRetester{found: false})
	res, err = service.Retest(context.Background(), "p3", "admin")
	if err != nil {
		t.Fatalf("Retest() error = %v", err)
	}
	if res.OK {
		t.Fatal("retest of deleted provider ok = true, want false")
	}
}

func TestPendingVerifyAfterConfigChange(t *testing.T) {
	t.Parallel()

	db := openAlertTestDB(t)
	repo := NewRepository(db)
	service := NewService(repo, staticConfigProvider{Config: lifecycleConfig()}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	past := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	insertAlertState(t, db, State{ProviderID: "p1", Scene: "summary", ConsecutiveFailures: 3, LastStatus: "failed", LastFailedAt: past})
	// 健康节点：不应因配置变更变黄。
	insertAlertState(t, db, State{ProviderID: "healthy", Scene: "summary", ConsecutiveFailures: 0, LastStatus: "success"})

	if err := service.NoteSceneConfigChanged(context.Background(), "admin", "summary"); err != nil {
		t.Fatalf("NoteSceneConfigChanged() error = %v", err)
	}

	overview, _ := service.Overview(context.Background())
	item, _ := findItem(overview.Items, "p1")
	if item.AlertStatus != StatusPendingVerify {
		t.Fatalf("after config change status = %q, want pending_verify", item.AlertStatus)
	}
	healthy, _ := findItem(overview.Items, "healthy")
	if healthy.AlertStatus == StatusPendingVerify {
		t.Fatal("healthy node must not become pending_verify on config change")
	}
}
