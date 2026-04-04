package recipe

import (
	"context"
	"database/sql"
	"testing"

	kitchenpkg "github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	_ "modernc.org/sqlite"
)

func TestRepositoryUpdateStatusDoesNotTouchRecipeUpdatedAt(t *testing.T) {
	db := openRecipeStatusTestDB(t)
	defer db.Close()

	seedRecipeStatusTestData(t, db)

	repo := NewRepository(db)
	if err := repo.UpdateStatus(
		context.Background(),
		"rec_status_1",
		1,
		"done",
		9,
		"2026-04-04T13:40:00+08:00",
	); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	var gotStatus string
	var gotUpdatedBy int64
	var gotUpdatedAt string
	if err := db.QueryRow(`
SELECT status, updated_by, updated_at
FROM recipes
WHERE id = 'rec_status_1'
`).Scan(&gotStatus, &gotUpdatedBy, &gotUpdatedAt); err != nil {
		t.Fatalf("query updated recipe error = %v", err)
	}

	if got, want := gotStatus, "done"; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if got, want := gotUpdatedBy, int64(9); got != want {
		t.Fatalf("updated_by = %d, want %d", got, want)
	}
	if got, want := gotUpdatedAt, "2026-04-01T09:00:00+08:00"; got != want {
		t.Fatalf("updated_at = %q, want %q", got, want)
	}
}

func TestServiceUpdateStatusKeepsRecipeUpdatedAtInResponse(t *testing.T) {
	db := openRecipeStatusTestDB(t)
	defer db.Close()

	seedRecipeStatusTestData(t, db)

	service := NewService(ServiceOptions{
		Repo:           NewRepository(db),
		KitchenService: kitchenpkg.NewService(kitchenpkg.NewRepository(db)),
	})

	item, err := service.UpdateStatus(context.Background(), 7, "rec_status_1", "done")
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	if got, want := item.Status, "done"; got != want {
		t.Fatalf("Status = %q, want %q", got, want)
	}
	if got, want := item.UpdatedAt, "2026-04-01T09:00:00+08:00"; got != want {
		t.Fatalf("UpdatedAt = %q, want %q", got, want)
	}
}

func openRecipeStatusTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	if _, err := db.Exec(`
CREATE TABLE kitchens (
  id INTEGER PRIMARY KEY,
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
  role TEXT NOT NULL DEFAULT 'member',
  joined_at TEXT NOT NULL
);

CREATE TABLE recipes (
  id TEXT PRIMARY KEY,
  kitchen_id INTEGER NOT NULL DEFAULT 1,
  title TEXT NOT NULL DEFAULT '',
  ingredient TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  link TEXT NOT NULL DEFAULT '',
  image_url TEXT NOT NULL DEFAULT '',
  image_urls_json TEXT NOT NULL DEFAULT '[]',
  image_meta_json TEXT NOT NULL DEFAULT '[]',
  flowchart_image_url TEXT NOT NULL DEFAULT '',
  flowchart_updated_at TEXT,
  flowchart_source_hash TEXT NOT NULL DEFAULT '',
  flowchart_status TEXT NOT NULL DEFAULT '',
  flowchart_error TEXT NOT NULL DEFAULT '',
  flowchart_requested_at TEXT,
  flowchart_finished_at TEXT,
  meal_type TEXT NOT NULL DEFAULT 'main',
  status TEXT NOT NULL DEFAULT 'wishlist',
  note TEXT NOT NULL DEFAULT '',
  ingredients_json TEXT NOT NULL DEFAULT '{}',
  steps_json TEXT NOT NULL DEFAULT '[]',
  parse_status TEXT NOT NULL DEFAULT '',
  parse_source TEXT NOT NULL DEFAULT '',
  parse_error TEXT NOT NULL DEFAULT '',
  parse_requested_at TEXT NOT NULL DEFAULT '',
  parse_finished_at TEXT NOT NULL DEFAULT '',
  parsed_content_edited INTEGER NOT NULL DEFAULT 0,
  pinned_at TEXT NOT NULL DEFAULT '',
  created_by INTEGER NOT NULL DEFAULT 0,
  updated_by INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  deleted_at TEXT
);
`); err != nil {
		db.Close()
		t.Fatalf("create test tables error = %v", err)
	}

	return db
}

func seedRecipeStatusTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '联调试吃厨房', 7, '2026-04-01T08:00:00+08:00', '2026-04-01T09:00:00+08:00', 'custom');

INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
VALUES (1, 7, 'owner', '2026-04-01T08:00:00+08:00');

INSERT INTO recipes (
  id, kitchen_id, title, meal_type, status, created_by, updated_by, created_at, updated_at
) VALUES (
  'rec_status_1', 1, '番茄牛腩', 'main', 'wishlist', 7, 7, '2026-04-01T08:00:00+08:00', '2026-04-01T09:00:00+08:00'
);
`); err != nil {
		t.Fatalf("seed test data error = %v", err)
	}
}
