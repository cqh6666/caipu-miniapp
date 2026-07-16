package recipe

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

const defaultImageMirrorJobTimeout = 2 * time.Minute

type ImageMirrorWorker struct {
	logger    *slog.Logger
	repo      *Repository
	upload    *upload.Service
	enabled   bool
	interval  time.Duration
	batchSize int

	lifecycle *workerLifecycle
}

func NewImageMirrorWorker(logger *slog.Logger, repo *Repository, uploadService *upload.Service, enabled bool, interval time.Duration, batchSize int) *ImageMirrorWorker {
	return &ImageMirrorWorker{
		logger:    logger,
		repo:      repo,
		upload:    uploadService,
		enabled:   enabled,
		interval:  interval,
		batchSize: batchSize,
		lifecycle: newWorkerLifecycle(),
	}
}

func (w *ImageMirrorWorker) Start(parent context.Context) error {
	if w == nil || !w.enabled || w.repo == nil || w.upload == nil {
		return nil
	}
	return w.lifecycle.Start(
		parent,
		"recipe image mirror",
		w.interval,
		w.runBatch,
		func() {
			w.logger.Info("recipe image mirror worker started", "interval", w.interval.String(), "batchSize", w.batchSize)
		},
		func() { w.logger.Info("recipe image mirror worker stopped") },
	)
}

func (w *ImageMirrorWorker) Stop(ctx context.Context) error {
	if w == nil {
		return nil
	}
	return w.lifecycle.Stop(ctx, "recipe image mirror")
}

func (w *ImageMirrorWorker) runBatch(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, defaultImageMirrorJobTimeout)
	defer cancel()

	scanLimit := w.batchSize * 8
	if scanLimit < 20 {
		scanLimit = 20
	}

	items, err := w.repo.ListImageMirrorCandidates(ctx, scanLimit)
	if err != nil {
		w.logger.Error("list recipe image mirror candidates failed", "error", err)
		return
	}

	processed := 0
	for _, item := range items {
		if processed >= w.batchSize {
			break
		}
		if !needsImageMirroring(item, w.upload) {
			continue
		}
		if err := w.processOne(parent, item); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Error("mirror recipe images failed", "recipeID", item.ID, "error", err)
			continue
		}
		processed++
	}
}

func (w *ImageMirrorWorker) processOne(parent context.Context, item Recipe) error {
	ctx, cancel := context.WithTimeout(parent, defaultImageMirrorJobTimeout)
	defer cancel()

	original := recipeImageURLsFromItem(item)
	currentMetas := fillManagedImageHashes(recipeImageMetasFromItem(item), w.upload)
	mirrored, changed, err := mirrorRecipeImages(ctx, currentMetas, w.upload)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}

	applied, err := w.repo.ApplyMirroredImages(ctx, item.ID, original, mirrored, time.Now().Format(time.RFC3339))
	if err != nil {
		return err
	}
	if applied {
		w.logger.Info("recipe images mirrored", "recipeID", item.ID, "count", len(mirrored))
	}
	return nil
}

func needsImageMirroring(item Recipe, uploadService *upload.Service) bool {
	if uploadService == nil {
		return false
	}

	for _, meta := range recipeImageMetasFromItem(item) {
		if !uploadService.IsManagedImageURL(meta.URL) && isRemoteImageURL(meta.URL) {
			return true
		}
	}
	return false
}

func mirrorRecipeImages(ctx context.Context, imageMetas []RecipeImageMeta, uploadService *upload.Service) ([]RecipeImageMeta, bool, error) {
	current := fillManagedImageHashes(imageMetas, uploadService)
	next := make([]RecipeImageMeta, 0, len(current))
	changed := false

	for _, meta := range current {
		if uploadService.IsManagedImageURL(meta.URL) || !isRemoteImageURL(meta.URL) {
			next = append(next, meta)
			continue
		}

		image, err := uploadService.SaveRemoteImage(ctx, meta.URL)
		if err != nil {
			return nil, false, err
		}

		mirrored := meta
		if strings.TrimSpace(mirrored.OriginURL) == "" {
			mirrored.OriginURL = strings.TrimSpace(meta.URL)
		}
		mirrored.URL = strings.TrimSpace(image.URL)
		mirrored.ContentHash = normalizeRecipeImageContentHash(image.ContentHash)
		next = append(next, mirrored)
		changed = true
	}

	next = dedupeRecipeImageMetas(next)
	if !recipeImageMetasEqual(current, next) {
		changed = true
	}

	return next, changed, nil
}

func isRemoteImageURL(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
