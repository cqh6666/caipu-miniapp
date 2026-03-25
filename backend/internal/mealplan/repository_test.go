package mealplan

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRepositoryReplaceDraftAndListByKitchenID(t *testing.T) {
	db := openMealPlanTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	err := repo.ReplaceDraft(ctx, Plan{
		KitchenID: 1,
		PlanDate:  "2026-03-30",
		Status:    StatusDraft,
		Note:      "周一想吃清淡一点",
		CreatedBy: 7,
		UpdatedBy: 7,
		CreatedAt: "2026-03-26T10:00:00Z",
		UpdatedAt: "2026-03-26T10:00:00Z",
		Items: []Item{
			{RecipeID: "rec_a", Quantity: 1, MealTypeSnapshot: "main", TitleSnapshot: "番茄炒蛋"},
			{RecipeID: "rec_b", Quantity: 1, MealTypeSnapshot: "main", TitleSnapshot: "蒜蓉西兰花"},
		},
	}, "2026-03-26T10:00:00Z")
	if err != nil {
		t.Fatalf("ReplaceDraft() error = %v", err)
	}

	plans, err := repo.ListByKitchenID(ctx, 1)
	if err != nil {
		t.Fatalf("ListByKitchenID() error = %v", err)
	}
	if got, want := len(plans), 1; got != want {
		t.Fatalf("len(plans) = %d, want %d", got, want)
	}
	if got, want := plans[0].Status, StatusDraft; got != want {
		t.Fatalf("plans[0].Status = %q, want %q", got, want)
	}
	if got, want := len(plans[0].Items), 2; got != want {
		t.Fatalf("len(plans[0].Items) = %d, want %d", got, want)
	}
	if got, want := plans[0].Items[0].TitleSnapshot, "番茄炒蛋"; got != want {
		t.Fatalf("plans[0].Items[0].TitleSnapshot = %q, want %q", got, want)
	}
}

func TestRepositoryReplaceSubmittedClearsDraftAndReplacesExistingSubmitted(t *testing.T) {
	db := openMealPlanTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO meal_plans (id, kitchen_id, plan_date, status, note, created_by, updated_by, submitted_by, created_at, updated_at, submitted_at)
VALUES
  (1, 1, '2026-03-30', 'draft', '旧草稿', 7, 7, 0, '2026-03-25T00:00:00Z', '2026-03-25T00:00:00Z', ''),
  (2, 1, '2026-03-30', 'submitted', '旧已提交', 7, 7, 7, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z');
INSERT INTO meal_plan_items (plan_id, recipe_id, quantity, meal_type_snapshot, title_snapshot, image_snapshot, sort_index, created_at, updated_at)
VALUES
  (1, 'rec_old_draft', 1, 'main', '旧草稿菜', '', 0, '2026-03-25T00:00:00Z', '2026-03-25T00:00:00Z'),
  (2, 'rec_old_submitted', 1, 'main', '旧提交菜', '', 0, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z');
`); err != nil {
		t.Fatalf("seed meal plans error = %v", err)
	}

	repo := NewRepository(db)
	ctx := context.Background()

	err := repo.ReplaceSubmitted(ctx, Plan{
		KitchenID:   1,
		PlanDate:    "2026-03-30",
		Status:      StatusSubmitted,
		Note:        "新的提交备注",
		CreatedBy:   7,
		UpdatedBy:   7,
		SubmittedBy: 7,
		CreatedAt:   "2026-03-26T10:00:00Z",
		UpdatedAt:   "2026-03-26T10:00:00Z",
		SubmittedAt: "2026-03-26T10:00:00Z",
		Items: []Item{
			{RecipeID: "rec_new", Quantity: 1, MealTypeSnapshot: "main", TitleSnapshot: "新的菜单菜"},
		},
	}, "2026-03-26T10:00:00Z")
	if err != nil {
		t.Fatalf("ReplaceSubmitted() error = %v", err)
	}

	plans, err := repo.ListByKitchenID(ctx, 1)
	if err != nil {
		t.Fatalf("ListByKitchenID() error = %v", err)
	}
	if got, want := len(plans), 1; got != want {
		t.Fatalf("len(plans) = %d, want %d", got, want)
	}
	if got, want := plans[0].Status, StatusSubmitted; got != want {
		t.Fatalf("plans[0].Status = %q, want %q", got, want)
	}
	if got, want := plans[0].Note, "新的提交备注"; got != want {
		t.Fatalf("plans[0].Note = %q, want %q", got, want)
	}
	if got, want := len(plans[0].Items), 1; got != want {
		t.Fatalf("len(plans[0].Items) = %d, want %d", got, want)
	}
	if got, want := plans[0].Items[0].RecipeID, "rec_new"; got != want {
		t.Fatalf("plans[0].Items[0].RecipeID = %q, want %q", got, want)
	}
}

func TestRepositoryCountRecipesByKitchenID(t *testing.T) {
	db := openMealPlanTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO recipes (id, kitchen_id, title, meal_type, status, created_by, updated_by, created_at, updated_at)
VALUES
  ('rec_a', 1, '番茄炒蛋', 'main', 'wishlist', 7, 7, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z'),
  ('rec_b', 1, '蒜蓉西兰花', 'main', 'wishlist', 7, 7, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z'),
  ('rec_other', 2, '别的厨房', 'main', 'wishlist', 7, 7, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z');
`); err != nil {
		t.Fatalf("seed recipes error = %v", err)
	}

	repo := NewRepository(db)
	ctx := context.Background()

	count, err := repo.CountRecipesByKitchenID(ctx, 1, []string{"rec_a", "rec_b", "rec_other"})
	if err != nil {
		t.Fatalf("CountRecipesByKitchenID() error = %v", err)
	}
	if got, want := count, 2; got != want {
		t.Fatalf("CountRecipesByKitchenID() = %d, want %d", got, want)
	}
}

func openMealPlanTestDB(t *testing.T) *sql.DB {
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
  updated_at TEXT NOT NULL
);

CREATE TABLE recipes (
  id TEXT PRIMARY KEY,
  kitchen_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  meal_type TEXT NOT NULL,
  status TEXT NOT NULL,
  created_by INTEGER NOT NULL,
  updated_by INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT
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
  submitted_at TEXT NOT NULL DEFAULT '',
  UNIQUE(kitchen_id, plan_date, status)
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
  updated_at TEXT NOT NULL,
  UNIQUE(plan_id, recipe_id)
);

INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at)
VALUES
  (1, '测试厨房', 7, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z'),
  (2, '别的厨房', 8, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z');
`); err != nil {
		_ = db.Close()
		t.Fatalf("create meal plan tables error = %v", err)
	}

	return db
}
