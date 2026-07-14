package db

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
)

func TestOpenCreatesDirectoriesAndAppliesPragmas(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		SQLitePath:          filepath.Join(dir, "database", "app.db"),
		SQLiteBusyTimeoutMS: 4321,
		UploadDir:           filepath.Join(dir, "uploads", "images"),
	}
	db, err := Open(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := os.Stat(cfg.SQLitePath); err != nil {
		t.Fatalf("sqlite file was not created: %v", err)
	}
	if info, err := os.Stat(cfg.UploadDir); err != nil || !info.IsDir() {
		t.Fatalf("upload directory was not created: info=%v err=%v", info, err)
	}
	if stats := db.Stats(); stats.MaxOpenConnections != 1 {
		t.Fatalf("unexpected max open connections: %d", stats.MaxOpenConnections)
	}

	var journalMode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if strings.ToLower(journalMode) != "wal" {
		t.Fatalf("unexpected journal mode: %q", journalMode)
	}
	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign keys should be enabled, got %d", foreignKeys)
	}
	var busyTimeout int
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if busyTimeout != cfg.SQLiteBusyTimeoutMS {
		t.Fatalf("unexpected busy timeout: %d", busyTimeout)
	}
}

func TestOpenReportsDirectoryCreationFailure(t *testing.T) {
	dir := t.TempDir()
	blockingFile := filepath.Join(dir, "blocking-file")
	if err := os.WriteFile(blockingFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(config.Config{
		SQLitePath:          filepath.Join(blockingFile, "app.db"),
		SQLiteBusyTimeoutMS: 5000,
		UploadDir:           filepath.Join(dir, "uploads"),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil || !strings.Contains(err.Error(), "create sqlite dir") {
		t.Fatalf("expected sqlite directory error, got %v", err)
	}
}
