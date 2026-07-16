package invite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inviteRecord{}, fmt.Errorf("begin create invite tx: %w", err)
	}
	defer tx.Rollback()

	var membership int
	if err := tx.QueryRowContext(
		ctx,
		`SELECT 1 FROM kitchen_members WHERE kitchen_id = ? AND user_id = ? LIMIT 1`,
		params.KitchenID,
		params.InviterUserID,
	).Scan(&membership); errors.Is(err, sql.ErrNoRows) {
		return inviteRecord{}, common.ErrForbidden
	} else if err != nil {
		return inviteRecord{}, fmt.Errorf("check invite creator membership: %w", err)
	}

	_, err = tx.ExecContext(
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
	if err := tx.Commit(); err != nil {
		return inviteRecord{}, fmt.Errorf("commit create invite: %w", err)
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

	defer tx.Rollback()

	current, err := findInviteByIDTx(ctx, tx, invite.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return acceptInviteResult{}, common.ErrNotFound
	}
	if err != nil {
		return acceptInviteResult{}, err
	}

	var role string
	err = tx.QueryRowContext(
		ctx,
		`SELECT role FROM kitchen_members WHERE kitchen_id = ? AND user_id = ? LIMIT 1`,
		current.KitchenID,
		userID,
	).Scan(&role)
	if err == nil {
		if commitErr := tx.Commit(); commitErr != nil {
			return acceptInviteResult{}, fmt.Errorf("commit accept invite tx: %w", commitErr)
		}
		return acceptInviteResult{
			Invite:        current,
			KitchenID:     current.KitchenID,
			KitchenName:   current.KitchenName,
			Role:          role,
			AlreadyMember: true,
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return acceptInviteResult{}, fmt.Errorf("check existing membership: %w", err)
	}

	if err := validateInviteAcceptance(current); err != nil {
		return acceptInviteResult{}, err
	}

	now := time.Now()
	nowValue := now.Format(time.RFC3339)
	usageResult, err := tx.ExecContext(
		ctx,
		`UPDATE kitchen_invites
		 SET used_count = used_count + 1,
		     status = CASE WHEN used_count + 1 >= max_uses THEN ? ELSE status END
		 WHERE id = ?
		   AND status = ?
		   AND used_count < max_uses
		   AND datetime(expires_at) > datetime(?)`,
		statusUsedUp,
		current.ID,
		statusActive,
		nowValue,
	)
	if err != nil {
		return acceptInviteResult{}, fmt.Errorf("update kitchen invite usage: %w", err)
	}
	affected, err := usageResult.RowsAffected()
	if err != nil {
		return acceptInviteResult{}, fmt.Errorf("read updated invite usage count: %w", err)
	}
	if affected == 0 {
		fresh, findErr := findInviteByIDTx(ctx, tx, current.ID)
		if findErr != nil {
			return acceptInviteResult{}, findErr
		}
		if validationErr := validateInviteAcceptance(fresh); validationErr != nil {
			return acceptInviteResult{}, validationErr
		}
		return acceptInviteResult{}, common.NewAppError(
			common.CodeConflict,
			"invite is no longer available",
			http.StatusConflict,
		)
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at) VALUES (?, ?, ?, ?)`,
		current.KitchenID,
		userID,
		"member",
		nowValue,
	); err != nil {
		return acceptInviteResult{}, fmt.Errorf("insert kitchen member: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE kitchens SET updated_at = ? WHERE id = ?`,
		nowValue,
		current.KitchenID,
	); err != nil {
		return acceptInviteResult{}, fmt.Errorf("touch kitchen updated_at: %w", err)
	}

	current.UsedCount++
	if current.UsedCount >= current.MaxUses {
		current.Status = statusUsedUp
	}

	if err := tx.Commit(); err != nil {
		return acceptInviteResult{}, fmt.Errorf("commit accept invite tx: %w", err)
	}

	return acceptInviteResult{
		Invite:      current,
		KitchenID:   current.KitchenID,
		KitchenName: current.KitchenName,
		Role:        "member",
	}, nil
}

func findInviteByIDTx(ctx context.Context, tx *sql.Tx, inviteID int64) (inviteRecord, error) {
	const query = findInviteBaseQuery + `
WHERE ki.id = ?
LIMIT 1
`
	var item inviteRecord
	err := tx.QueryRowContext(ctx, query, inviteID).Scan(
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
		return inviteRecord{}, fmt.Errorf("find invite by id in accept tx: %w", err)
	}
	return item, nil
}

func validateInviteAcceptance(record inviteRecord) error {
	status, err := effectiveStatus(record)
	if err != nil {
		return err
	}
	switch status {
	case statusExpired:
		return common.NewAppError(common.CodeConflict, "invite has expired", http.StatusConflict)
	case statusUsedUp:
		return common.NewAppError(common.CodeConflict, "invite has reached its usage limit", http.StatusConflict)
	case statusRevoked:
		return common.NewAppError(common.CodeConflict, "invite is no longer available", http.StatusConflict)
	default:
		return nil
	}
}
