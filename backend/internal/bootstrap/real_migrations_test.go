package bootstrap

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRealMigrationsApplyFromEmptyDatabase(t *testing.T) {
	db := openRealMigrationTestDB(t, "empty.db")
	dir := realMigrationDir(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := RunMigrations(context.Background(), db, logger, dir); err != nil {
		t.Fatalf("RunMigrations(empty) error = %v", err)
	}
	files, err := migrationFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	var applied, checksummed int
	if err := db.QueryRow(`SELECT COUNT(*), SUM(CASE WHEN checksum != '' THEN 1 ELSE 0 END) FROM schema_migrations`).Scan(&applied, &checksummed); err != nil {
		t.Fatal(err)
	}
	if applied != len(files) || checksummed != len(files) {
		t.Fatalf("applied/checksummed/files = %d/%d/%d", applied, checksummed, len(files))
	}
	if err := checkSQLiteIntegrity(context.Background(), db); err != nil {
		t.Fatalf("integrity after empty migration: %v", err)
	}
}

func TestRealMigrationsUpgradeRepresentativeHistoricalDatabase(t *testing.T) {
	db := openRealMigrationTestDB(t, "historical.db")
	fullDir := realMigrationDir(t)
	historicalDir := t.TempDir()
	copyMigrationsThrough(t, fullDir, historicalDir, 14)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := RunMigrations(context.Background(), db, logger, historicalDir); err != nil {
		t.Fatalf("RunMigrations(historical) error = %v", err)
	}
	if _, err := db.Exec(`
INSERT INTO app_runtime_settings (
	key, group_name, value_text, value_type, updated_by_subject, updated_at
) VALUES (
	'miniapp.features.diet_assistant_enabled', 'miniapp.features', 'true', 'bool', 'legacy-admin', '2026-07-01T00:00:00Z'
)
`); err != nil {
		t.Fatalf("seed historical runtime setting: %v", err)
	}

	if err := RunMigrations(context.Background(), db, logger, fullDir); err != nil {
		t.Fatalf("RunMigrations(upgrade) error = %v", err)
	}
	var version int
	if err := db.QueryRow(`SELECT version FROM app_runtime_setting_groups WHERE group_name = 'miniapp.features'`).Scan(&version); err != nil {
		t.Fatalf("query backfilled runtime group version: %v", err)
	}
	if version != 1 {
		t.Fatalf("runtime group version = %d, want 1", version)
	}
	var alertOutboxExists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'ai_provider_alert_deliveries'`).Scan(&alertOutboxExists); err != nil {
		t.Fatal(err)
	}
	if alertOutboxExists != 1 {
		t.Fatal("ai_provider_alert_deliveries table was not created")
	}
	if err := checkSQLiteIntegrity(context.Background(), db); err != nil {
		t.Fatalf("integrity after historical upgrade: %v", err)
	}
}

func openRealMigrationTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func realMigrationDir(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve migration test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations"))
}

func copyMigrationsThrough(t *testing.T, sourceDir, targetDir string, maxSequence int) {
	t.Helper()
	files, err := migrationFiles(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		sequenceText, _, _ := strings.Cut(name, "_")
		sequence, err := strconv.Atoi(sequenceText)
		if err != nil {
			t.Fatal(err)
		}
		if sequence > maxSequence {
			continue
		}
		content, err := os.ReadFile(filepath.Join(sourceDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, name), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
