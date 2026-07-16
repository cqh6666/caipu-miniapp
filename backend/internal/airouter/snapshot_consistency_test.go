package airouter

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

func TestGetSceneKeepsSceneVersionAndProvidersInOneSnapshot(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "scene-snapshot.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(6)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
CREATE TABLE ai_route_scenes (
	scene TEXT PRIMARY KEY,
	version INTEGER NOT NULL,
	enabled INTEGER NOT NULL,
	strategy TEXT NOT NULL,
	max_attempts INTEGER NOT NULL,
	retry_policy_json TEXT NOT NULL,
	breaker_failure_threshold INTEGER NOT NULL,
	breaker_cooldown_seconds INTEGER NOT NULL,
	request_options_json TEXT NOT NULL,
	updated_by_subject TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE ai_route_providers (
	id TEXT PRIMARY KEY,
	scene TEXT NOT NULL,
	name TEXT NOT NULL DEFAULT '',
	adapter TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL,
	priority INTEGER NOT NULL,
	weight INTEGER NOT NULL,
	base_url TEXT NOT NULL DEFAULT '',
	api_key_ciphertext TEXT NOT NULL DEFAULT '',
	model TEXT NOT NULL DEFAULT '',
	timeout_seconds INTEGER NOT NULL,
	extra_json TEXT NOT NULL DEFAULT '{}',
	updated_by_subject TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL DEFAULT ''
);
INSERT INTO ai_route_scenes (
	scene, version, enabled, strategy, max_attempts, retry_policy_json,
	breaker_failure_threshold, breaker_cooldown_seconds, request_options_json
) VALUES ('summary', 1, 1, 'priority_failover', 1, '[]', 3, 60, '{}');
INSERT INTO ai_route_providers (
	id, scene, name, adapter, enabled, priority, weight, base_url, model, timeout_seconds
) VALUES ('summary-main', 'summary', '主节点', 'openai-compatible', 1, 10, 100, 'https://example.com/v1', '1', 30);
`); err != nil {
		t.Fatal(err)
	}
	service := NewService(NewRepository(db), "snapshot-secret", nil, nil, nil)

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
			if _, err := tx.Exec(`UPDATE ai_route_scenes SET version = ? WHERE scene = 'summary'`, version); err != nil {
				_ = tx.Rollback()
				errCh <- err
				return
			}
			if _, err := tx.Exec(`UPDATE ai_route_providers SET model = ? WHERE scene = 'summary'`, strconv.Itoa(version)); err != nil {
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
				config, err := service.GetScene(context.Background(), SceneSummary)
				if err != nil {
					errCh <- err
					return
				}
				if len(config.Providers) != 1 || config.Providers[0].Model != strconv.Itoa(config.Version) {
					errCh <- fmt.Errorf("incoherent scene snapshot: version=%d providers=%#v", config.Version, config.Providers)
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
