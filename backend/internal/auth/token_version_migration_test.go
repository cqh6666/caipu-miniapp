package auth

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "modernc.org/sqlite"
)

func TestTokenVersionMigrationBackfillsExistingUsers(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve migration test path")
	}
	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", "026_add_user_token_version.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatal(err)
	}
	database, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "migration.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if _, err := database.Exec(`
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  openid TEXT NOT NULL UNIQUE,
  nickname TEXT,
  avatar_url TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
INSERT INTO users (id, openid, nickname, created_at, updated_at)
VALUES (7, 'legacy-user', 'Legacy', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(string(migrationSQL)); err != nil {
		t.Fatalf("execute token version migration: %v", err)
	}

	var version int64
	if err := database.QueryRow(`SELECT token_version FROM users WHERE id = 7`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("legacy token version=%d, want=1", version)
	}
	if _, err := database.Exec(`UPDATE users SET token_version = 0 WHERE id = 7`); err == nil {
		t.Fatal("token_version CHECK constraint accepted zero")
	}
}
