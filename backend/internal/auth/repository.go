package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByID(ctx context.Context, userID int64) (User, error) {
	const query = `
SELECT id, openid, COALESCE(nickname, ''), COALESCE(avatar_url, ''), created_at, updated_at
FROM users
WHERE id = ?
LIMIT 1
`

	var user User
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
		&user.OpenID,
		&user.Nickname,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, common.ErrNotFound.WithErr(err)
	}
	if err != nil {
		return User{}, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

func (r *Repository) FindOrCreateByOpenID(ctx context.Context, openID, nickname, avatarURL string) (User, error) {
	openID = strings.TrimSpace(openID)
	if openID == "" {
		return User{}, common.NewAppError(common.CodeBadRequest, "openid is required", http.StatusBadRequest)
	}

	user, err := r.findByOpenID(ctx, openID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return User{}, fmt.Errorf("find user by openid: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	result, insertErr := r.db.ExecContext(
		ctx,
		`INSERT INTO users (openid, nickname, avatar_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		openID,
		nullableString(nickname),
		nullableString(avatarURL),
		now,
		now,
	)
	if insertErr != nil {
		user, retryErr := r.findByOpenID(ctx, openID)
		if retryErr == nil {
			return user, nil
		}
		return User{}, fmt.Errorf("create user: %w", insertErr)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("read user id: %w", err)
	}

	return User{
		ID:        userID,
		OpenID:    openID,
		Nickname:  strings.TrimSpace(nickname),
		AvatarURL: strings.TrimSpace(avatarURL),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *Repository) findByOpenID(ctx context.Context, openID string) (User, error) {
	const query = `
SELECT id, openid, COALESCE(nickname, ''), COALESCE(avatar_url, ''), created_at, updated_at
FROM users
WHERE openid = ?
LIMIT 1
`

	var user User
	err := r.db.QueryRowContext(ctx, query, openID).Scan(
		&user.ID,
		&user.OpenID,
		&user.Nickname,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
