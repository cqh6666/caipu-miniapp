package kitchen

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"

	_ "modernc.org/sqlite"
)

func TestRepositoryUpdateOwnedAutoNamesOnlyTouchesAutoKitchens(t *testing.T) {
	db := openKitchenTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES
  (1, '厨友青柠27的厨房', 7, '2026-03-22T00:00:00Z', '2026-03-22T00:00:00Z', 'auto'),
  (2, '周末聚餐厨房', 7, '2026-03-22T00:00:00Z', '2026-03-22T00:00:00Z', 'custom'),
  (3, '厨友青柠27的厨房', 9, '2026-03-22T00:00:00Z', '2026-03-22T00:00:00Z', 'auto');
`); err != nil {
		t.Fatalf("seed kitchens error = %v", err)
	}

	repo := NewRepository(db)
	if err := repo.UpdateOwnedAutoNames(context.Background(), 7, "小明的厨房"); err != nil {
		t.Fatalf("UpdateOwnedAutoNames() error = %v", err)
	}

	assertKitchenName(t, db, 1, "小明的厨房", nameSourceAuto)
	assertKitchenName(t, db, 2, "周末聚餐厨房", nameSourceCustom)
	assertKitchenName(t, db, 3, "厨友青柠27的厨房", nameSourceAuto)
}

func TestRepositoryUpdateNameMarksKitchenAsCustom(t *testing.T) {
	db := openKitchenTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '厨友青柠27的厨房', 7, '2026-03-22T00:00:00Z', '2026-03-22T00:00:00Z', 'auto');
`); err != nil {
		t.Fatalf("seed kitchens error = %v", err)
	}

	repo := NewRepository(db)
	if err := repo.UpdateName(context.Background(), 1, "夜宵小厨房"); err != nil {
		t.Fatalf("UpdateName() error = %v", err)
	}

	assertKitchenName(t, db, 1, "夜宵小厨房", nameSourceCustom)
}

func TestRepositoryLeaveDeletesMembershipAndRevokesOwnActiveInvites(t *testing.T) {
	db := openKitchenTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '周末聚餐空间', 7, '2026-03-22T00:00:00Z', '2026-03-22T00:00:00Z', 'custom');
INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
VALUES (1, 7, 'owner', '2026-03-22T00:00:00Z'),
       (1, 9, 'member', '2026-03-22T00:00:00Z');
INSERT INTO kitchen_invites (id, kitchen_id, inviter_user_id, status)
VALUES (1, 1, 9, 'active'),
       (2, 1, 9, 'used_up'),
       (3, 1, 7, 'active');
`); err != nil {
		t.Fatalf("seed leave data error = %v", err)
	}

	repo := NewRepository(db)
	if err := repo.Leave(context.Background(), 9, 1); err != nil {
		t.Fatalf("Leave() error = %v", err)
	}

	assertNoKitchenMember(t, db, 1, 9)
	assertInviteStatus(t, db, 1, "revoked")
	assertInviteStatus(t, db, 2, "used_up")
	assertInviteStatus(t, db, 3, "active")
}

func TestRepositoryLeaveRejectsOwner(t *testing.T) {
	db := openKitchenTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '我的空间', 7, '2026-03-22T00:00:00Z', '2026-03-22T00:00:00Z', 'custom');
INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
VALUES (1, 7, 'owner', '2026-03-22T00:00:00Z');
`); err != nil {
		t.Fatalf("seed owner data error = %v", err)
	}

	repo := NewRepository(db)
	err := repo.Leave(context.Background(), 7, 1)
	if err == nil {
		t.Fatal("expected owner leave to fail")
	}

	var appErr *common.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != common.CodeConflict {
		t.Fatalf("app error code = %d, want %d", appErr.Code, common.CodeConflict)
	}
	assertKitchenMemberRole(t, db, 1, 7, "owner")
}

func openKitchenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	if _, err := db.Exec(`
CREATE TABLE kitchens (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  owner_user_id INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  name_source TEXT NOT NULL DEFAULT 'custom'
);
CREATE TABLE kitchen_members (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kitchen_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  role TEXT NOT NULL,
  joined_at TEXT NOT NULL,
  UNIQUE(kitchen_id, user_id)
);
CREATE TABLE kitchen_invites (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kitchen_id INTEGER NOT NULL,
  inviter_user_id INTEGER NOT NULL,
  status TEXT NOT NULL
);
`); err != nil {
		db.Close()
		t.Fatalf("create kitchens table error = %v", err)
	}

	return db
}

func assertKitchenName(t *testing.T, db *sql.DB, kitchenID int64, wantName, wantSource string) {
	t.Helper()

	var gotName string
	var gotSource string
	if err := db.QueryRow(`SELECT name, name_source FROM kitchens WHERE id = ?`, kitchenID).Scan(&gotName, &gotSource); err != nil {
		t.Fatalf("query kitchen %d error = %v", kitchenID, err)
	}

	if gotName != wantName {
		t.Fatalf("kitchen %d name = %q, want %q", kitchenID, gotName, wantName)
	}
	if gotSource != wantSource {
		t.Fatalf("kitchen %d name_source = %q, want %q", kitchenID, gotSource, wantSource)
	}
}

func assertNoKitchenMember(t *testing.T, db *sql.DB, kitchenID, userID int64) {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM kitchen_members WHERE kitchen_id = ? AND user_id = ?`, kitchenID, userID).Scan(&count); err != nil {
		t.Fatalf("query kitchen member count error = %v", err)
	}
	if count != 0 {
		t.Fatalf("kitchen member count = %d, want 0", count)
	}
}

func assertKitchenMemberRole(t *testing.T, db *sql.DB, kitchenID, userID int64, wantRole string) {
	t.Helper()

	var gotRole string
	if err := db.QueryRow(`SELECT role FROM kitchen_members WHERE kitchen_id = ? AND user_id = ?`, kitchenID, userID).Scan(&gotRole); err != nil {
		t.Fatalf("query kitchen member role error = %v", err)
	}
	if gotRole != wantRole {
		t.Fatalf("kitchen member role = %q, want %q", gotRole, wantRole)
	}
}

func assertInviteStatus(t *testing.T, db *sql.DB, inviteID int64, wantStatus string) {
	t.Helper()

	var gotStatus string
	if err := db.QueryRow(`SELECT status FROM kitchen_invites WHERE id = ?`, inviteID).Scan(&gotStatus); err != nil {
		t.Fatalf("query invite status error = %v", err)
	}
	if gotStatus != wantStatus {
		t.Fatalf("invite %d status = %q, want %q", inviteID, gotStatus, wantStatus)
	}
}
