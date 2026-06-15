package place

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	_ "modernc.org/sqlite"
)

func TestServiceCreateListSearchAndSoftDelete(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()

	service := newPlaceTestService(db)
	ctx := context.Background()

	created, err := service.Create(ctx, 7, 1, placeRequest{
		Name:      "Bites & Brews",
		Type:      TypeFood,
		Address:   "静安区武定路150号",
		Latitude:  31.2321,
		Longitude: 121.4432,
		Price:     "¥98/人",
		Source:    SourceManual,
		ImageURLs: []string{"https://cdn.example.com/place.jpg"},
		Status:    StatusWant,
		Tags:      []string{"早午餐", "氛围感"},
		Note:      "周末可去",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == "" {
		t.Fatalf("created.ID is empty")
	}
	if got, want := created.Status, StatusWant; got != want {
		t.Fatalf("created.Status = %q, want %q", got, want)
	}

	items, err := service.ListByKitchenID(ctx, 7, 1, ListFilter{Keyword: "武定"})
	if err != nil {
		t.Fatalf("ListByKitchenID() search error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("len(search items) = %d, want %d", got, want)
	}
	if got, want := items[0].Name, "Bites & Brews"; got != want {
		t.Fatalf("search item name = %q, want %q", got, want)
	}

	if err := service.Delete(ctx, 7, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	items, err = service.ListByKitchenID(ctx, 7, 1, ListFilter{})
	if err != nil {
		t.Fatalf("ListByKitchenID() after delete error = %v", err)
	}
	if got, want := len(items), 0; got != want {
		t.Fatalf("len(items after delete) = %d, want %d", got, want)
	}
}

func TestServiceUpdateStatusManagesVisitedAt(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()

	service := newPlaceTestService(db)
	ctx := context.Background()

	created, err := service.Create(ctx, 7, 1, placeRequest{
		Name:   "共青国家森林公园",
		Type:   TypeAttraction,
		Status: StatusWant,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.VisitedAt != "" {
		t.Fatalf("created.VisitedAt = %q, want empty", created.VisitedAt)
	}

	visited, err := service.UpdateStatus(ctx, 7, created.ID, StatusVisited)
	if err != nil {
		t.Fatalf("UpdateStatus(visited) error = %v", err)
	}
	if visited.VisitedAt == "" {
		t.Fatalf("visited.VisitedAt is empty")
	}

	want, err := service.UpdateStatus(ctx, 7, created.ID, StatusWant)
	if err != nil {
		t.Fatalf("UpdateStatus(want) error = %v", err)
	}
	if want.VisitedAt != "" {
		t.Fatalf("want.VisitedAt = %q, want empty", want.VisitedAt)
	}
}

func TestServiceRejectsInvalidInputAndNonMember(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()

	service := newPlaceTestService(db)
	ctx := context.Background()

	_, err := service.Create(ctx, 7, 1, placeRequest{Name: "", Status: StatusWant})
	var appErr *common.AppError
	if !errors.As(err, &appErr) || appErr.Code != common.CodeBadRequest {
		t.Fatalf("Create(empty name) error = %v, want bad request", err)
	}

	_, err = service.ListByKitchenID(ctx, 8, 1, ListFilter{})
	if !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("ListByKitchenID(non-member) error = %v, want forbidden", err)
	}
}

func newPlaceTestService(db *sql.DB) *Service {
	return NewService(NewRepository(db), kitchen.NewService(kitchen.NewRepository(db)))
}

func openPlaceTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  openid TEXT NOT NULL,
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
  role TEXT NOT NULL,
  joined_at TEXT NOT NULL
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
  created_by INTEGER NOT NULL,
  updated_by INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT
);

INSERT INTO users (id, openid, created_at, updated_at)
VALUES
  (7, 'dev:alice', '2026-06-15T00:00:00+08:00', '2026-06-15T00:00:00+08:00'),
  (8, 'dev:bob', '2026-06-15T00:00:00+08:00', '2026-06-15T00:00:00+08:00');

INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '联调试吃空间', 7, '2026-06-15T00:00:00+08:00', '2026-06-15T00:00:00+08:00', 'custom');

INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
VALUES (1, 7, 'owner', '2026-06-15T00:00:00+08:00');
`); err != nil {
		db.Close()
		t.Fatalf("create test tables error = %v", err)
	}

	return db
}
