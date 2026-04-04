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
	"github.com/cqh6666/caipu-miniapp/backend/internal/profile"
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
		return r.EnsureProfile(ctx, user, nickname, avatarURL)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return User{}, fmt.Errorf("find user by openid: %w", err)
	}

	nickname = strings.TrimSpace(nickname)
	if profile.IsPlaceholderNickname(nickname) {
		nickname = profile.FallbackNickname(0, openID)
	}
	avatarURL = strings.TrimSpace(avatarURL)
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
			return r.EnsureProfile(ctx, user, nickname, avatarURL)
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
		Nickname:  nickname,
		AvatarURL: avatarURL,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *Repository) EnsureProfile(ctx context.Context, user User, nickname, avatarURL string) (User, error) {
	nextNickname := strings.TrimSpace(user.Nickname)
	providedNickname := strings.TrimSpace(nickname)
	if !profile.IsPlaceholderNickname(providedNickname) && profile.IsPlaceholderNickname(nextNickname) {
		nextNickname = providedNickname
	}
	if profile.IsPlaceholderNickname(nextNickname) {
		nextNickname = profile.FallbackNickname(user.ID, user.OpenID)
	}

	nextAvatarURL := strings.TrimSpace(user.AvatarURL)
	providedAvatarURL := strings.TrimSpace(avatarURL)
	if nextAvatarURL == "" && providedAvatarURL != "" {
		nextAvatarURL = providedAvatarURL
	}

	return r.updateUserProfile(ctx, user, nextNickname, nextAvatarURL)
}

func (r *Repository) UpdateProfile(ctx context.Context, user User, nickname, avatarURL string) (User, error) {
	nextNickname := strings.TrimSpace(user.Nickname)
	providedNickname := strings.TrimSpace(nickname)
	if !profile.IsPlaceholderNickname(providedNickname) && providedNickname != nextNickname {
		nextNickname = providedNickname
	}
	if profile.IsPlaceholderNickname(nextNickname) {
		nextNickname = profile.FallbackNickname(user.ID, user.OpenID)
	}

	nextAvatarURL := strings.TrimSpace(user.AvatarURL)
	providedAvatarURL := strings.TrimSpace(avatarURL)
	if providedAvatarURL != "" && providedAvatarURL != nextAvatarURL {
		nextAvatarURL = providedAvatarURL
	}

	return r.updateUserProfile(ctx, user, nextNickname, nextAvatarURL)
}

func (r *Repository) updateUserProfile(ctx context.Context, user User, nextNickname, nextAvatarURL string) (User, error) {
	nextNickname = strings.TrimSpace(nextNickname)
	nextAvatarURL = strings.TrimSpace(nextAvatarURL)

	if nextNickname == strings.TrimSpace(user.Nickname) && nextAvatarURL == strings.TrimSpace(user.AvatarURL) {
		user.Nickname = nextNickname
		user.AvatarURL = nextAvatarURL
		return user, nil
	}

	now := time.Now().Format(time.RFC3339)
	if _, err := r.db.ExecContext(
		ctx,
		`UPDATE users SET nickname = ?, avatar_url = ?, updated_at = ? WHERE id = ?`,
		nullableString(nextNickname),
		nullableString(nextAvatarURL),
		now,
		user.ID,
	); err != nil {
		return User{}, fmt.Errorf("update user profile: %w", err)
	}

	user.Nickname = nextNickname
	user.AvatarURL = nextAvatarURL
	user.UpdatedAt = now
	return user, nil
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
