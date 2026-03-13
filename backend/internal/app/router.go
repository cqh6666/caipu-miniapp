package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/invite"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	appmiddleware "github.com/cqh6666/caipu-miniapp/backend/internal/middleware"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

func NewRouter(
	cfg config.Config,
	logger *slog.Logger,
	appSettingsHandler *appsettings.Handler,
	authHandler *auth.Handler,
	kitchenHandler *kitchen.Handler,
	inviteHandler *invite.Handler,
	recipeHandler *recipe.Handler,
	linkParseHandler *linkparse.Handler,
	uploadHandler *upload.Handler,
	authMiddleware func(http.Handler) http.Handler,
) http.Handler {
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
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))
	r.Route("/api", func(api chi.Router) {
		api.Get("/healthz", healthHandler)

		api.Route("/auth", func(authRouter chi.Router) {
			authRouter.Post("/wechat/login", authHandler.WechatLogin)

			if cfg.AppEnv == "local" {
				authRouter.Post("/dev-login", authHandler.DevLogin)
			}

			authRouter.Group(func(protected chi.Router) {
				protected.Use(authMiddleware)
				protected.Get("/me", authHandler.Me)
				protected.Patch("/profile", authHandler.UpdateProfile)
			})
		})

		api.Get("/invites/{token}", inviteHandler.Preview)
		api.Get("/invite-codes/{code}", inviteHandler.PreviewByCode)

		api.Group(func(protected chi.Router) {
			protected.Use(authMiddleware)
			protected.Get("/app-settings/bilibili-session", appSettingsHandler.GetBilibiliSession)
			protected.Put("/app-settings/bilibili-session", appSettingsHandler.UpdateBilibiliSession)
			protected.Delete("/app-settings/bilibili-session", appSettingsHandler.ClearBilibiliSession)
			protected.Get("/kitchens", kitchenHandler.List)
			protected.Post("/kitchens", kitchenHandler.Create)
			protected.Patch("/kitchens/{kitchenID}", kitchenHandler.Update)
			protected.Get("/kitchens/{kitchenID}/members", kitchenHandler.ListMembers)
			protected.Post("/kitchens/{kitchenID}/invites", inviteHandler.Create)
			protected.Post("/invites/{token}/accept", inviteHandler.Accept)
			protected.Post("/invite-codes/{code}/accept", inviteHandler.AcceptByCode)
			protected.Post("/link-parsers/bilibili", linkParseHandler.ParseBilibili)
			protected.Get("/kitchens/{kitchenID}/recipes", recipeHandler.List)
			protected.Post("/kitchens/{kitchenID}/recipes", recipeHandler.Create)
			protected.Get("/recipes/{recipeID}", recipeHandler.Detail)
			protected.Put("/recipes/{recipeID}", recipeHandler.Update)
			protected.Post("/recipes/{recipeID}/reparse", recipeHandler.RequeueAutoParse)
			protected.Patch("/recipes/{recipeID}/status", recipeHandler.UpdateStatus)
			protected.Delete("/recipes/{recipeID}", recipeHandler.Delete)
			protected.Post("/uploads/images", uploadHandler.UploadImage)
		})
	})

	return r
}
