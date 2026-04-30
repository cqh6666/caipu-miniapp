package dietassistant

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRepositoryListAndClearMessages(t *testing.T) {
	db := openDietAssistantTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	if err := repo.AddMessage(ctx, 1, 10, "user", "第一句", "2026-05-01T00:00:00Z"); err != nil {
		t.Fatalf("AddMessage user error = %v", err)
	}
	if err := repo.AddMessage(ctx, 1, 10, "assistant", "第一答", "2026-05-01T00:00:01Z"); err != nil {
		t.Fatalf("AddMessage assistant error = %v", err)
	}
	if err := repo.AddMessage(ctx, 2, 10, "user", "别人的消息", "2026-05-01T00:00:02Z"); err != nil {
		t.Fatalf("AddMessage other user error = %v", err)
	}
	if err := repo.AddTurn(ctx, 1, 20, "第二句", "第二答", "2026-05-01T00:00:03Z"); err != nil {
		t.Fatalf("AddTurn error = %v", err)
	}

	items, err := repo.ListMessages(ctx, 1, 10, 50)
	if err != nil {
		t.Fatalf("ListMessages error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Role != "user" || items[0].Content != "第一句" {
		t.Fatalf("first item = %#v", items[0])
	}
	if items[1].Role != "assistant" || items[1].Content != "第一答" {
		t.Fatalf("second item = %#v", items[1])
	}

	turnItems, err := repo.ListMessages(ctx, 1, 20, 50)
	if err != nil {
		t.Fatalf("ListMessages turn error = %v", err)
	}
	if len(turnItems) != 2 {
		t.Fatalf("len(turnItems) = %d, want 2", len(turnItems))
	}
	if turnItems[0].Role != "user" || turnItems[0].Content != "第二句" {
		t.Fatalf("turn user = %#v", turnItems[0])
	}
	if turnItems[1].Role != "assistant" || turnItems[1].Content != "第二答" {
		t.Fatalf("turn assistant = %#v", turnItems[1])
	}

	if err := repo.ClearMessages(ctx, 1, 10); err != nil {
		t.Fatalf("ClearMessages error = %v", err)
	}
	items, err = repo.ListMessages(ctx, 1, 10, 50)
	if err != nil {
		t.Fatalf("ListMessages after clear error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) after clear = %d, want 0", len(items))
	}
}

func openDietAssistantTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`
CREATE TABLE diet_assistant_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  kitchen_id INTEGER NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX idx_diet_assistant_messages_user_kitchen_id
  ON diet_assistant_messages(user_id, kitchen_id, id);
`); err != nil {
		_ = db.Close()
		t.Fatalf("setup diet assistant db error = %v", err)
	}
	return db
}
