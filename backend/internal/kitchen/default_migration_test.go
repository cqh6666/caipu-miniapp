package kitchen

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	appdb "github.com/cqh6666/caipu-miniapp/backend/internal/db"
)

func TestConsistencyMigrationBackfillsOneDefaultKitchenPerOwner(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	migrationsDir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations"))
	preMigrationDir := t.TempDir()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() == "027_add_consistency_versions.sql" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(preMigrationDir, entry.Name()), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	root := t.TempDir()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	database, err := appdb.Open(config.Config{
		SQLitePath:          filepath.Join(root, "migration.db"),
		SQLiteBusyTimeoutMS: 5000,
		UploadDir:           filepath.Join(root, "uploads"),
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if err := bootstrap.RunMigrations(context.Background(), database, logger, preMigrationDir); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
INSERT INTO users (id, openid, created_at, updated_at)
VALUES (1, 'owner-1', '2026-07-16T00:00:00Z', '2026-07-16T00:00:00Z');
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES
  (1, '自动空间', 1, '2026-07-16T00:00:00Z', '2026-07-16T00:00:00Z', 'auto'),
  (2, '自定义空间', 1, '2026-07-16T00:00:01Z', '2026-07-16T00:00:01Z', 'custom');
INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
VALUES (1, 1, 'owner', '2026-07-16T00:00:00Z'),
       (2, 1, 'owner', '2026-07-16T00:00:01Z');`); err != nil {
		t.Fatal(err)
	}

	if err := bootstrap.RunMigrations(context.Background(), database, logger, migrationsDir); err != nil {
		t.Fatal(err)
	}
	var defaultID int64
	if err := database.QueryRow(`SELECT id FROM kitchens WHERE owner_user_id = 1 AND is_default = 1`).Scan(&defaultID); err != nil {
		t.Fatal(err)
	}
	if defaultID != 1 {
		t.Fatalf("default kitchen id = %d, want auto kitchen 1", defaultID)
	}
	if _, err := database.Exec(`
INSERT INTO kitchens (name, owner_user_id, created_at, updated_at, name_source, is_default)
VALUES ('重复默认空间', 1, '2026-07-16T00:00:02Z', '2026-07-16T00:00:02Z', 'custom', 1)`); err == nil {
		t.Fatal("expected unique default kitchen constraint")
	}
	for _, table := range []string{"recipes", "places"} {
		var columns int
		if err := database.QueryRow(
			`SELECT COUNT(1) FROM pragma_table_info(?) WHERE name = 'version' AND dflt_value = '1'`,
			table,
		).Scan(&columns); err != nil {
			t.Fatal(err)
		}
		if columns != 1 {
			t.Fatalf("%s version column count = %d, want 1", table, columns)
		}
	}
}
