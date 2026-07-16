package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/buildinfo"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
)

const readinessTimeout = 2 * time.Second

type healthHandler struct {
	cfg    config.Config
	db     *sql.DB
	logger *slog.Logger
	build  buildinfo.Info
}

func newHealthHandler(cfg config.Config, db *sql.DB, logger *slog.Logger) *healthHandler {
	return &healthHandler{
		cfg:    cfg,
		db:     db,
		logger: logger,
		build:  buildinfo.Current(),
	}
}

func (h *healthHandler) live(w http.ResponseWriter, _ *http.Request) {
	h.writeStatus(w, http.StatusOK, "ok")
}

func (h *healthHandler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()

	if err := h.checkReady(ctx); err != nil {
		h.logger.Warn("readiness check failed", "error", err)
		h.writeStatus(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	h.writeStatus(w, http.StatusOK, "ok")
}

func (h *healthHandler) checkReady(ctx context.Context) error {
	if err := h.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if err := bootstrap.CheckMigrationsCurrent(ctx, h.db, h.cfg.MigrationDir); err != nil {
		return fmt.Errorf("check migrations: %w", err)
	}
	if err := checkDirectoryWritable(filepath.Dir(h.cfg.SQLitePath)); err != nil {
		return fmt.Errorf("check SQLite directory: %w", err)
	}
	if err := checkDirectoryWritable(h.cfg.UploadDir); err != nil {
		return fmt.Errorf("check upload directory: %w", err)
	}
	return nil
}

func checkDirectoryWritable(dir string) error {
	probe, err := os.CreateTemp(dir, ".readiness-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	if err := probe.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Remove(name); err != nil {
		return err
	}
	return nil
}

func (h *healthHandler) writeStatus(w http.ResponseWriter, status int, state string) {
	w.Header().Set("X-Release-ID", h.build.ReleaseID)
	payload := common.Response{
		Code:    common.CodeOK,
		Message: "ok",
		Data: map[string]any{
			"status":      state,
			"app":         h.cfg.AppName,
			"env":         h.cfg.AppEnv,
			"releaseId":   h.build.ReleaseID,
			"gitCommit":   h.build.GitCommit,
			"buildTime":   h.build.BuildTime,
			"goToolchain": h.build.GoToolchain,
			"time":        time.Now().Format(time.RFC3339),
		},
	}
	if status == http.StatusServiceUnavailable {
		payload.Code = common.CodeServiceUnavailable
		payload.Message = "service unavailable"
	}
	common.WriteJSON(w, status, payload)
}
