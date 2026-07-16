package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/addpreview"
	"github.com/cqh6666/caipu-miniapp/backend/internal/admin"
	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/buildinfo"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
	"github.com/cqh6666/caipu-miniapp/backend/internal/db"
	"github.com/cqh6666/caipu-miniapp/backend/internal/dietassistant"
	"github.com/cqh6666/caipu-miniapp/backend/internal/invite"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	"github.com/cqh6666/caipu-miniapp/backend/internal/logging"
	"github.com/cqh6666/caipu-miniapp/backend/internal/mealplan"
	appmiddleware "github.com/cqh6666/caipu-miniapp/backend/internal/middleware"
	"github.com/cqh6666/caipu-miniapp/backend/internal/place"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
	"github.com/cqh6666/caipu-miniapp/backend/internal/spacestats"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
	"github.com/cqh6666/caipu-miniapp/backend/internal/wechat"
)

type App struct {
	Config            config.Config
	Logger            *slog.Logger
	DB                *sql.DB
	Server            *http.Server
	RecipeAutoParser  *recipe.AutoParseWorker
	RecipeFlowchart   *recipe.FlowchartWorker
	RecipeImageMirror *recipe.ImageMirrorWorker
	workers           []backgroundWorker
}

type backgroundWorker interface {
	Start(context.Context) error
	Stop(context.Context) error
}

func New(cfg config.Config) (*App, error) {
	logger := newLogger(cfg.LogLevel)
	credentialVersion := cfg.CredentialsKeyVersion
	if credentialVersion == "" {
		credentialVersion = "v1"
	}
	previousCredentialKeys, err := credentialcipher.ParsePreviousKeys(cfg.CredentialsPreviousKeys)
	if err != nil {
		return nil, fmt.Errorf("parse previous credential keys: %w", err)
	}

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
	mealPlanRepo := mealplan.NewRepository(dbConn)
	mealPlanService := mealplan.NewService(mealPlanRepo, kitchenService)
	mealPlanHandler := mealplan.NewHandler(mealPlanService)
	placeRepo := place.NewRepository(dbConn)
	placeService := place.NewService(placeRepo, kitchenService)
	placeHandler := place.NewHandler(placeService)
	spaceStatsRepo := spacestats.NewRepository(dbConn)
	spaceStatsService := spacestats.NewService(spaceStatsRepo, kitchenService)
	spaceStatsHandler := spacestats.NewHandler(spaceStatsService)

	appSettingsRepo := appsettings.NewRepository(dbConn)
	runtimeProvider := appsettings.NewRuntimeProvider(appSettingsRepo, cfg.CredentialsSecret, cfg)
	if err := runtimeProvider.ConfigureCredentialKeys(cfg.CredentialsSecret, credentialVersion, previousCredentialKeys); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("configure runtime credential keys: %w", err)
	}
	auditService := audit.NewService(dbConn, logger)
	alertSender := aialert.NewSMTPSender()
	runtimeProvider.SetAIAlertSender(alertSender)
	aiAlertRepo := aialert.NewRepository(dbConn)
	aiAlertService := aialert.NewService(aiAlertRepo, runtimeProvider, alertSender, logger)
	aiRoutingRepo := airouter.NewRepository(dbConn)
	aiRoutingService := airouter.NewService(
		aiRoutingRepo,
		cfg.CredentialsSecret,
		buildAIRoutingCompatibilityLoader(runtimeProvider),
		auditService,
		aiAlertService,
	)
	if err := aiRoutingService.ConfigureCredentialKeys(cfg.CredentialsSecret, credentialVersion, previousCredentialKeys); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("configure AI routing credential keys: %w", err)
	}
	aiRoutingService.SetTestInputBuilder(buildAIRoutingTestInputBuilder())
	// 反向注入：aialert 通过接口消费 airouter 的运行时状态与复测能力（打破循环依赖）。
	aiAlertService.SetProviderStatusResolver(aiRoutingService)
	aiAlertService.SetProviderRetester(aiRoutingService)
	appSettingsRef := &appSettingsServiceRef{logger: logger}

	inviteRepo := invite.NewRepository(dbConn)
	inviteShareImageRenderer := invite.NewShareImageRenderer(cfg.InviteShareFontPath, cfg.InviteShareFontBoldPath)
	inviteService := invite.NewService(
		inviteRepo,
		kitchenService,
		cfg.InviteDefaultExpireHours,
		cfg.InviteDefaultMaxUses,
		inviteShareImageRenderer,
	)
	inviteHandler := invite.NewHandler(inviteService)

	recipeRepo := recipe.NewRepository(dbConn)
	linkParseService := newLinkParseService(
		cfg,
		runtimeProvider,
		aiRoutingService,
		auditService,
		appSettingsRef.bilibiliSessdata,
	)
	linkParseHandler := linkparse.NewHandler(linkParseService)
	uploadService := upload.NewServiceWithLogger(cfg.UploadDir, cfg.UploadPublicBaseURL, cfg.UploadMaxImageMB, logger)
	uploadHandler := upload.NewHandler(uploadService)
	placeService.SetUploadService(uploadService)
	addPreviewService := addpreview.NewService(kitchenService, linkParseService, addpreview.Options{
		AMapEnabled:     cfg.AMapPlacePreviewEnabled,
		AMapKey:         cfg.AMapWebServiceKey,
		AMapDefaultCity: cfg.AMapPlacePreviewDefaultCity,
		AMapTimeout:     time.Duration(cfg.AMapPlacePreviewTimeoutSeconds) * time.Second,
		AMapMaxAttempts: cfg.AMapPlacePreviewMaxAttempts,
		AMapQPSDelay:    time.Duration(cfg.AMapPlacePreviewQPSDelayMS) * time.Millisecond,
	})
	addPreviewHandler := addpreview.NewHandler(addPreviewService)
	runtimeProvider.SetBilibiliVerifier(linkParseService.VerifyBilibiliSessdata)
	recipeFlowchart := newRecipeFlowchartGenerator(
		cfg,
		runtimeProvider,
		aiRoutingService,
		auditService,
		uploadService,
	)
	recipeService := recipe.NewService(recipe.ServiceOptions{
		Repo:               recipeRepo,
		KitchenService:     kitchenService,
		UploadService:      uploadService,
		Flowchart:          recipeFlowchart,
		FlowchartEnabled:   cfg.RecipeFlowchartEnabled,
		AutoParseEnabled:   cfg.RecipeAutoParseEnabled,
		AutoParseInterval:  time.Duration(cfg.RecipeAutoParseInterval) * time.Second,
		AutoParseBatchSize: cfg.RecipeAutoParseBatchSize,
		FlowchartInterval:  time.Duration(cfg.RecipeFlowchartInterval) * time.Second,
		FlowchartBatchSize: cfg.RecipeFlowchartBatchSize,
	})
	recipeHandler := recipe.NewHandler(recipeService)
	dietAssistantRepo := dietassistant.NewRepository(dbConn)
	dietAssistantService := newDietAssistantService(
		cfg,
		dietAssistantRepo,
		kitchenService.EnsureMember,
		recipeService,
		linkParseService,
	)
	dietAssistantHandler := dietassistant.NewHandler(dietAssistantService)

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
	authMiddleware := appmiddleware.Authenticate(tokenManager, authRepo)
	appSettingsService := appsettings.NewService(appSettingsRepo, cfg.CredentialsSecret, linkParseService, authService.EnsureCanManageAppSettings)
	if err := appSettingsService.ConfigureCredentialKeys(cfg.CredentialsSecret, credentialVersion, previousCredentialKeys); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("configure app settings credential keys: %w", err)
	}
	appSettingsRef.service = appSettingsService
	appSettingsHandler := appsettings.NewHandler(appSettingsService, runtimeProvider)
	adminTokenManager := admin.NewTokenManager(cfg.AdminJWTSecret, 24*time.Hour, cfg.AdminUsername)
	adminService := admin.NewService(cfg.AdminUsername, cfg.AdminPasswordHash, adminTokenManager, cfg.AppEnv != "local", cfg.AdminCookiePath)
	serverHealthService := admin.NewServerHealthService(cfg, runtimeProvider)
	adminHandler := admin.NewHandler(adminService, auditService, runtimeProvider, appSettingsService, serverHealthService, aiRoutingService, aiAlertService)
	configureRequestGuards(adminHandler, authHandler, inviteHandler)
	adminAuthMiddleware := admin.NewAuthMiddleware(adminTokenManager)
	recipeWorkers := newRecipeWorkers(
		cfg,
		logger,
		recipeRepo,
		linkParseService,
		recipeFlowchart,
		auditService,
		uploadService,
	)

	router := NewRouter(
		cfg,
		logger,
		newHealthHandler(cfg, dbConn, logger),
		adminHandler,
		appSettingsHandler,
		authHandler,
		kitchenHandler,
		inviteHandler,
		mealPlanHandler,
		placeHandler,
		recipeHandler,
		spaceStatsHandler,
		linkParseHandler,
		addPreviewHandler,
		dietAssistantHandler,
		uploadHandler,
		authMiddleware,
		adminAuthMiddleware.Require,
	)

	server := &http.Server{
		Addr:              cfg.AppAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	workers := []backgroundWorker{
		aiAlertService,
		recipeWorkers.autoParser,
		recipeWorkers.flowchart,
		recipeWorkers.imageMirror,
	}
	return &App{
		Config:            cfg,
		Logger:            logger,
		DB:                dbConn,
		Server:            server,
		RecipeAutoParser:  recipeWorkers.autoParser,
		RecipeFlowchart:   recipeWorkers.flowchart,
		RecipeImageMirror: recipeWorkers.imageMirror,
		workers:           workers,
	}, nil
}

func (a *App) Start() error {
	for _, worker := range a.workers {
		if worker == nil {
			continue
		}
		if err := worker.Start(context.Background()); err != nil {
			return fmt.Errorf("start background worker: %w", err)
		}
	}

	build := buildinfo.Current()
	a.Logger.Info(
		"http server starting",
		"addr", a.Config.AppAddr,
		"env", a.Config.AppEnv,
		"configSources", a.Config.ConfigSourceSummary,
		"releaseId", build.ReleaseID,
		"gitCommit", build.GitCommit,
		"buildTime", build.BuildTime,
		"goToolchain", build.GoToolchain,
	)
	return a.Server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var joined error

	// Shutdown first closes listeners and drains in-flight HTTP requests. Only
	// after the server stops accepting traffic do workers get cancelled, all
	// under the same caller-provided deadline.
	if a.Server != nil {
		if err := a.Server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			joined = errors.Join(joined, err)
		}
	}
	if err := stopBackgroundWorkers(ctx, a.workers); err != nil {
		joined = errors.Join(joined, err)
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

func stopBackgroundWorkers(ctx context.Context, workers []backgroundWorker) error {
	type stopResult struct {
		worker backgroundWorker
		err    error
	}
	results := make(chan stopResult, len(workers))
	count := 0
	for _, worker := range workers {
		if worker == nil {
			continue
		}
		count++
		go func(worker backgroundWorker) {
			results <- stopResult{worker: worker, err: worker.Stop(ctx)}
		}(worker)
	}

	var joined error
	for range count {
		select {
		case result := <-results:
			if result.err != nil {
				joined = errors.Join(joined, fmt.Errorf("stop background worker %T: %w", result.worker, result.err))
			}
		case <-ctx.Done():
			return errors.Join(joined, fmt.Errorf("stop background workers: %w", ctx.Err()))
		}
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

	build := buildinfo.Current()
	return slog.New(logging.NewRedactingHandler(handler)).With(
		"release_id", build.ReleaseID,
		"git_commit", build.GitCommit,
	)
}
