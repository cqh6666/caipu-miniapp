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
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
	"github.com/cqh6666/caipu-miniapp/backend/internal/db"
	"github.com/cqh6666/caipu-miniapp/backend/internal/dietassistant"
	"github.com/cqh6666/caipu-miniapp/backend/internal/invite"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
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
	uploadService := upload.NewService(cfg.UploadDir, cfg.UploadPublicBaseURL, cfg.UploadMaxImageMB)
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
	authMiddleware := appmiddleware.Authenticate(tokenManager)
	appSettingsService := appsettings.NewService(appSettingsRepo, cfg.CredentialsSecret, linkParseService, authService.EnsureCanManageAppSettings)
	if err := appSettingsService.ConfigureCredentialKeys(cfg.CredentialsSecret, credentialVersion, previousCredentialKeys); err != nil {
		_ = dbConn.Close()
		return nil, fmt.Errorf("configure app settings credential keys: %w", err)
	}
	appSettingsRef.service = appSettingsService
	appSettingsHandler := appsettings.NewHandler(appSettingsService, runtimeProvider)
	adminTokenManager := admin.NewTokenManager(cfg.AdminJWTSecret, 24*time.Hour, cfg.AdminUsername)
	adminService := admin.NewService(cfg.AdminUsername, cfg.AdminPasswordHash, adminTokenManager, cfg.AppEnv != "local")
	serverHealthService := admin.NewServerHealthService(cfg, runtimeProvider)
	adminHandler := admin.NewHandler(adminService, auditService, runtimeProvider, appSettingsService, serverHealthService, aiRoutingService, aiAlertService)
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
		IdleTimeout:       60 * time.Second,
	}

	return &App{
		Config:            cfg,
		Logger:            logger,
		DB:                dbConn,
		Server:            server,
		RecipeAutoParser:  recipeWorkers.autoParser,
		RecipeFlowchart:   recipeWorkers.flowchart,
		RecipeImageMirror: recipeWorkers.imageMirror,
	}, nil
}

func (a *App) Start() error {
	if a.RecipeAutoParser != nil {
		a.RecipeAutoParser.Start(context.Background())
	}
	if a.RecipeFlowchart != nil {
		a.RecipeFlowchart.Start(context.Background())
	}
	if a.RecipeImageMirror != nil {
		a.RecipeImageMirror.Start(context.Background())
	}

	a.Logger.Info("http server starting", "addr", a.Config.AppAddr, "env", a.Config.AppEnv)
	return a.Server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	var joined error

	if a.RecipeAutoParser != nil {
		a.RecipeAutoParser.Stop()
	}
	if a.RecipeFlowchart != nil {
		a.RecipeFlowchart.Stop()
	}
	if a.RecipeImageMirror != nil {
		a.RecipeImageMirror.Stop()
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
