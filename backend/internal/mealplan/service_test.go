package mealplan

import (
	"context"
	"database/sql"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
)

func TestServiceCreateDraftFromSubmittedCopiesSubmittedPlan(t *testing.T) {
	db := openMealPlanTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO meal_plans (id, kitchen_id, plan_date, status, note, created_by, updated_by, submitted_by, created_at, updated_at, submitted_at)
VALUES
  (1, 1, '2026-03-30', 'submitted', '记得提前备菜', 7, 7, 7, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z');
INSERT INTO meal_plan_items (plan_id, recipe_id, quantity, meal_type_snapshot, title_snapshot, image_snapshot, sort_index, created_at, updated_at)
VALUES
  (1, 'rec_a', 1, 'main', '番茄炒蛋', '', 0, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z');
`); err != nil {
		t.Fatalf("seed meal plans error = %v", err)
	}

	service := newMealPlanServiceForTest(db)
	ctx := context.Background()

	store, err := service.CreateDraftFromSubmitted(ctx, 7, 1, "2026-03-30")
	if err != nil {
		t.Fatalf("CreateDraftFromSubmitted() error = %v", err)
	}

	draft, ok := store.Drafts["2026-03-30"]
	if !ok {
		t.Fatalf("draft not found after CreateDraftFromSubmitted()")
	}
	if got, want := draft.Note, "记得提前备菜"; got != want {
		t.Fatalf("draft.Note = %q, want %q", got, want)
	}
	if got, want := len(draft.Items), 1; got != want {
		t.Fatalf("len(draft.Items) = %d, want %d", got, want)
	}
	if got, want := len(store.Submitted), 1; got != want {
		t.Fatalf("len(store.Submitted) = %d, want %d", got, want)
	}
}

func TestServiceCreateDraftFromSubmittedKeepsExistingDraft(t *testing.T) {
	db := openMealPlanTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO meal_plans (id, kitchen_id, plan_date, status, note, created_by, updated_by, submitted_by, created_at, updated_at, submitted_at)
VALUES
  (1, 1, '2026-03-30', 'submitted', '旧安排', 7, 7, 7, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z'),
  (2, 1, '2026-03-30', 'draft', '用户已经改过一版', 7, 7, 0, '2026-03-25T00:00:00Z', '2026-03-25T00:00:00Z', '');
INSERT INTO meal_plan_items (plan_id, recipe_id, quantity, meal_type_snapshot, title_snapshot, image_snapshot, sort_index, created_at, updated_at)
VALUES
  (1, 'rec_a', 1, 'main', '番茄炒蛋', '', 0, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z'),
  (2, 'rec_b', 1, 'main', '蒜蓉西兰花', '', 0, '2026-03-25T00:00:00Z', '2026-03-25T00:00:00Z');
`); err != nil {
		t.Fatalf("seed meal plans error = %v", err)
	}

	service := newMealPlanServiceForTest(db)
	ctx := context.Background()

	store, err := service.CreateDraftFromSubmitted(ctx, 7, 1, "2026-03-30")
	if err != nil {
		t.Fatalf("CreateDraftFromSubmitted() error = %v", err)
	}

	draft, ok := store.Drafts["2026-03-30"]
	if !ok {
		t.Fatalf("draft not found after CreateDraftFromSubmitted()")
	}
	if got, want := draft.Note, "用户已经改过一版"; got != want {
		t.Fatalf("draft.Note = %q, want %q", got, want)
	}
	if got, want := draft.Items[0].RecipeID, "rec_b"; got != want {
		t.Fatalf("draft.Items[0].RecipeID = %q, want %q", got, want)
	}
}

func TestServiceCreateDraftFromSubmittedReplacesNoteOnlyDraftWithSubmittedItems(t *testing.T) {
	db := openMealPlanTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO meal_plans (id, kitchen_id, plan_date, status, note, created_by, updated_by, submitted_by, created_at, updated_at, submitted_at)
VALUES
  (1, 1, '2026-03-30', 'submitted', '正式安排', 7, 7, 7, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z'),
  (2, 1, '2026-03-30', 'draft', '先记一句备注', 7, 7, 0, '2026-03-25T00:00:00Z', '2026-03-25T00:00:00Z', '');
INSERT INTO meal_plan_items (plan_id, recipe_id, quantity, meal_type_snapshot, title_snapshot, image_snapshot, sort_index, created_at, updated_at)
VALUES
  (1, 'rec_a', 1, 'main', '番茄炒蛋', '', 0, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z');
`); err != nil {
		t.Fatalf("seed meal plans error = %v", err)
	}

	service := newMealPlanServiceForTest(db)
	ctx := context.Background()

	store, err := service.CreateDraftFromSubmitted(ctx, 7, 1, "2026-03-30")
	if err != nil {
		t.Fatalf("CreateDraftFromSubmitted() error = %v", err)
	}

	draft, ok := store.Drafts["2026-03-30"]
	if !ok {
		t.Fatalf("draft not found after CreateDraftFromSubmitted()")
	}
	if got, want := len(draft.Items), 1; got != want {
		t.Fatalf("len(draft.Items) = %d, want %d", got, want)
	}
	if got, want := draft.Items[0].RecipeID, "rec_a"; got != want {
		t.Fatalf("draft.Items[0].RecipeID = %q, want %q", got, want)
	}
	if got, want := draft.Note, "正式安排"; got != want {
		t.Fatalf("draft.Note = %q, want %q", got, want)
	}
}

func TestServiceDeleteSubmittedRemovesPlan(t *testing.T) {
	db := openMealPlanTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`
INSERT INTO meal_plans (id, kitchen_id, plan_date, status, note, created_by, updated_by, submitted_by, created_at, updated_at, submitted_at)
VALUES
  (1, 1, '2026-03-30', 'submitted', '要删掉的安排', 7, 7, 7, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z');
INSERT INTO meal_plan_items (plan_id, recipe_id, quantity, meal_type_snapshot, title_snapshot, image_snapshot, sort_index, created_at, updated_at)
VALUES
  (1, 'rec_a', 1, 'main', '番茄炒蛋', '', 0, '2026-03-24T00:00:00Z', '2026-03-24T00:00:00Z');
`); err != nil {
		t.Fatalf("seed meal plans error = %v", err)
	}

	service := newMealPlanServiceForTest(db)
	ctx := context.Background()

	store, err := service.DeleteSubmitted(ctx, 7, 1, "2026-03-30")
	if err != nil {
		t.Fatalf("DeleteSubmitted() error = %v", err)
	}
	if got := len(store.Submitted); got != 0 {
		t.Fatalf("len(store.Submitted) = %d, want 0", got)
	}
}

func newMealPlanServiceForTest(db *sql.DB) *Service {
	kitchenRepo := kitchen.NewRepository(db)
	kitchenService := kitchen.NewService(kitchenRepo)
	repo := NewRepository(db)
	return NewService(repo, kitchenService)
}
