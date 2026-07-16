package recipe

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	kitchenpkg "github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
)

func TestRecipeVersionRejectsSecondClientUpdate(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()
	seedRecipeConsistencyData(t, db)

	repo := NewRepository(db)
	first, err := repo.FindByID(context.Background(), "rec_versioned")
	if err != nil {
		t.Fatal(err)
	}
	second := first
	first.Title = "客户端 A 保存的标题"
	first.UpdatedAt = "2026-07-16T01:00:01Z"
	updated, err := repo.Update(context.Background(), first)
	if err != nil {
		t.Fatalf("first Update() error = %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("updated version = %d, want 2", updated.Version)
	}

	version := second.Version
	service := NewService(ServiceOptions{
		Repo:           repo,
		KitchenService: kitchenpkg.NewService(kitchenpkg.NewRepository(db)),
	})
	_, err = service.Update(context.Background(), 7, second.ID, updateRecipeRequest{
		Version:       &version,
		Title:         second.Title,
		Ingredient:    second.Ingredient,
		Summary:       second.Summary,
		Link:          second.Link,
		ImageURLs:     second.ImageURLs,
		MealType:      second.MealType,
		Status:        second.Status,
		Note:          "客户端 B 的旧备注",
		ParsedContent: second.ParsedContent,
	})
	var appErr *common.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusConflict || appErr.Code != common.CodeConflict {
		t.Fatalf("second update error = %T %v, want 409", err, err)
	}

	fresh, err := repo.FindByID(context.Background(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Title != first.Title || fresh.Note == "客户端 B 的旧备注" {
		t.Fatalf("fresh recipe = %#v", fresh)
	}
}

func TestRecipeWriteRechecksMembershipAfterServicePrecheck(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()
	seedRecipeConsistencyData(t, db)

	repo := NewRepository(db)
	snapshot, err := repo.FindByID(context.Background(), "rec_versioned")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM kitchen_members WHERE kitchen_id = 1 AND user_id = 7`); err != nil {
		t.Fatal(err)
	}
	snapshot.Title = "退出后不应写入"
	snapshot.UpdatedAt = "2026-07-16T01:00:02Z"
	_, err = repo.Update(context.Background(), snapshot)
	if !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("Update() error = %v, want forbidden", err)
	}
}

func TestRecipeCreateAndDeleteRecheckMembership(t *testing.T) {
	db := openRecipeCreateTestDB(t)
	defer db.Close()
	seedRecipeConsistencyData(t, db)
	if _, err := db.Exec(`DELETE FROM kitchen_members WHERE kitchen_id = 1 AND user_id = 7`); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(db)
	_, err := repo.Create(context.Background(), Recipe{
		ID:        "rec_after_leave",
		KitchenID: 1,
		Title:     "退出后新增",
		MealType:  "main",
		Status:    "wishlist",
		CreatedBy: 7,
		UpdatedBy: 7,
		CreatedAt: "2026-07-16T01:00:03Z",
		UpdatedAt: "2026-07-16T01:00:03Z",
	})
	if !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("Create() error = %v, want forbidden", err)
	}
	if err := repo.SoftDelete(context.Background(), "rec_versioned", 1, 7, "2026-07-16T01:00:04Z"); !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("SoftDelete() error = %v, want forbidden", err)
	}
}

func seedRecipeConsistencyData(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '并发厨房', 7, '2026-07-16T01:00:00Z', '2026-07-16T01:00:00Z', 'custom');
INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
VALUES (1, 7, 'owner', '2026-07-16T01:00:00Z');
INSERT INTO recipes (
  id, kitchen_id, title, meal_type, status, ingredients_json, steps_json,
  created_by, updated_by, created_at, updated_at, version
) VALUES (
  'rec_versioned', 1, '原始标题', 'main', 'wishlist', '{}', '[]',
  7, 7, '2026-07-16T01:00:00Z', '2026-07-16T01:00:00Z', 1
);`); err != nil {
		t.Fatal(err)
	}
}
