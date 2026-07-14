package app

import (
	"log/slog"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

type recipeWorkers struct {
	autoParser  *recipe.AutoParseWorker
	flowchart   *recipe.FlowchartWorker
	imageMirror *recipe.ImageMirrorWorker
}

func newRecipeWorkers(
	cfg config.Config,
	logger *slog.Logger,
	repo *recipe.Repository,
	parser *linkparse.Service,
	flowchartGenerator *recipe.FlowchartGenerator,
	tracker audit.Tracker,
	uploadService *upload.Service,
) recipeWorkers {
	return recipeWorkers{
		autoParser: recipe.NewAutoParseWorkerWithOptions(recipe.AutoParseWorkerOptions{
			Logger:                   logger,
			Repo:                     repo,
			Parser:                   parser,
			Enabled:                  cfg.RecipeAutoParseEnabled,
			Interval:                 time.Duration(cfg.RecipeAutoParseInterval) * time.Second,
			BatchSize:                cfg.RecipeAutoParseBatchSize,
			MaxAttempts:              cfg.RecipeAutoParseMaxAttempts,
			RetryBaseDelay:           time.Duration(cfg.RecipeAutoParseRetryBaseSec) * time.Second,
			StaleProcessingThreshold: time.Duration(cfg.RecipeAutoParseStaleSec) * time.Second,
		}),
		flowchart: recipe.NewFlowchartWorker(
			logger,
			repo,
			flowchartGenerator,
			tracker,
			cfg.RecipeFlowchartEnabled,
			cfg.RecipeFlowchartAutoEnqueue,
			time.Duration(cfg.RecipeFlowchartInterval)*time.Second,
			cfg.RecipeFlowchartBatchSize,
		),
		imageMirror: recipe.NewImageMirrorWorker(
			logger,
			repo,
			uploadService,
			cfg.RecipeImageMirrorEnabled,
			time.Duration(cfg.RecipeImageMirrorInterval)*time.Second,
			cfg.RecipeImageMirrorBatchSize,
		),
	}
}
