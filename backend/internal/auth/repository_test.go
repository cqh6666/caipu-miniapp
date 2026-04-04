package auth

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRepositoryEnsureProfileOnlyFillsMissingFields(t *testing.T) {
	db := openAuthTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	user := seedAuthUser(t, db, User{
		ID:        1,
		OpenID:    "wx-openid-1",
		Nickname:  "手动昵称",
		AvatarURL: "https://cdn.example.com/avatar-old.png",
	})

	got, err := repo.EnsureProfile(context.Background(), user, "微信昵称", "https://wx.qlogo.cn/mmopen/new-avatar")
	if err != nil {
		t.Fatalf("EnsureProfile() error = %v", err)
	}

	if got.Nickname != "手动昵称" {
		t.Fatalf("EnsureProfile() nickname = %q, want %q", got.Nickname, "手动昵称")
	}
	if got.AvatarURL != "https://cdn.example.com/avatar-old.png" {
		t.Fatalf("EnsureProfile() avatar = %q, want %q", got.AvatarURL, "https://cdn.example.com/avatar-old.png")
	}
}

func TestRepositoryUpdateProfileAllowsReplacingNicknameAndAvatar(t *testing.T) {
	db := openAuthTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	user := seedAuthUser(t, db, User{
		ID:        2,
		OpenID:    "wx-openid-2",
		Nickname:  "旧昵称",
		AvatarURL: "https://cdn.example.com/avatar-old.png",
	})

	got, err := repo.UpdateProfile(context.Background(), user, "新昵称", "https://cdn.example.com/avatar-new.png")
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}

	if got.Nickname != "新昵称" {
		t.Fatalf("UpdateProfile() nickname = %q, want %q", got.Nickname, "新昵称")
	}
	if got.AvatarURL != "https://cdn.example.com/avatar-new.png" {
		t.Fatalf("UpdateProfile() avatar = %q, want %q", got.AvatarURL, "https://cdn.example.com/avatar-new.png")
	}

	assertAuthUser(t, db, 2, "新昵称", "https://cdn.example.com/avatar-new.png")
}

func openAuthTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	if _, err := db.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  openid TEXT NOT NULL UNIQUE,
  nickname TEXT,
  avatar_url TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
`); err != nil {
		db.Close()
		t.Fatalf("create users table error = %v", err)
	}

	return db
}

func seedAuthUser(t *testing.T, db *sql.DB, user User) User {
	t.Helper()

	user.CreatedAt = "2026-04-04T00:00:00Z"
	user.UpdatedAt = "2026-04-04T00:00:00Z"

	if _, err := db.Exec(
		`INSERT INTO users (id, openid, nickname, avatar_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID,
		user.OpenID,
		nullableString(user.Nickname),
		nullableString(user.AvatarURL),
		user.CreatedAt,
		user.UpdatedAt,
	); err != nil {
		t.Fatalf("insert user error = %v", err)
	}

	return user
}

func assertAuthUser(t *testing.T, db *sql.DB, userID int64, wantNickname, wantAvatar string) {
	t.Helper()

	var gotNickname string
	var gotAvatar string
	if err := db.QueryRow(`SELECT COALESCE(nickname, ''), COALESCE(avatar_url, '') FROM users WHERE id = ?`, userID).Scan(&gotNickname, &gotAvatar); err != nil {
		t.Fatalf("query user %d error = %v", userID, err)
	}

	if gotNickname != wantNickname {
		t.Fatalf("user %d nickname = %q, want %q", userID, gotNickname, wantNickname)
	}
	if gotAvatar != wantAvatar {
		t.Fatalf("user %d avatar = %q, want %q", userID, gotAvatar, wantAvatar)
	}
}
