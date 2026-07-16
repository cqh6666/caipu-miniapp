package appsettings

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRuntimeSettingsSnapshotKeepsValuesAndVersionCoherent(t *testing.T) {
	db := openRuntimeSnapshotTestDB(t)
	if _, err := db.Exec(`
CREATE TABLE app_runtime_settings (
	key TEXT PRIMARY KEY,
	group_name TEXT NOT NULL,
	value_text TEXT NOT NULL DEFAULT '',
	value_ciphertext TEXT NOT NULL DEFAULT '',
	value_type TEXT NOT NULL DEFAULT 'string',
	is_secret INTEGER NOT NULL DEFAULT 0,
	is_restart_required INTEGER NOT NULL DEFAULT 0,
	description TEXT NOT NULL DEFAULT '',
	updated_by_subject TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE app_runtime_setting_groups (
	group_name TEXT PRIMARY KEY,
	version INTEGER NOT NULL,
	updated_by_subject TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL DEFAULT ''
);
INSERT INTO app_runtime_settings (key, group_name, value_text) VALUES ('coherent.value', 'coherent', '1');
INSERT INTO app_runtime_setting_groups (group_name, version) VALUES ('coherent', 1);
`); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)

	start := make(chan struct{})
	errCh := make(chan error, 8)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for version := 2; version <= 600; version++ {
			tx, err := db.BeginTx(context.Background(), nil)
			if err != nil {
				errCh <- err
				return
			}
			if _, err := tx.Exec(`UPDATE app_runtime_settings SET value_text = ? WHERE key = 'coherent.value'`, strconv.Itoa(version)); err != nil {
				_ = tx.Rollback()
				errCh <- err
				return
			}
			if _, err := tx.Exec(`UPDATE app_runtime_setting_groups SET version = ? WHERE group_name = 'coherent'`, version); err != nil {
				_ = tx.Rollback()
				errCh <- err
				return
			}
			if err := tx.Commit(); err != nil {
				errCh <- err
				return
			}
		}
	}()
	for reader := 0; reader < 4; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for iteration := 0; iteration < 600; iteration++ {
				records, versions, err := repo.ListRuntimeSettingsSnapshot(context.Background())
				if err != nil {
					errCh <- err
					return
				}
				if len(records) != 1 || records[0].ValueText != strconv.Itoa(versions["coherent"]) {
					errCh <- fmt.Errorf("incoherent runtime snapshot: records=%#v versions=%#v", records, versions)
					return
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func openRuntimeSnapshotTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "runtime-snapshot.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(6)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000;`); err != nil {
		t.Fatal(err)
	}
	return db
}
