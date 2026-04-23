package recipe

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	kitchenpkg "github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	_ "modernc.org/sqlite"
)

func openRecipeShareTokenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	// SQLite :memory: 每个连接独立，强制单连接确保所有并发请求走同一 DB 实例
	// 这恰好模拟「条件 UPDATE 串行化」场景：第二个 UPDATE 必然因 share_token 已非空而 affected=0
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  openid TEXT NOT NULL,
  nickname TEXT,
  avatar_url TEXT,
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
  joined_at TEXT NOT NULL,
  UNIQUE(kitchen_id, user_id)
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
  pinned_at TEXT NOT NULL DEFAULT '',
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

	if _, err := db.Exec(`
INSERT INTO users (id, openid, nickname, avatar_url, created_at, updated_at)
VALUES (7, 'openid_owner', '老张', '', '2026-04-01T08:00:00+08:00', '2026-04-01T08:00:00+08:00');

INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '联调试吃厨房', 7, '2026-04-01T08:00:00+08:00', '2026-04-01T09:00:00+08:00', 'custom');

INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
VALUES (1, 7, 'owner', '2026-04-01T08:00:00+08:00');

INSERT INTO recipes (
  id, kitchen_id, title, meal_type, status, note, created_by, updated_by, created_at, updated_at
) VALUES (
  'rec_share_1', 1, '番茄牛腩', 'main', 'wishlist', '私人备注：少放盐', 7, 7,
  '2026-04-01T08:00:00+08:00', '2026-04-01T09:00:00+08:00'
);
`); err != nil {
		t.Fatalf("seed share token test data error = %v", err)
	}

	return db
}

func newShareTokenTestService(db *sql.DB) *Service {
	return NewService(ServiceOptions{
		Repo:           NewRepository(db),
		KitchenService: kitchenpkg.NewService(kitchenpkg.NewRepository(db)),
	})
}

func TestEnsureShareTokenIsIdempotent(t *testing.T) {
	db := openRecipeShareTokenTestDB(t)
	defer db.Close()

	service := newShareTokenTestService(db)
	ctx := context.Background()

	first, err := service.EnsureShareToken(ctx, 7, "rec_share_1")
	if err != nil {
		t.Fatalf("first EnsureShareToken() error = %v", err)
	}
	if first == "" {
		t.Fatalf("first token is empty")
	}
	if got, want := len(first), shareTokenLength; got != want {
		t.Fatalf("first token length = %d, want %d", got, want)
	}

	second, err := service.EnsureShareToken(ctx, 7, "rec_share_1")
	if err != nil {
		t.Fatalf("second EnsureShareToken() error = %v", err)
	}
	if second != first {
		t.Fatalf("second token = %q, want %q (idempotent)", second, first)
	}
}

func TestEnsureShareTokenRejectsNonMember(t *testing.T) {
	db := openRecipeShareTokenTestDB(t)
	defer db.Close()

	service := newShareTokenTestService(db)
	if _, err := service.EnsureShareToken(context.Background(), 999, "rec_share_1"); err == nil {
		t.Fatalf("expected error for non-member, got nil")
	}
}

func TestGetByShareTokenReturnsKitchenAndCreator(t *testing.T) {
	db := openRecipeShareTokenTestDB(t)
	defer db.Close()

	service := newShareTokenTestService(db)
	ctx := context.Background()

	token, err := service.EnsureShareToken(ctx, 7, "rec_share_1")
	if err != nil {
		t.Fatalf("EnsureShareToken() error = %v", err)
	}

	view, err := service.GetByShareToken(ctx, token)
	if err != nil {
		t.Fatalf("GetByShareToken() error = %v", err)
	}

	if got, want := view.Recipe.ID, "rec_share_1"; got != want {
		t.Fatalf("Recipe.ID = %q, want %q", got, want)
	}
	if got, want := view.KitchenName, "联调试吃厨房"; got != want {
		t.Fatalf("KitchenName = %q, want %q", got, want)
	}
	if got, want := view.CreatorName, "老张"; got != want {
		t.Fatalf("CreatorName = %q, want %q", got, want)
	}
}

func TestGetByShareTokenReturnsNotFoundForUnknownToken(t *testing.T) {
	db := openRecipeShareTokenTestDB(t)
	defer db.Close()

	service := newShareTokenTestService(db)
	_, err := service.GetByShareToken(context.Background(), "nonexistent_token_22ch_xxxx")
	if !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// TestEnsureShareTokenConcurrentReturnsSameToken 验证 P1 修复：并发 EnsureShareToken
// 必须返回同一个 token，避免「先返回给前端的链接立刻失效」
func TestEnsureShareTokenConcurrentReturnsSameToken(t *testing.T) {
	db := openRecipeShareTokenTestDB(t)
	defer db.Close()

	service := newShareTokenTestService(db)
	ctx := context.Background()

	const goroutines = 16
	tokens := make([]string, goroutines)
	errs := make([]error, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			tk, err := service.EnsureShareToken(ctx, 7, "rec_share_1")
			tokens[i] = tk
			errs[i] = err
		}(i)
	}
	close(start)
	wg.Wait()

	first := tokens[0]
	for i, tk := range tokens {
		if errs[i] != nil {
			t.Fatalf("goroutine %d EnsureShareToken error = %v", i, errs[i])
		}
		if tk == "" {
			t.Fatalf("goroutine %d got empty token", i)
		}
		if tk != first {
			t.Fatalf("goroutine %d token = %q, want %q (all should match the winner)", i, tk, first)
		}
	}

	// 二次确认：库里的 share_token 与并发拿到的 token 一致
	persisted, err := service.repo.GetShareToken(ctx, "rec_share_1")
	if err != nil {
		t.Fatalf("GetShareToken after concurrent ensure: %v", err)
	}
	if persisted != first {
		t.Fatalf("persisted token = %q, want %q", persisted, first)
	}
}

// TestPublicRecipeViewExcludesPrivateFields 验证 Open Q1 修复：公开 DTO 必须剔除
// note 等私人字段，且把 Recipe 类型收窄为 PublicRecipe（编译期保证字段白名单）
func TestPublicRecipeViewExcludesPrivateFields(t *testing.T) {
	db := openRecipeShareTokenTestDB(t)
	defer db.Close()

	service := newShareTokenTestService(db)
	ctx := context.Background()

	token, err := service.EnsureShareToken(ctx, 7, "rec_share_1")
	if err != nil {
		t.Fatalf("EnsureShareToken() error = %v", err)
	}

	view, err := service.GetByShareToken(ctx, token)
	if err != nil {
		t.Fatalf("GetByShareToken() error = %v", err)
	}

	// PublicRecipe 是白名单 DTO，不应包含 note 字段
	// 此处通过 JSON 序列化间接验证：序列化结果不含 "note" key
	// （PublicRecipe 编译期就没有 Note 字段，但保留运行时断言以防回归）
	pub := view.Recipe
	if pub.Title != "番茄牛腩" {
		t.Fatalf("PublicRecipe.Title = %q, want 番茄牛腩", pub.Title)
	}
	// 通过 JSON marshal 确认敏感字段未泄漏
	jsonBytes, err := json.Marshal(pub)
	if err != nil {
		t.Fatalf("marshal PublicRecipe: %v", err)
	}
	body := string(jsonBytes)
	for _, banned := range []string{"\"note\"", "\"link\"", "\"createdBy\"", "\"shareToken\"", "\"flowchartProvider\"", "\"parseStatus\""} {
		if strings.Contains(body, banned) {
			t.Fatalf("PublicRecipe JSON 不应包含 %s，实际内容: %s", banned, body)
		}
	}
}
