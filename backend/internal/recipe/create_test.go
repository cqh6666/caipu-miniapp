package recipe

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRepositoryCreatePersistsRecipeAndBumpsKitchenUpdatedAt(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '联调试吃厨房', 7, '2026-04-01T08:00:00+08:00', '2026-04-01T09:00:00+08:00', 'custom');
INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
VALUES (1, 7, 'owner', '2026-04-01T08:00:00+08:00');
`); err != nil {
		t.Fatalf("seed kitchen error = %v", err)
	}

	repo := NewRepository(db)
	item := Recipe{
		ID:        "rec_create_1",
		KitchenID: 1,
		Title:     "番茄滑蛋",
		MealType:  "main",
		Status:    "wishlist",
		CreatedBy: 7,
		UpdatedBy: 7,
		CreatedAt: "2026-04-01T10:00:00+08:00",
		UpdatedAt: "2026-04-01T10:00:00+08:00",
		ImageURLs: []string{"https://cdn.example.com/recipe-cover.jpg"},
		ParsedContent: ParsedContent{
			MainIngredients:      []string{"番茄 2个", "鸡蛋 3个"},
			SecondaryIngredients: []string{"盐 适量"},
			Steps: []ParsedStep{
				{Title: "备料", Detail: "番茄切块，鸡蛋打散。"},
				{Title: "炒制", Detail: "先炒蛋后炒番茄，再回锅翻匀。"},
			},
		},
	}

	created, err := repo.Create(context.Background(), item)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got, want := created.ID, item.ID; got != want {
		t.Fatalf("Create() returned id = %q, want %q", got, want)
	}

	var recipeUpdatedAt string
	var imageURL string
	var doneAt string
	if err := db.QueryRow(`
SELECT updated_at, image_url, done_at
FROM recipes
WHERE id = ?
`, item.ID).Scan(&recipeUpdatedAt, &imageURL, &doneAt); err != nil {
		t.Fatalf("query created recipe error = %v", err)
	}
	if got, want := recipeUpdatedAt, item.UpdatedAt; got != want {
		t.Fatalf("recipe updated_at = %q, want %q", got, want)
	}
	if got, want := imageURL, "https://cdn.example.com/recipe-cover.jpg"; got != want {
		t.Fatalf("recipe image_url = %q, want %q", got, want)
	}
	if got := doneAt; got != "" {
		t.Fatalf("recipe done_at = %q, want empty", got)
	}

	var eventToStatus string
	var eventSource string
	if err := db.QueryRow(`
SELECT to_status, source
FROM recipe_status_events
WHERE recipe_id = ?
`, item.ID).Scan(&eventToStatus, &eventSource); err != nil {
		t.Fatalf("query recipe status event error = %v", err)
	}
	if got, want := eventToStatus, "wishlist"; got != want {
		t.Fatalf("recipe status event to_status = %q, want %q", got, want)
	}
	if got, want := eventSource, "api"; got != want {
		t.Fatalf("recipe status event source = %q, want %q", got, want)
	}

	var kitchenUpdatedAt string
	if err := db.QueryRow(`SELECT updated_at FROM kitchens WHERE id = 1`).Scan(&kitchenUpdatedAt); err != nil {
		t.Fatalf("query kitchen updated_at error = %v", err)
	}
	if got, want := kitchenUpdatedAt, item.UpdatedAt; got != want {
		t.Fatalf("kitchen updated_at = %q, want %q", got, want)
	}
}

func openRecipeCreateTestDB(t *testing.T) *sql.DB {
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

CREATE TABLE recipes (
  id TEXT PRIMARY KEY,
  kitchen_id INTEGER NOT NULL DEFAULT 1,
  title TEXT NOT NULL DEFAULT '',
  title_source TEXT NOT NULL DEFAULT 'manual',
  ingredient TEXT,
  summary TEXT NOT NULL DEFAULT '',
  link TEXT,
  image_url TEXT,
  image_urls_json TEXT NOT NULL DEFAULT '[]',
  image_meta_json TEXT NOT NULL DEFAULT '[]',
  flowchart_image_url TEXT NOT NULL DEFAULT '',
  flowchart_provider TEXT NOT NULL DEFAULT '',
  flowchart_model TEXT NOT NULL DEFAULT '',
  flowchart_updated_at TEXT,
  flowchart_source_hash TEXT NOT NULL DEFAULT '',
  flowchart_status TEXT NOT NULL DEFAULT '',
  flowchart_error TEXT NOT NULL DEFAULT '',
  flowchart_requested_at TEXT,
  flowchart_finished_at TEXT,
  meal_type TEXT NOT NULL DEFAULT 'main',
  status TEXT NOT NULL DEFAULT 'wishlist',
  note TEXT,
  ingredients_json TEXT NOT NULL DEFAULT '{}',
  steps_json TEXT NOT NULL DEFAULT '[]',
  parse_status TEXT NOT NULL DEFAULT '',
  parse_source TEXT NOT NULL DEFAULT '',
  parse_error TEXT NOT NULL DEFAULT '',
  parse_requested_at TEXT,
  parse_finished_at TEXT,
  parse_attempts INTEGER NOT NULL DEFAULT 0,
  parse_next_attempt_at TEXT NOT NULL DEFAULT '',
  parse_last_error_type TEXT NOT NULL DEFAULT '',
  parse_processing_started_at TEXT NOT NULL DEFAULT '',
  parsed_content_edited INTEGER NOT NULL DEFAULT 0,
  content_version INTEGER NOT NULL DEFAULT 0,
	version INTEGER NOT NULL DEFAULT 1,
  parse_claim_token TEXT NOT NULL DEFAULT '',
  parse_claim_content_version INTEGER NOT NULL DEFAULT 0,
  parse_lease_expires_at TEXT NOT NULL DEFAULT '',
  flowchart_claim_token TEXT NOT NULL DEFAULT '',
  flowchart_claim_content_version INTEGER NOT NULL DEFAULT 0,
  flowchart_lease_expires_at TEXT NOT NULL DEFAULT '',
  pinned_at TEXT,
  created_by INTEGER NOT NULL DEFAULT 0,
  updated_by INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  done_at TEXT NOT NULL DEFAULT '',
  deleted_at TEXT,
  share_token TEXT NOT NULL DEFAULT '',
  share_token_created_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE kitchen_members (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kitchen_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  role TEXT NOT NULL,
  joined_at TEXT NOT NULL,
  UNIQUE(kitchen_id, user_id)
);

CREATE TABLE recipe_status_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kitchen_id INTEGER NOT NULL,
  recipe_id TEXT NOT NULL,
  from_status TEXT NOT NULL DEFAULT '',
  to_status TEXT NOT NULL,
  changed_by INTEGER NOT NULL DEFAULT 0,
  changed_at TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'api'
);
`); err != nil {
		db.Close()
		t.Fatalf("create test tables error = %v", err)
	}

	return db
}
