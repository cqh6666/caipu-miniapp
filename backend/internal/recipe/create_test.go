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
	if err := db.QueryRow(`
SELECT updated_at, image_url
FROM recipes
WHERE id = ?
`, item.ID).Scan(&recipeUpdatedAt, &imageURL); err != nil {
		t.Fatalf("query created recipe error = %v", err)
	}
	if got, want := recipeUpdatedAt, item.UpdatedAt; got != want {
		t.Fatalf("recipe updated_at = %q, want %q", got, want)
	}
	if got, want := imageURL, "https://cdn.example.com/recipe-cover.jpg"; got != want {
		t.Fatalf("recipe image_url = %q, want %q", got, want)
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
  parsed_content_edited INTEGER NOT NULL DEFAULT 0,
  pinned_at TEXT,
  created_by INTEGER NOT NULL DEFAULT 0,
  updated_by INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  deleted_at TEXT,
  share_token TEXT NOT NULL DEFAULT '',
  share_token_created_at TEXT NOT NULL DEFAULT ''
);
`); err != nil {
		db.Close()
		t.Fatalf("create test tables error = %v", err)
	}

	return db
}
