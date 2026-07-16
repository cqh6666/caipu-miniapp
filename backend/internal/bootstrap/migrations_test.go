package bootstrap

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	if err := CheckMigrationsCurrent(context.Background(), db, dir); err != nil {
		t.Fatalf("migrations should be current: %v", err)
	}
}

func TestCheckMigrationsCurrentRejectsPendingMigration(t *testing.T) {
	db := openMigrationTestDB(t)
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_create.sql", `CREATE TABLE sample_items (id INTEGER PRIMARY KEY);`)
	if err := RunMigrations(context.Background(), db, slog.New(slog.NewTextHandler(io.Discard, nil)), dir); err != nil {
		t.Fatal(err)
	}
	writeMigrationFile(t, dir, "002_pending.sql", `ALTER TABLE sample_items ADD COLUMN name TEXT;`)

	err := CheckMigrationsCurrent(context.Background(), db, dir)
	if err == nil || !strings.Contains(err.Error(), "002_pending.sql") {
		t.Fatalf("expected pending migration error, got %v", err)
	}
}

func TestCheckMigrationsCurrentAllowsNewerAppliedRowsForBinaryRollback(t *testing.T) {
	db := openMigrationTestDB(t)
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_create.sql", `CREATE TABLE sample_items (id INTEGER PRIMARY KEY);`)
	if err := RunMigrations(context.Background(), db, slog.New(slog.NewTextHandler(io.Discard, nil)), dir); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (filename, applied_at) VALUES ('002_newer.sql', '2026-07-16T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}

	if err := CheckMigrationsCurrent(context.Background(), db, dir); err != nil {
		t.Fatalf("older forward-compatible release should remain ready: %v", err)
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

func TestRunMigrationsRejectsModifiedAppliedFile(t *testing.T) {
	db := openMigrationTestDB(t)
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_create.sql", `CREATE TABLE sample_items (id INTEGER PRIMARY KEY);`)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := RunMigrations(context.Background(), db, logger, dir); err != nil {
		t.Fatal(err)
	}
	writeMigrationFile(t, dir, "001_create.sql", `CREATE TABLE sample_items (id INTEGER PRIMARY KEY, name TEXT);`)

	err := RunMigrations(context.Background(), db, logger, dir)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("RunMigrations() error = %v, want checksum mismatch", err)
	}
	if err := CheckMigrationsCurrent(context.Background(), db, dir); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("CheckMigrationsCurrent() error = %v, want checksum mismatch", err)
	}
}

func TestRunMigrationsUpgradesLegacyMigrationTableAndBackfillsChecksum(t *testing.T) {
	db := openMigrationTestDB(t)
	dir := t.TempDir()
	content := `CREATE TABLE already_applied (id INTEGER PRIMARY KEY);`
	writeMigrationFile(t, dir, "001_create.sql", content)
	if _, err := db.Exec(`
CREATE TABLE schema_migrations (
	filename TEXT PRIMARY KEY,
	applied_at TEXT NOT NULL
);
INSERT INTO schema_migrations (filename, applied_at) VALUES ('001_create.sql', '2026-07-16T00:00:00Z');
`); err != nil {
		t.Fatal(err)
	}

	if err := RunMigrations(context.Background(), db, slog.New(slog.NewTextHandler(io.Discard, nil)), dir); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}
	digest := sha256.Sum256([]byte(content))
	want := hex.EncodeToString(digest[:])
	var checksum string
	if err := db.QueryRow(`SELECT checksum FROM schema_migrations WHERE filename = '001_create.sql'`).Scan(&checksum); err != nil {
		t.Fatal(err)
	}
	if checksum != want {
		t.Fatalf("checksum = %q, want %q", checksum, want)
	}
}

func TestMigrationFilesRejectDuplicateSequence(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "001_create.sql", `SELECT 1;`)
	writeMigrationFile(t, dir, "001_duplicate.sql", `SELECT 1;`)
	if _, err := migrationFiles(dir); err == nil || !strings.Contains(err.Error(), "duplicate migration sequence 001") {
		t.Fatalf("migrationFiles() error = %v, want duplicate sequence", err)
	}
}

func TestMigrationFilesAllowsDocumentedLegacy019Pair(t *testing.T) {
	dir := t.TempDir()
	writeMigrationFile(t, dir, "019_add_diet_assistant_messages.sql", `SELECT 1;`)
	writeMigrationFile(t, dir, "019_add_places.sql", `SELECT 1;`)
	files, err := migrationFiles(dir)
	if err != nil {
		t.Fatalf("migrationFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
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
