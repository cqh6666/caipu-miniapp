package app

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/db"
	"github.com/cqh6666/caipu-miniapp/backend/internal/invite"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	appmiddleware "github.com/cqh6666/caipu-miniapp/backend/internal/middleware"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
	"github.com/cqh6666/caipu-miniapp/backend/internal/wechat"
)

type App struct {
	Config config.Config
	Logger *slog.Logger
	DB     *sql.DB
	Server *http.Server
}

func New(cfg config.Config) (*App, error) {
	logger := newLogger(cfg.LogLevel)

	dbConn, err := db.Open(cfg, logger)
	if err != nil {
		return nil, err
	}

	if err := bootstrap.RunMigrations(context.Background(), dbConn, logger, cfg.MigrationDir); err != nil {
		_ = dbConn.Close()
		return nil, err
	}

	kitchenRepo := kitchen.NewRepository(dbConn)
	kitchenService := kitchen.NewService(kitchenRepo)
	kitchenHandler := kitchen.NewHandler(kitchenService)

	inviteRepo := invite.NewRepository(dbConn)
	inviteService := invite.NewService(inviteRepo, kitchenService, cfg.InviteDefaultExpireHours, cfg.InviteDefaultMaxUses)
	inviteHandler := invite.NewHandler(inviteService)

	recipeRepo := recipe.NewRepository(dbConn)
	recipeService := recipe.NewService(recipeRepo, kitchenService)
	recipeHandler := recipe.NewHandler(recipeService)

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTExpireHours)
	authRepo := auth.NewRepository(dbConn)
	wechatClient := wechat.NewClient(cfg.WechatAppID, cfg.WechatAppSecret)
	authService := auth.NewService(authRepo, kitchenService, tokenManager, wechatClient)
	authHandler := auth.NewHandler(authService)
	authMiddleware := appmiddleware.Authenticate(tokenManager)

	router := NewRouter(cfg, logger, authHandler, kitchenHandler, inviteHandler, recipeHandler, authMiddleware)

	server := &http.Server{
		Addr:              cfg.AppAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &App{
		Config: cfg,
		Logger: logger,
		DB:     dbConn,
		Server: server,
	}, nil
}

func (a *App) Start() error {
	a.Logger.Info("http server starting", "addr", a.Config.AppAddr, "env", a.Config.AppEnv)
	return a.Server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	var joined error

	if a.Server != nil {
		if err := a.Server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			joined = errors.Join(joined, err)
		}
	}

	if a.DB != nil {
		if err := a.DB.Close(); err != nil {
			joined = errors.Join(joined, err)
		}
	}

	if joined == nil {
		a.Logger.Info("app shutdown complete")
	}

	return joined
}

func newLogger(level string) *slog.Logger {
	var slogLevel slog.Level

	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: false,
	})

	return slog.New(handler)
}
