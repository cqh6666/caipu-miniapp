package invite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const findInviteBaseQuery = `
SELECT ki.id,
       ki.kitchen_id,
       k.name,
       ki.inviter_user_id,
       COALESCE(u.nickname, ''),
       ki.token,
       COALESCE(ki.code, ''),
       ki.status,
       ki.max_uses,
       ki.used_count,
       ki.expires_at,
       ki.created_at
FROM kitchen_invites ki
JOIN kitchens k ON k.id = ki.kitchen_id
JOIN users u ON u.id = ki.inviter_user_id
`

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, params createInviteParams) (inviteRecord, error) {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO kitchen_invites (kitchen_id, inviter_user_id, token, code, status, max_uses, used_count, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		params.KitchenID,
		params.InviterUserID,
		params.Token,
		params.Code,
		params.Status,
		params.MaxUses,
		params.ExpiresAt,
		params.CreatedAt,
	)
	if err != nil {
		return inviteRecord{}, fmt.Errorf("insert kitchen invite: %w", err)
	}

	return r.FindByToken(ctx, params.Token)
}

func (r *Repository) FindByToken(ctx context.Context, token string) (inviteRecord, error) {
	const query = findInviteBaseQuery + `
WHERE ki.token = ?
LIMIT 1
`

	var item inviteRecord
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&item.ID,
		&item.KitchenID,
		&item.KitchenName,
		&item.InviterUserID,
		&item.InviterNickname,
		&item.Token,
		&item.Code,
		&item.Status,
		&item.MaxUses,
		&item.UsedCount,
		&item.ExpiresAt,
		&item.CreatedAt,
	)
	if err != nil {
		return inviteRecord{}, fmt.Errorf("find invite by token: %w", err)
	}

	return item, nil
}

func (r *Repository) FindByCode(ctx context.Context, code string) (inviteRecord, error) {
	const query = findInviteBaseQuery + `
WHERE ki.code = ?
LIMIT 1
`

	var item inviteRecord
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&item.ID,
		&item.KitchenID,
		&item.KitchenName,
		&item.InviterUserID,
		&item.InviterNickname,
		&item.Token,
		&item.Code,
		&item.Status,
		&item.MaxUses,
		&item.UsedCount,
		&item.ExpiresAt,
		&item.CreatedAt,
	)
	if err != nil {
		return inviteRecord{}, fmt.Errorf("find invite by code: %w", err)
	}

	return item, nil
}

func (r *Repository) Accept(ctx context.Context, userID int64, invite inviteRecord) (acceptInviteResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return acceptInviteResult{}, fmt.Errorf("begin accept invite tx: %w", err)
	}

	var role string
	err = tx.QueryRowContext(
		ctx,
		`SELECT role FROM kitchen_members WHERE kitchen_id = ? AND user_id = ? LIMIT 1`,
		invite.KitchenID,
		userID,
	).Scan(&role)
	if err == nil {
		if commitErr := tx.Commit(); commitErr != nil {
			return acceptInviteResult{}, fmt.Errorf("commit accept invite tx: %w", commitErr)
		}
		return acceptInviteResult{
			KitchenID:     invite.KitchenID,
			KitchenName:   invite.KitchenName,
			Role:          role,
			AlreadyMember: true,
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return acceptInviteResult{}, fmt.Errorf("check existing membership: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		invite.KitchenID,
		userID,
		"member",
		now,
	); err != nil {
		_ = tx.Rollback()
		return acceptInviteResult{}, fmt.Errorf("insert kitchen member: %w", err)
	}

	nextUsedCount := invite.UsedCount + 1
	nextStatus := invite.Status
	if nextUsedCount >= invite.MaxUses {
		nextStatus = statusUsedUp
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE kitchen_invites SET used_count = ?, status = ? WHERE id = ?`,
		nextUsedCount,
		nextStatus,
		invite.ID,
	); err != nil {
		_ = tx.Rollback()
		return acceptInviteResult{}, fmt.Errorf("update kitchen invite usage: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE kitchens SET updated_at = ? WHERE id = ?`,
		now,
		invite.KitchenID,
	); err != nil {
		_ = tx.Rollback()
		return acceptInviteResult{}, fmt.Errorf("touch kitchen updated_at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return acceptInviteResult{}, fmt.Errorf("commit accept invite tx: %w", err)
	}

	return acceptInviteResult{
		KitchenID:   invite.KitchenID,
		KitchenName: invite.KitchenName,
		Role:        "member",
	}, nil
}
