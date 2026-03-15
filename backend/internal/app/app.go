package app

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/db"
	"github.com/cqh6666/caipu-miniapp/backend/internal/invite"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	appmiddleware "github.com/cqh6666/caipu-miniapp/backend/internal/middleware"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
	"github.com/cqh6666/caipu-miniapp/backend/internal/wechat"
)

type App struct {
	Config           config.Config
	Logger           *slog.Logger
	DB               *sql.DB
	Server           *http.Server
	RecipeAutoParser *recipe.AutoParseWorker
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

	appSettingsRepo := appsettings.NewRepository(dbConn)
	var appSettingsService *appsettings.Service

	inviteRepo := invite.NewRepository(dbConn)
	inviteService := invite.NewService(inviteRepo, kitchenService, cfg.InviteDefaultExpireHours, cfg.InviteDefaultMaxUses)
	inviteHandler := invite.NewHandler(inviteService)

	recipeRepo := recipe.NewRepository(dbConn)
	recipeService := recipe.NewService(recipeRepo, kitchenService)
	recipeHandler := recipe.NewHandler(recipeService)

	linkParseService := linkparse.NewService(linkparse.Options{
		AIBaseURL:          cfg.AIBaseURL,
		AIAPIKey:           cfg.AIAPIKey,
		AIModel:            cfg.AIModel,
		AITimeout:          time.Duration(cfg.AITimeoutSeconds) * time.Second,
		XHSSidecarEnabled:  cfg.XHSSidecarEnabled,
		XHSSidecarBaseURL:  cfg.XHSSidecarBaseURL,
		XHSSidecarTimeout:  time.Duration(cfg.XHSSidecarTimeoutSeconds) * time.Second,
		XHSSidecarProvider: cfg.XHSSidecarProvider,
		XHSSidecarAPIKey:   cfg.XHSSidecarAPIKey,
		BilibiliSessdataProvider: func(ctx context.Context) string {
			if appSettingsService == nil {
				return ""
			}
			sessdata, err := appSettingsService.CurrentBilibiliSessdata(ctx)
			if err != nil {
				logger.Warn("failed to load bilibili sessdata", "err", err)
				return ""
			}
			return sessdata
		},
	})
	linkParseHandler := linkparse.NewHandler(linkParseService)
	uploadService := upload.NewService(cfg.UploadDir, cfg.UploadPublicBaseURL, cfg.UploadMaxImageMB)
	uploadHandler := upload.NewHandler(uploadService)

	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTExpireHours)
	authRepo := auth.NewRepository(dbConn)
	wechatClient := wechat.NewClient(cfg.WechatAppID, cfg.WechatAppSecret)
	authService := auth.NewService(
		authRepo,
		kitchenService,
		tokenManager,
		wechatClient,
		cfg.WechatAppID,
		cfg.AdminOpenIDs,
		cfg.AppSettingsAccessMode,
		cfg.AppSettingsAllowedOpenIDs,
	)
	authHandler := auth.NewHandler(authService)
	authMiddleware := appmiddleware.Authenticate(tokenManager)
	appSettingsService = appsettings.NewService(appSettingsRepo, cfg.CredentialsSecret, linkParseService, authService.EnsureCanManageAppSettings)
	appSettingsHandler := appsettings.NewHandler(appSettingsService)
	recipeAutoParser := recipe.NewAutoParseWorker(
		logger,
		recipeRepo,
		linkParseService,
		cfg.RecipeAutoParseEnabled,
		time.Duration(cfg.RecipeAutoParseInterval)*time.Second,
		cfg.RecipeAutoParseBatchSize,
	)

	router := NewRouter(cfg, logger, appSettingsHandler, authHandler, kitchenHandler, inviteHandler, recipeHandler, linkParseHandler, uploadHandler, authMiddleware)

	server := &http.Server{
		Addr:              cfg.AppAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &App{
		Config:           cfg,
		Logger:           logger,
		DB:               dbConn,
		Server:           server,
		RecipeAutoParser: recipeAutoParser,
	}, nil
}

func (a *App) Start() error {
	if a.RecipeAutoParser != nil {
		a.RecipeAutoParser.Start(context.Background())
	}

	a.Logger.Info("http server starting", "addr", a.Config.AppAddr, "env", a.Config.AppEnv)
	return a.Server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	var joined error

	if a.RecipeAutoParser != nil {
		a.RecipeAutoParser.Stop()
	}

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
