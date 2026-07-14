package bootstrap

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunMigrationsAppliesSortedAndOnlyOnce(t *testing.T) {
	db := openMigrationTestDB(t)
	dir := t.TempDir()
	writeMigrationFile(t, dir, "002_insert.sql", `INSERT INTO sample_items (name) VALUES ('second');`)
	writeMigrationFile(t, dir, "001_create.sql", `CREATE TABLE sample_items (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`)
	writeMigrationFile(t, dir, "README.txt", "ignored")
	if err := os.Mkdir(filepath.Join(dir, "003_dir.sql"), 0o755); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := RunMigrations(context.Background(), db, logger, dir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if err := RunMigrations(context.Background(), db, logger, dir); err != nil {
		t.Fatalf("rerun migrations: %v", err)
	}

	var itemCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sample_items`).Scan(&itemCount); err != nil {
		t.Fatal(err)
	}
	if itemCount != 1 {
		t.Fatalf("migration should be idempotent, item count=%d", itemCount)
	}
	var migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	if migrationCount != 2 {
		t.Fatalf("unexpected migration count: %d", migrationCount)
	}
}

func TestRunMigrationsRollsBackFailedFile(t *testing.T) {
	db := openMigrationTestDB(t)
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_broken.sql", `
CREATE TABLE should_rollback (id INTEGER PRIMARY KEY);
INSERT INTO missing_table (id) VALUES (1);
`)

	err := RunMigrations(context.Background(), db, slog.New(slog.NewTextHandler(io.Discard, nil)), dir)
	if err == nil || !strings.Contains(err.Error(), "execute migration 001_broken.sql") {
		t.Fatalf("expected migration execution error, got %v", err)
	}

	var tableCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'should_rollback'`).Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if tableCount != 0 {
		t.Fatal("failed migration should roll back its DDL")
	}
	var appliedCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename = '001_broken.sql'`).Scan(&appliedCount); err != nil {
		t.Fatal(err)
	}
	if appliedCount != 0 {
		t.Fatal("failed migration should not be recorded")
	}
}

func TestRunMigrationsReportsMissingDirectory(t *testing.T) {
	db := openMigrationTestDB(t)
	err := RunMigrations(
		context.Background(),
		db,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		filepath.Join(t.TempDir(), "missing"),
	)
	if err == nil || !strings.Contains(err.Error(), "read migration dir") {
		t.Fatalf("expected missing directory error, got %v", err)
	}
}

func openMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "migration-test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func writeMigrationFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
