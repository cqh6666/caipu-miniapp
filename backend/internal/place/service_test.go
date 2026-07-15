package place

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
	_ "modernc.org/sqlite"
)

func TestServiceCreateListSearchAndSoftDelete(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()

	service := newPlaceTestService(db)
	ctx := context.Background()

	created, err := service.Create(ctx, 7, 1, placeRequest{
		Name:      ptrString("Bites & Brews"),
		Type:      ptrString(TypeFood),
		Address:   ptrString("静安区武定路150号"),
		Latitude:  ptrFloat64(31.2321),
		Longitude: ptrFloat64(121.4432),
		Price:     ptrString("¥98/人"),
		Source:    ptrString(SourceManual),
		ImageURLs: []string{"https://cdn.example.com/place.jpg"},
		Status:    ptrString(StatusWant),
		Tags:      []string{"早午餐", "氛围感"},
		Note:      ptrString("周末可去"),
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
	if got, want := created.PriceAmountCents, int64(9800); got != want {
		t.Fatalf("created.PriceAmountCents = %d, want %d", got, want)
	}
	if got, want := created.PriceType, "per_person"; got != want {
		t.Fatalf("created.PriceType = %q, want %q", got, want)
	}

	var eventToStatus string
	if err := db.QueryRow(`
SELECT to_status
FROM place_status_events
WHERE place_id = ?
`, created.ID).Scan(&eventToStatus); err != nil {
		t.Fatalf("query place status event error = %v", err)
	}
	if got, want := eventToStatus, StatusWant; got != want {
		t.Fatalf("place status event to_status = %q, want %q", got, want)
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
		Name:   ptrString("共青国家森林公园"),
		Type:   ptrString(TypeAttraction),
		Status: ptrString(StatusWant),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.VisitedAt != "" {
		t.Fatalf("created.VisitedAt = %q, want empty", created.VisitedAt)
	}

	visited, err := service.UpdateStatus(ctx, 7, created.ID, statusUpdateInput{Status: StatusVisited})
	if err != nil {
		t.Fatalf("UpdateStatus(visited) error = %v", err)
	}
	if visited.VisitedAt == "" {
		t.Fatalf("visited.VisitedAt is empty")
	}
	var fromStatus string
	var toStatus string
	if err := db.QueryRow(`
SELECT from_status, to_status
FROM place_status_events
WHERE place_id = ? AND to_status = ?
`, created.ID, StatusVisited).Scan(&fromStatus, &toStatus); err != nil {
		t.Fatalf("query visited status event error = %v", err)
	}
	if got, want := fromStatus, StatusWant; got != want {
		t.Fatalf("visited event from_status = %q, want %q", got, want)
	}
	if got, want := toStatus, StatusVisited; got != want {
		t.Fatalf("visited event to_status = %q, want %q", got, want)
	}

	want, err := service.UpdateStatus(ctx, 7, created.ID, statusUpdateInput{Status: StatusWant})
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

	_, err := service.Create(ctx, 7, 1, placeRequest{Name: ptrString(""), Status: ptrString(StatusWant)})
	var appErr *common.AppError
	if !errors.As(err, &appErr) || appErr.Code != common.CodeBadRequest {
		t.Fatalf("Create(empty name) error = %v, want bad request", err)
	}

	_, err = service.ListByKitchenID(ctx, 8, 1, ListFilter{})
	if !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("ListByKitchenID(non-member) error = %v, want forbidden", err)
	}
}

func TestServiceCreateMirrorsRemoteImages(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
			0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x04, 0x00, 0x00, 0x00, 0xb5, 0x1c, 0x0c,
			0x02, 0x00, 0x00, 0x00, 0x0b, 0x49, 0x44, 0x41,
			0x54, 0x78, 0xda, 0x63, 0xfc, 0xff, 0x1f, 0x00,
			0x03, 0x03, 0x02, 0x00, 0xef, 0x9a, 0x17, 0xdb,
			0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44,
			0xae, 0x42, 0x60, 0x82,
		})
	}))
	defer imageServer.Close()

	service := newPlaceTestService(db)
	service.SetUploadService(upload.NewServiceWithHTTPClient(t.TempDir(), "https://static.example.com/uploads", 10, imageServer.Client()))

	created, err := service.Create(context.Background(), 7, 1, placeRequest{
		Name:      ptrString("远程图片店"),
		Status:    ptrString(StatusWant),
		ImageURLs: []string{imageServer.URL + "/cover.png"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := len(created.ImageURLs); got != 1 {
		t.Fatalf("len(created.ImageURLs) = %d, want 1", got)
	}
	if got := created.ImageURLs[0]; !strings.HasPrefix(got, "https://static.example.com/uploads/") {
		t.Fatalf("created.ImageURLs[0] = %q, want mirrored uploads url", got)
	}
}

func TestUploadServiceRecognizesCaipuUploadsAsManaged(t *testing.T) {
	service := upload.NewService(t.TempDir(), "", 10)
	if !service.IsManagedImageURL("https://www.gxm1227.top/caipu-uploads/2026/06/img_test.jpg") {
		t.Fatal("expected /caipu-uploads/ url to be managed")
	}
}

func newPlaceTestService(db *sql.DB) *Service {
	return NewService(NewRepository(db), kitchen.NewService(kitchen.NewRepository(db)))
}

func ptrString(value string) *string {
	return &value
}

func ptrFloat64(value float64) *float64 {
	return &value
}

func ptrInt(value int) *int {
	return &value
}

func TestServicePartialUpdatePreservesExistingEnhancementFields(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()

	service := newPlaceTestService(db)
	ctx := context.Background()

	created, err := service.Create(ctx, 7, 1, placeRequest{
		Name:             ptrString("旺记碳烤肥牛"),
		Status:           ptrString(StatusVisited),
		Phone:            ptrString("17303028852"),
		ExternalProvider: ptrString(ExternalProviderAMap),
		ExternalPOIID:    ptrString("B0JUN7FVJK"),
		Rating:           ptrString("4.7"),
		RecommendedItems: []string{"碳烤肥牛", "烤鸡翅"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := service.Update(ctx, 7, created.ID, placeRequest{
		Name:  ptrString("旺记碳烤肥牛"),
		Phone: ptrString("17303028852"),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got, want := updated.ExternalProvider, ExternalProviderAMap; got != want {
		t.Fatalf("updated.ExternalProvider = %q, want %q", got, want)
	}
	if got, want := updated.ExternalPOIID, "B0JUN7FVJK"; got != want {
		t.Fatalf("updated.ExternalPOIID = %q, want %q", got, want)
	}
	if got, want := updated.Rating, "4.7"; got != want {
		t.Fatalf("updated.Rating = %q, want %q", got, want)
	}
	if got, want := len(updated.RecommendedItems), 2; got != want {
		t.Fatalf("len(updated.RecommendedItems) = %d, want %d", got, want)
	}
}

func TestServiceUpdateStatusStoresExperienceFields(t *testing.T) {
	db := openPlaceTestDB(t)
	defer db.Close()

	service := newPlaceTestService(db)
	ctx := context.Background()

	created, err := service.Create(ctx, 7, 1, placeRequest{
		Name:   ptrString("旺记碳烤肥牛"),
		Status: ptrString(StatusWant),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	visitedAt := "2026-06-25T18:30:00+08:00"
	visited, err := service.UpdateStatus(ctx, 7, created.ID, statusUpdateInput{
		Status:           StatusVisited,
		VisitedAt:        &visitedAt,
		RevisitRating:    ptrInt(5),
		RecommendedItems: []string{"碳烤肥牛", "烤鸡翅"},
	})
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if got, want := visited.VisitedAt, visitedAt; got != want {
		t.Fatalf("visited.VisitedAt = %q, want %q", got, want)
	}
	if got, want := visited.RevisitRating, 5; got != want {
		t.Fatalf("visited.RevisitRating = %d, want %d", got, want)
	}
	if got, want := len(visited.RecommendedItems), 2; got != want {
		t.Fatalf("len(visited.RecommendedItems) = %d, want %d", got, want)
	}
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
  created_by INTEGER NOT NULL,
  updated_by INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  deleted_at TEXT
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
