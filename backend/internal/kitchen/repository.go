package kitchen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByUserID(ctx context.Context, userID int64) ([]Summary, error) {
	const query = `
SELECT k.id, k.name, km.role
FROM kitchen_members km
JOIN kitchens k ON k.id = km.kitchen_id
WHERE km.user_id = ?
ORDER BY k.updated_at DESC, k.id DESC
`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list kitchens by user: %w", err)
	}
	defer rows.Close()

	items := make([]Summary, 0)
	for rows.Next() {
		var item Summary
		if err := rows.Scan(&item.ID, &item.Name, &item.Role); err != nil {
			return nil, fmt.Errorf("scan kitchen: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate kitchens: %w", err)
	}

	return items, nil
}

func (r *Repository) CreateWithOwner(ctx context.Context, ownerUserID int64, name string) (Summary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Summary{}, fmt.Errorf("begin create kitchen tx: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO kitchens (name, owner_user_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		name,
		ownerUserID,
		now,
		now,
	)
	if err != nil {
		_ = tx.Rollback()
		return Summary{}, fmt.Errorf("insert kitchen: %w", err)
	}

	kitchenID, err := result.LastInsertId()
	if err != nil {
		_ = tx.Rollback()
		return Summary{}, fmt.Errorf("read kitchen id: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		kitchenID,
		ownerUserID,
		"owner",
		now,
	); err != nil {
		_ = tx.Rollback()
		return Summary{}, fmt.Errorf("insert kitchen member: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Summary{}, fmt.Errorf("commit create kitchen: %w", err)
	}

	return Summary{
		ID:   kitchenID,
		Name: name,
		Role: "owner",
	}, nil
}

func (r *Repository) HasMembership(ctx context.Context, userID, kitchenID int64) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM kitchen_members WHERE user_id = ? AND kitchen_id = ? LIMIT 1`,
		userID,
		kitchenID,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check kitchen membership: %w", err)
	}

	return true, nil
}
