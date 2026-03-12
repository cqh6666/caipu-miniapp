package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	appmiddleware "github.com/cqh6666/caipu-miniapp/backend/internal/middleware"
)

func NewRouter(cfg config.Config, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(appmiddleware.RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	healthHandler := func(w http.ResponseWriter, r *http.Request) {
		common.WriteData(w, http.StatusOK, map[string]any{
			"status": "ok",
			"app":    cfg.AppName,
			"env":    cfg.AppEnv,
			"time":   time.Now().Format(time.RFC3339),
		})
	}

	r.Get("/healthz", healthHandler)
	r.Route("/api", func(api chi.Router) {
		api.Get("/healthz", healthHandler)
	})

	return r
}
