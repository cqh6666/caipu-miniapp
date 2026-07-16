package airouter

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	_ "modernc.org/sqlite"
)

func TestSaveSceneConcurrentUpdatesRejectStaleVersion(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "airouter-cas.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := bootstrap.RunMigrations(context.Background(), db, logger, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}

	service := NewService(NewRepository(db), "test-secret", nil, nil, nil)
	start := make(chan struct{})
	errorsCh := make(chan error, 2)
	var wg sync.WaitGroup
	for _, enabled := range []bool{true, false} {
		wg.Add(1)
		go func(enabled bool) {
			defer wg.Done()
			<-start
			_, err := service.SaveScene(context.Background(), "tester", "req-cas", SceneSummary, SceneConfig{
				Scene:       SceneSummary,
				Version:     0,
				Enabled:     enabled,
				Strategy:    StrategyPriorityFailover,
				MaxAttempts: 1,
				RetryOn:     DefaultRetryOn(),
				Breaker: BreakerConfig{
					FailureThreshold: 3,
					CooldownSeconds:  60,
				},
			})
			errorsCh <- err
		}(enabled)
	}
	close(start)
	wg.Wait()
	close(errorsCh)

	succeeded := 0
	conflicted := 0
	for err := range errorsCh {
		if err == nil {
			succeeded++
			continue
		}
		var appErr *common.AppError
		if errors.As(err, &appErr) && appErr.HTTPStatus == http.StatusConflict {
			conflicted++
			continue
		}
		t.Fatalf("SaveScene() unexpected error = %v", err)
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("success/conflict = %d/%d, want 1/1", succeeded, conflicted)
	}
	config, err := service.GetScene(context.Background(), SceneSummary)
	if err != nil {
		t.Fatalf("GetScene() error = %v", err)
	}
	if config.Version != 1 {
		t.Fatalf("config.Version = %d, want 1", config.Version)
	}
}
