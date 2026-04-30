package dietassistant

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AddMessage(ctx context.Context, userID, kitchenID int64, role, content, createdAt string) error {
	if r == nil || r.db == nil {
		return nil
	}
	if err := insertMessage(ctx, r.db, userID, kitchenID, role, content, createdAt); err != nil {
		return fmt.Errorf("add diet assistant message: %w", err)
	}
	return nil
}

func (r *Repository) AddTurn(ctx context.Context, userID, kitchenID int64, userContent, assistantContent, createdAt string) error {
	if r == nil || r.db == nil {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin diet assistant turn transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := insertMessage(ctx, tx, userID, kitchenID, "user", userContent, createdAt); err != nil {
		return fmt.Errorf("add diet assistant user message: %w", err)
	}
	if err := insertMessage(ctx, tx, userID, kitchenID, "assistant", assistantContent, createdAt); err != nil {
		return fmt.Errorf("add diet assistant assistant message: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit diet assistant turn transaction: %w", err)
	}
	committed = true
	return nil
}

type sqlExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func insertMessage(ctx context.Context, execer sqlExecer, userID, kitchenID int64, role, content, createdAt string) error {
	_, err := execer.ExecContext(ctx, `
INSERT INTO diet_assistant_messages (
	user_id,
	kitchen_id,
	role,
	content,
	created_at
) VALUES (?, ?, ?, ?, ?)
`, userID, kitchenID, role, content, createdAt)
	return err
}

func (r *Repository) ListMessages(ctx context.Context, userID, kitchenID int64, limit int) ([]StoredMessage, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT id, role, content, created_at
FROM (
	SELECT id, role, content, created_at
	  FROM diet_assistant_messages
	 WHERE user_id = ? AND kitchen_id = ?
	 ORDER BY id DESC
	 LIMIT ?
)
ORDER BY id ASC
`, userID, kitchenID, limit)
	if err != nil {
		return nil, fmt.Errorf("list diet assistant messages: %w", err)
	}
	defer rows.Close()

	items := make([]StoredMessage, 0, limit)
	for rows.Next() {
		var item StoredMessage
		if err := rows.Scan(&item.ID, &item.Role, &item.Content, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan diet assistant message: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate diet assistant messages: %w", err)
	}
	return items, nil
}

func (r *Repository) ClearMessages(ctx context.Context, userID, kitchenID int64) error {
	if r == nil || r.db == nil {
		return nil
	}
	if _, err := r.db.ExecContext(ctx, `
DELETE FROM diet_assistant_messages
WHERE user_id = ? AND kitchen_id = ?
`, userID, kitchenID); err != nil {
		return fmt.Errorf("clear diet assistant messages: %w", err)
	}
	return nil
}
