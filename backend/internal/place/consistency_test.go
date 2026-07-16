package place

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func TestPlaceVersionRejectsSecondClientUpdate(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()
	service := newPlaceTestService(db)
	ctx := context.Background()

	created, err := service.Create(ctx, 7, 1, placeRequest{Name: ptrString("原始地点")})
	if err != nil {
		t.Fatal(err)
	}
	first := created
	second := created
	first.Name = "客户端 A 保存的地点"
	first.UpdatedAt = "2026-07-16T01:10:01Z"
	updated, err := service.repo.Update(ctx, first)
	if err != nil {
		t.Fatalf("first Update() error = %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("updated version = %d, want 2", updated.Version)
	}

	version := second.Version
	_, err = service.Update(ctx, 7, second.ID, placeRequest{
		Version: &version,
		Name:    ptrString(second.Name),
		Note:    ptrString("客户端 B 的旧备注"),
	})
	var appErr *common.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusConflict || appErr.Code != common.CodeConflict {
		t.Fatalf("second update error = %T %v, want 409", err, err)
	}

	fresh, err := service.repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Name != first.Name || fresh.Note == "客户端 B 的旧备注" {
		t.Fatalf("fresh place = %#v", fresh)
	}
}

func TestPlaceWriteRechecksMembershipAfterServicePrecheck(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()
	service := newPlaceTestService(db)
	ctx := context.Background()

	created, err := service.Create(ctx, 7, 1, placeRequest{Name: ptrString("退出前地点")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM kitchen_members WHERE kitchen_id = 1 AND user_id = 7`); err != nil {
		t.Fatal(err)
	}
	created.Name = "退出后不应写入"
	created.UpdatedAt = "2026-07-16T01:10:02Z"
	_, err = service.repo.Update(ctx, created)
	if !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("Update() error = %v, want forbidden", err)
	}
}

func TestPlaceCreateAndDeleteRecheckMembership(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()
	service := newPlaceTestService(db)
	ctx := context.Background()
	created, err := service.Create(ctx, 7, 1, placeRequest{Name: ptrString("退出前地点")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM kitchen_members WHERE kitchen_id = 1 AND user_id = 7`); err != nil {
		t.Fatal(err)
	}

	_, err = service.repo.Create(ctx, Place{
		ID:        "pla_after_leave",
		KitchenID: 1,
		Name:      "退出后新增",
		Type:      TypeFood,
		Status:    StatusWant,
		CreatedBy: 7,
		UpdatedBy: 7,
		CreatedAt: "2026-07-16T01:10:03Z",
		UpdatedAt: "2026-07-16T01:10:03Z",
	})
	if !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("Create() error = %v, want forbidden", err)
	}
	if err := service.repo.Delete(ctx, created.ID, created.KitchenID, 7, "2026-07-16T01:10:04Z"); !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("Delete() error = %v, want forbidden", err)
	}
}
