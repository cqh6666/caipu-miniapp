package invite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	appdb "github.com/cqh6666/caipu-miniapp/backend/internal/db"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
)

func TestAcceptInviteIsAtomicUnderConcurrency(t *testing.T) {
	database := openInviteMigrationTestDB(t)
	defer database.Close()

	now := time.Now().UTC()
	nowValue := now.Format(time.RFC3339)
	if _, err := database.Exec(
		`INSERT INTO users (id, openid, nickname, created_at, updated_at) VALUES (1, 'owner', 'Owner', ?, ?)`,
		nowValue, nowValue,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		`INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
		 VALUES (1, 'Atomic Kitchen', 1, ?, ?, 'custom')`,
		nowValue, nowValue,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		`INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at) VALUES (1, 1, 'owner', ?)`,
		nowValue,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(
		`INSERT INTO kitchen_invites (
		   id, kitchen_id, inviter_user_id, token, code, status, max_uses, used_count, expires_at, created_at
		 ) VALUES (1, 1, 1, 'atomic-token', 'ATOMIC01', 'active', 1, 0, ?, ?)`,
		now.Add(time.Hour).Format(time.RFC3339), nowValue,
	); err != nil {
		t.Fatal(err)
	}

	const contenders = 20
	for index := 0; index < contenders; index++ {
		userID := int64(index + 2)
		if _, err := database.Exec(
			`INSERT INTO users (id, openid, nickname, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			userID,
			fmt.Sprintf("contender-%d", userID),
			fmt.Sprintf("Contender %d", userID),
			nowValue,
			nowValue,
		); err != nil {
			t.Fatal(err)
		}
	}

	kitchenService := kitchen.NewService(kitchen.NewRepository(database))
	repository := NewRepository(database)
	service := NewService(repository, kitchenService, 72, 10, nil)
	staleRecord, err := repository.FindByToken(context.Background(), "atomic-token")
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		userID int64
		result acceptInviteResult
		err    error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, contenders)
	var wait sync.WaitGroup
	for index := 0; index < contenders; index++ {
		userID := int64(index + 2)
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			result, err := repository.Accept(context.Background(), userID, staleRecord)
			outcomes <- outcome{userID: userID, result: result, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(outcomes)

	var acceptedUserID int64
	successes := 0
	conflicts := 0
	conflictMessages := map[string]int{}
	for outcome := range outcomes {
		if outcome.err == nil {
			successes++
			acceptedUserID = outcome.userID
			if outcome.result.AlreadyMember || outcome.result.Invite.UsedCount != 1 {
				t.Fatalf("unexpected success result: %#v", outcome.result)
			}
			continue
		}
		var appErr *common.AppError
		if !errors.As(outcome.err, &appErr) || appErr.HTTPStatus != http.StatusConflict {
			t.Fatalf("user %d error=%#v, want conflict", outcome.userID, outcome.err)
		}
		conflicts++
		conflictMessages[appErr.Message]++
	}
	if successes != 1 || conflicts != contenders-1 {
		t.Fatalf("successes=%d conflicts=%d messages=%v", successes, conflicts, conflictMessages)
	}

	var usedCount int
	var status string
	if err := database.QueryRow(
		`SELECT used_count, status FROM kitchen_invites WHERE id = 1`,
	).Scan(&usedCount, &status); err != nil {
		t.Fatal(err)
	}
	if usedCount != 1 || status != statusUsedUp {
		t.Fatalf("invite used_count=%d status=%q", usedCount, status)
	}
	var newMemberCount int
	if err := database.QueryRow(
		`SELECT COUNT(1) FROM kitchen_members WHERE kitchen_id = 1 AND user_id <> 1`,
	).Scan(&newMemberCount); err != nil {
		t.Fatal(err)
	}
	if newMemberCount != 1 {
		t.Fatalf("new member count=%d, want 1", newMemberCount)
	}

	repeated, err := service.Accept(context.Background(), acceptedUserID, "atomic-token")
	if err != nil {
		t.Fatalf("idempotent accept error=%v", err)
	}
	if !repeated.AlreadyMember || repeated.Invite.UsedCount != 1 {
		t.Fatalf("idempotent result=%#v", repeated)
	}

	if _, err := database.Exec(
		`INSERT INTO users (id, openid, nickname, created_at, updated_at) VALUES (100, 'status-check', 'Status Check', ?, ?)`,
		nowValue, nowValue,
	); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		id        int
		token     string
		status    string
		expiresAt string
		message   string
	}{
		{id: 2, token: "expired-token", status: statusActive, expiresAt: now.Add(-time.Minute).Format(time.RFC3339), message: "invite has expired"},
		{id: 3, token: "revoked-token", status: statusRevoked, expiresAt: now.Add(time.Hour).Format(time.RFC3339), message: "invite is no longer available"},
	} {
		if _, err := database.Exec(
			`INSERT INTO kitchen_invites (
			   id, kitchen_id, inviter_user_id, token, code, status, max_uses, used_count, expires_at, created_at
			 ) VALUES (?, 1, 1, ?, ?, ?, 1, 0, ?, ?)`,
			test.id, test.token, fmt.Sprintf("STATUS%02d", test.id), test.status, test.expiresAt, nowValue,
		); err != nil {
			t.Fatal(err)
		}
		_, err := service.Accept(context.Background(), 100, test.token)
		var appErr *common.AppError
		if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusConflict || appErr.Message != test.message {
			t.Fatalf("token=%s error=%#v, want %q conflict", test.token, err, test.message)
		}
	}
	var rejectedMemberCount int
	if err := database.QueryRow(
		`SELECT COUNT(1) FROM kitchen_members WHERE kitchen_id = 1 AND user_id = 100`,
	).Scan(&rejectedMemberCount); err != nil {
		t.Fatal(err)
	}
	if rejectedMemberCount != 0 {
		t.Fatalf("rejected user membership count=%d", rejectedMemberCount)
	}

	foreignKeyRows, err := database.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer foreignKeyRows.Close()
	if foreignKeyRows.Next() {
		t.Fatal("foreign_key_check returned a violation")
	}
	if err := foreignKeyRows.Err(); err != nil {
		t.Fatalf("foreign_key_check error=%v", err)
	}
}

func openInviteMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	dir := t.TempDir()
	database, err := appdb.Open(config.Config{
		SQLitePath:          filepath.Join(dir, "invite.db"),
		SQLiteBusyTimeoutMS: 5000,
		UploadDir:           filepath.Join(dir, "uploads"),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	migrations := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations"))
	if err := bootstrap.RunMigrations(context.Background(), database, slog.New(slog.NewTextHandler(io.Discard, nil)), migrations); err != nil {
		database.Close()
		t.Fatal(err)
	}
	return database
}
