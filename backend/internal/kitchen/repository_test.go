package kitchen

import (
	"context"
	"database/sql"
	"testing"

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
