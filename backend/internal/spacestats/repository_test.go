package spacestats

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRepositoryGetStatsAggregatesSpaceSignals(t *testing.T) {
	db := openSpaceStatsTestDB(t)
	defer db.Close()
	seedSpaceStatsTestData(t, db)

	stats, err := NewRepository(db).GetStats(context.Background(), 1, "2026-06-01T00:00:00+08:00", "2026-06-26")
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if got, want := stats.Overview.RecipeTotal, 2; got != want {
		t.Fatalf("Overview.RecipeTotal = %d, want %d", got, want)
	}
	if got, want := stats.Overview.WishlistRecipeTotal, 1; got != want {
		t.Fatalf("Overview.WishlistRecipeTotal = %d, want %d", got, want)
	}
	if got, want := stats.Overview.WantPlaceTotal, 1; got != want {
		t.Fatalf("Overview.WantPlaceTotal = %d, want %d", got, want)
	}
	if got, want := stats.Recipes.DoneTrendTotal, 1; got != want {
		t.Fatalf("Recipes.DoneTrendTotal = %d, want %d", got, want)
	}
	if got, want := stats.Places.PricedPlaceTotal, 2; got != want {
		t.Fatalf("Places.PricedPlaceTotal = %d, want %d", got, want)
	}
	if got, want := stats.Places.AveragePriceAmountCents, int64(10400); got != want {
		t.Fatalf("Places.AveragePriceAmountCents = %d, want %d", got, want)
	}
	if got, want := stats.Places.RecentVisitedTotal, 1; got != want {
		t.Fatalf("Places.RecentVisitedTotal = %d, want %d", got, want)
	}
	if got, want := len(stats.Overview.TopRevisitPlaces), 1; got != want {
		t.Fatalf("len(Overview.TopRevisitPlaces) = %d, want %d", got, want)
	}
	if got, want := stats.Overview.TopRevisitPlaces[0].Name, "旺记碳烤肥牛"; got != want {
		t.Fatalf("TopRevisitPlaces[0].Name = %q, want %q", got, want)
	}
	if got, want := stats.MealPlans.SubmittedDays, 1; got != want {
		t.Fatalf("MealPlans.SubmittedDays = %d, want %d", got, want)
	}
	if got, want := stats.MealPlans.DraftDays, 1; got != want {
		t.Fatalf("MealPlans.DraftDays = %d, want %d", got, want)
	}
	if got, want := stats.Members.Total, 2; got != want {
		t.Fatalf("Members.Total = %d, want %d", got, want)
	}
	if got := len(stats.Trends.RecipeDone); got == 0 {
		t.Fatalf("Trends.RecipeDone is empty, want done event trend")
	}
	if got := len(stats.Actions); got == 0 {
		t.Fatalf("Actions is empty, want actionable suggestions")
	}
}

func openSpaceStatsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  openid TEXT NOT NULL,
  nickname TEXT NOT NULL DEFAULT '',
  avatar_url TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

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
  kitchen_id INTEGER NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  ingredient TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  link TEXT NOT NULL DEFAULT '',
  image_url TEXT NOT NULL DEFAULT '',
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
  note TEXT NOT NULL DEFAULT '',
  ingredients_json TEXT NOT NULL DEFAULT '{}',
  steps_json TEXT NOT NULL DEFAULT '[]',
  parse_status TEXT NOT NULL DEFAULT '',
  parse_source TEXT NOT NULL DEFAULT '',
  parse_error TEXT NOT NULL DEFAULT '',
  parse_requested_at TEXT NOT NULL DEFAULT '',
  parse_finished_at TEXT NOT NULL DEFAULT '',
  parsed_content_edited INTEGER NOT NULL DEFAULT 0,
  content_version INTEGER NOT NULL DEFAULT 0,
  parse_claim_token TEXT NOT NULL DEFAULT '',
  parse_claim_content_version INTEGER NOT NULL DEFAULT 0,
  parse_lease_expires_at TEXT NOT NULL DEFAULT '',
  flowchart_claim_token TEXT NOT NULL DEFAULT '',
  flowchart_claim_content_version INTEGER NOT NULL DEFAULT 0,
  flowchart_lease_expires_at TEXT NOT NULL DEFAULT '',
  pinned_at TEXT NOT NULL DEFAULT '',
  done_at TEXT NOT NULL DEFAULT '',
  created_by INTEGER NOT NULL DEFAULT 0,
  updated_by INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  deleted_at TEXT
);

CREATE TABLE places (
  id TEXT PRIMARY KEY,
  kitchen_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL DEFAULT 'food',
  address TEXT NOT NULL DEFAULT '',
  latitude REAL NOT NULL DEFAULT 0,
  longitude REAL NOT NULL DEFAULT 0,
  price TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT 'manual',
  source_url TEXT NOT NULL DEFAULT '',
  image_urls_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL DEFAULT 'want',
  tags_json TEXT NOT NULL DEFAULT '[]',
  note TEXT NOT NULL DEFAULT '',
  visited_at TEXT NOT NULL DEFAULT '',
  revisit_rating INTEGER NOT NULL DEFAULT 0,
  recommended_items_json TEXT NOT NULL DEFAULT '[]',
  price_amount_cents INTEGER NOT NULL DEFAULT 0,
  price_currency TEXT NOT NULL DEFAULT 'CNY',
  price_type TEXT NOT NULL DEFAULT '',
  phone TEXT NOT NULL DEFAULT '',
  external_provider TEXT NOT NULL DEFAULT '',
  external_poi_id TEXT NOT NULL DEFAULT '',
  rating TEXT NOT NULL DEFAULT '',
  dining_tips TEXT NOT NULL DEFAULT '',
  scenes_json TEXT NOT NULL DEFAULT '[]',
  best_time TEXT NOT NULL DEFAULT '',
  duration TEXT NOT NULL DEFAULT '',
  companion_tags_json TEXT NOT NULL DEFAULT '[]',
  parking_note TEXT NOT NULL DEFAULT '',
  created_by INTEGER NOT NULL DEFAULT 0,
  updated_by INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  deleted_at TEXT
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

CREATE TABLE place_status_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kitchen_id INTEGER NOT NULL,
  place_id TEXT NOT NULL,
  from_status TEXT NOT NULL DEFAULT '',
  to_status TEXT NOT NULL,
  changed_by INTEGER NOT NULL DEFAULT 0,
  changed_at TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'api'
);

CREATE TABLE meal_plans (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kitchen_id INTEGER NOT NULL,
  plan_date TEXT NOT NULL,
  status TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  created_by INTEGER NOT NULL,
  updated_by INTEGER NOT NULL,
  submitted_by INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  submitted_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE meal_plan_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  plan_id INTEGER NOT NULL,
  recipe_id TEXT NOT NULL,
  quantity INTEGER NOT NULL DEFAULT 1,
  meal_type_snapshot TEXT NOT NULL DEFAULT 'main',
  title_snapshot TEXT NOT NULL DEFAULT '',
  image_snapshot TEXT NOT NULL DEFAULT '',
  sort_index INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
`); err != nil {
		db.Close()
		t.Fatalf("create test tables error = %v", err)
	}

	return db
}

func seedSpaceStatsTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
INSERT INTO users (id, openid, nickname, avatar_url, created_at, updated_at)
VALUES
  (7, 'dev:alice', 'Alice', '', '2026-06-01T00:00:00+08:00', '2026-06-01T00:00:00+08:00'),
  (8, 'dev:bob', 'Bob', '', '2026-06-01T00:00:00+08:00', '2026-06-01T00:00:00+08:00');

INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '联调试吃空间', 7, '2026-06-01T00:00:00+08:00', '2026-06-26T00:00:00+08:00', 'custom');

INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
VALUES
  (1, 7, 'owner', '2026-06-01T00:00:00+08:00'),
  (1, 8, 'member', '2026-06-02T00:00:00+08:00');

INSERT INTO recipes (
  id, kitchen_id, title, image_url, image_urls_json, meal_type, status, ingredients_json, steps_json,
  parse_status, flowchart_status, done_at, created_by, updated_by, created_at, updated_at
) VALUES
  ('rec_wishlist', 1, '番茄滑蛋', '', '[]', 'main', 'wishlist', '{}', '[]', '', '', '', 7, 7, '2026-06-20T10:00:00+08:00', '2026-06-20T10:00:00+08:00'),
  ('rec_done', 1, '葱油拌面', 'https://cdn.example.com/noodle.jpg', '["https://cdn.example.com/noodle.jpg"]', 'breakfast', 'done', '{"mainIngredients":["面条"]}', '[{"title":"拌面"}]', 'done', 'done', '2026-06-21T12:00:00+08:00', 8, 8, '2026-05-20T10:00:00+08:00', '2026-06-21T12:00:00+08:00');

INSERT INTO recipe_status_events (kitchen_id, recipe_id, from_status, to_status, changed_by, changed_at, source)
VALUES
  (1, 'rec_done', 'wishlist', 'done', 8, '2026-06-21T12:00:00+08:00', 'api');

INSERT INTO places (
  id, kitchen_id, name, price, image_urls_json, status, tags_json, visited_at, revisit_rating,
  recommended_items_json, price_amount_cents, price_currency, price_type, scenes_json,
  external_provider, external_poi_id, created_by, updated_by, created_at, updated_at
) VALUES
  ('pla_want', 1, '周末咖啡', '¥88/人', '[]', 'want', '["咖啡"]', '', 0, '[]', 8800, 'CNY', 'per_person', '["下午茶"]', 'amap', 'poi-want', 7, 7, '2026-06-22T10:00:00+08:00', '2026-06-22T10:00:00+08:00'),
  ('pla_visit', 1, '旺记碳烤肥牛', '¥120/人', '["https://cdn.example.com/place.jpg"]', 'visited', '["聚餐"]', '2026-06-23T18:00:00+08:00', 5, '["碳烤肥牛"]', 12000, 'CNY', 'per_person', '["朋友小聚"]', 'amap', 'poi-visit', 8, 8, '2026-06-10T10:00:00+08:00', '2026-06-23T18:00:00+08:00');

INSERT INTO place_status_events (kitchen_id, place_id, from_status, to_status, changed_by, changed_at, source)
VALUES
  (1, 'pla_visit', 'want', 'visited', 8, '2026-06-23T18:00:00+08:00', 'api');

INSERT INTO meal_plans (id, kitchen_id, plan_date, status, note, created_by, updated_by, submitted_by, created_at, updated_at, submitted_at)
VALUES
  (1, 1, '2026-06-27', 'submitted', '', 7, 7, 7, '2026-06-24T10:00:00+08:00', '2026-06-24T10:00:00+08:00', '2026-06-24T10:00:00+08:00'),
  (2, 1, '2026-06-28', 'draft', '', 8, 8, 0, '2026-06-25T10:00:00+08:00', '2026-06-25T10:00:00+08:00', '');

INSERT INTO meal_plan_items (plan_id, recipe_id, quantity, meal_type_snapshot, title_snapshot, sort_index, created_at, updated_at)
VALUES
  (1, 'rec_done', 1, 'breakfast', '葱油拌面', 0, '2026-06-24T10:00:00+08:00', '2026-06-24T10:00:00+08:00'),
  (2, 'rec_wishlist', 1, 'main', '番茄滑蛋', 0, '2026-06-25T10:00:00+08:00', '2026-06-25T10:00:00+08:00');
`); err != nil {
		t.Fatalf("seed space stats data error = %v", err)
	}
}
