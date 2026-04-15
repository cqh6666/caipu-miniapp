package app

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/admin"
	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/db"
	"github.com/cqh6666/caipu-miniapp/backend/internal/invite"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	"github.com/cqh6666/caipu-miniapp/backend/internal/mealplan"
	appmiddleware "github.com/cqh6666/caipu-miniapp/backend/internal/middleware"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
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

	appSettingsRepo := appsettings.NewRepository(dbConn)
	runtimeProvider := appsettings.NewRuntimeProvider(appSettingsRepo, cfg.CredentialsSecret, cfg)
	auditService := audit.NewService(dbConn, logger)
	aiRoutingRepo := airouter.NewRepository(dbConn)
	aiRoutingService := airouter.NewService(
		aiRoutingRepo,
		cfg.CredentialsSecret,
		buildAIRoutingCompatibilityLoader(runtimeProvider),
		auditService,
	)
	aiRoutingService.SetTestInputBuilder(buildAIRoutingTestInputBuilder())
	var appSettingsService *appsettings.Service

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
	linkParseService := linkparse.NewService(linkparse.Options{
		AIBaseURL:               cfg.AIBaseURL,
		AIAPIKey:                cfg.AIAPIKey,
		AIModel:                 cfg.AIModel,
		AITimeout:               time.Duration(cfg.AITimeoutSeconds) * time.Second,
		AITitleEnabled:          cfg.AITitleEnabled,
		AITitleBaseURL:          cfg.AITitleBaseURL,
		AITitleAPIKey:           cfg.AITitleAPIKey,
		AITitleModel:            cfg.AITitleModel,
		AITitleStream:           cfg.AITitleStream,
		AITitleTemperature:      cfg.AITitleTemperature,
		AITitleMaxTokens:        cfg.AITitleMaxTokens,
		AITitleTimeout:          time.Duration(cfg.AITitleTimeoutSeconds) * time.Second,
		LinkparseSidecarEnabled: cfg.LinkparseSidecarEnabled,
		LinkparseSidecarBaseURL: cfg.LinkparseSidecarBaseURL,
		LinkparseSidecarTimeout: time.Duration(cfg.LinkparseSidecarTimeoutSec) * time.Second,
		LinkparseSidecarAPIKey:  cfg.LinkparseSidecarAPIKey,
		RuntimeConfigLoader: func(ctx context.Context) linkparse.RuntimeConfig {
			summary := runtimeProvider.SummaryAI(ctx)
			title := runtimeProvider.TitleAI(ctx)
			sidecar := runtimeProvider.LinkparseSidecar(ctx)
			return linkparse.RuntimeConfig{
				SummaryAI: linkparse.SummaryAIConfig{
					BaseURL: summary.BaseURL,
					APIKey:  summary.APIKey,
					Model:   summary.Model,
					Timeout: summary.Timeout,
				},
				TitleAI: linkparse.TitleAIConfig{
					Enabled:     title.Enabled,
					BaseURL:     title.BaseURL,
					APIKey:      title.APIKey,
					Model:       title.Model,
					Stream:      title.Stream,
					Temperature: title.Temperature,
					MaxTokens:   title.MaxTokens,
					Timeout:     title.Timeout,
				},
				LinkparseSidecar: linkparse.LinkparseSidecarConfig{
					Enabled: sidecar.Enabled,
					BaseURL: sidecar.BaseURL,
					APIKey:  sidecar.APIKey,
					Timeout: sidecar.Timeout,
				},
			}
		},
		AIRouter: aiRoutingService,
		Tracker: auditService,
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
	runtimeProvider.SetBilibiliVerifier(linkParseService.VerifyBilibiliSessdata)
	uploadService := upload.NewService(cfg.UploadDir, cfg.UploadPublicBaseURL, cfg.UploadMaxImageMB)
	uploadHandler := upload.NewHandler(uploadService)
	recipeFlowchart := recipe.NewFlowchartGenerator(recipe.FlowchartOptions{
		BaseURL: cfg.AIFlowchartBaseURL,
		APIKey:  cfg.AIFlowchartAPIKey,
		Model:   cfg.AIFlowchartModel,
		Timeout: time.Duration(cfg.AIFlowchartTimeoutSeconds) * time.Second,
		RuntimeConfigLoader: func(ctx context.Context) recipe.FlowchartRuntimeConfig {
			flowchartCfg := runtimeProvider.FlowchartAI(ctx)
			return recipe.FlowchartRuntimeConfig{
				BaseURL: flowchartCfg.BaseURL,
				APIKey:  flowchartCfg.APIKey,
				Model:   flowchartCfg.Model,
				Timeout: flowchartCfg.Timeout,
			}
		},
		AIRouter: aiRoutingService,
		Tracker: auditService,
	}, uploadService)
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
	adminTokenManager := admin.NewTokenManager(cfg.AdminJWTSecret, 24*time.Hour)
	adminService := admin.NewService(cfg.AdminUsername, cfg.AdminPasswordHash, adminTokenManager, cfg.AppEnv != "local")
	serverHealthService := admin.NewServerHealthService(cfg, runtimeProvider)
	adminHandler := admin.NewHandler(adminService, auditService, runtimeProvider, appSettingsService, serverHealthService, aiRoutingService)
	adminAuthMiddleware := admin.NewAuthMiddleware(adminTokenManager)
	recipeAutoParser := recipe.NewAutoParseWorker(
		logger,
		recipeRepo,
		linkParseService,
		cfg.RecipeAutoParseEnabled,
		time.Duration(cfg.RecipeAutoParseInterval)*time.Second,
		cfg.RecipeAutoParseBatchSize,
	)
	recipeFlowchartWorker := recipe.NewFlowchartWorker(
		logger,
		recipeRepo,
		recipeFlowchart,
		auditService,
		cfg.RecipeFlowchartEnabled,
		cfg.RecipeFlowchartAutoEnqueue,
		time.Duration(cfg.RecipeFlowchartInterval)*time.Second,
		cfg.RecipeFlowchartBatchSize,
	)
	recipeImageMirror := recipe.NewImageMirrorWorker(
		logger,
		recipeRepo,
		uploadService,
		cfg.RecipeImageMirrorEnabled,
		time.Duration(cfg.RecipeImageMirrorInterval)*time.Second,
		cfg.RecipeImageMirrorBatchSize,
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
		recipeHandler,
		linkParseHandler,
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
		RecipeAutoParser:  recipeAutoParser,
		RecipeFlowchart:   recipeFlowchartWorker,
		RecipeImageMirror: recipeImageMirror,
	}, nil
}

func buildAIRoutingCompatibilityLoader(runtimeProvider *appsettings.RuntimeProvider) airouter.CompatibilityLoader {
	return func(ctx context.Context, scene airouter.Scene) airouter.SceneConfig {
		switch scene {
		case airouter.SceneSummary:
			summary := runtimeProvider.SummaryAI(ctx)
			enabled := summary.BaseURL != "" && summary.Model != ""
			return airouter.SceneConfig{
				Scene:       scene,
				Enabled:     enabled,
				Strategy:    airouter.StrategyPriorityFailover,
				MaxAttempts: 1,
				RetryOn:     airouter.DefaultRetryOn(),
				Breaker:     airouter.DefaultBreakerConfig(),
				Providers: []airouter.ProviderConfig{
					{
						ID:             "summary-compat",
						Name:           "兼容单节点",
						Adapter:        airouter.AdapterOpenAICompatible,
						Enabled:        enabled,
						Priority:       10,
						BaseURL:        summary.BaseURL,
						APIKey:         summary.APIKey,
						APIKeyMasked:   maskCompatSecret(summary.APIKey),
						HasAPIKey:      strings.TrimSpace(summary.APIKey) != "",
						Model:          summary.Model,
						TimeoutSeconds: int(summary.Timeout.Seconds()),
					},
				},
			}
		case airouter.SceneTitle:
			summary := runtimeProvider.SummaryAI(ctx)
			title := runtimeProvider.TitleAI(ctx)
			if strings.TrimSpace(title.BaseURL) == "" {
				title.BaseURL = summary.BaseURL
			}
			if strings.TrimSpace(title.APIKey) == "" {
				title.APIKey = summary.APIKey
			}
			if strings.TrimSpace(title.Model) == "" {
				title.Model = summary.Model
			}
			enabled := title.Enabled && title.BaseURL != "" && title.Model != ""
			return airouter.SceneConfig{
				Scene:       scene,
				Enabled:     enabled,
				Strategy:    airouter.StrategyRoundRobinFailover,
				MaxAttempts: 1,
				RetryOn:     airouter.DefaultRetryOn(),
				Breaker:     airouter.DefaultBreakerConfig(),
				RequestOptions: airouter.RequestOptions{
					Stream:      title.Stream,
					Temperature: title.Temperature,
					MaxTokens:   title.MaxTokens,
				},
				Providers: []airouter.ProviderConfig{
					{
						ID:             "title-compat",
						Name:           "兼容单节点",
						Adapter:        airouter.AdapterOpenAICompatible,
						Enabled:        enabled,
						Priority:       10,
						BaseURL:        title.BaseURL,
						APIKey:         title.APIKey,
						APIKeyMasked:   maskCompatSecret(title.APIKey),
						HasAPIKey:      strings.TrimSpace(title.APIKey) != "",
						Model:          title.Model,
						TimeoutSeconds: int(title.Timeout.Seconds()),
					},
				},
			}
		case airouter.SceneFlowchart:
			flowchart := runtimeProvider.FlowchartAI(ctx)
			enabled := flowchart.BaseURL != "" && flowchart.Model != ""
			return airouter.SceneConfig{
				Scene:       scene,
				Enabled:     enabled,
				Strategy:    airouter.StrategyPriorityFailover,
				MaxAttempts: 1,
				RetryOn:     airouter.DefaultRetryOn(),
				Breaker:     airouter.DefaultBreakerConfig(),
				Providers: []airouter.ProviderConfig{
					{
						ID:             "flowchart-compat",
						Name:           "兼容单节点",
						Adapter:        airouter.AdapterOpenAICompatible,
						Enabled:        enabled,
						Priority:       10,
						BaseURL:        flowchart.BaseURL,
						APIKey:         flowchart.APIKey,
						APIKeyMasked:   maskCompatSecret(flowchart.APIKey),
						HasAPIKey:      strings.TrimSpace(flowchart.APIKey) != "",
						Model:          flowchart.Model,
						TimeoutSeconds: int(flowchart.Timeout.Seconds()),
					},
				},
			}
		default:
			return airouter.SceneConfig{
				Scene:       scene,
				Strategy:    airouter.StrategyPriorityFailover,
				MaxAttempts: 1,
				RetryOn:     airouter.DefaultRetryOn(),
				Breaker:     airouter.DefaultBreakerConfig(),
			}
		}
	}
}

func buildAIRoutingTestInputBuilder() airouter.TestInputBuilder {
	return func(scene airouter.Scene) (airouter.ChatCompletionInput, bool) {
		switch scene {
		case airouter.SceneSummary:
			return linkparse.BuildSummaryRouteTestInput(), true
		case airouter.SceneTitle:
			return linkparse.BuildTitleRouteTestInput(), true
		case airouter.SceneFlowchart:
			return recipe.BuildFlowchartRouteTestInput(), true
		default:
			return airouter.ChatCompletionInput{}, false
		}
	}
}

func maskCompatSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
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
