package recipe

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
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

	cancel func()
	done   chan struct{}
	once   sync.Once
}

func NewImageMirrorWorker(logger *slog.Logger, repo *Repository, uploadService *upload.Service, enabled bool, interval time.Duration, batchSize int) *ImageMirrorWorker {
	return &ImageMirrorWorker{
		logger:    logger,
		repo:      repo,
		upload:    uploadService,
		enabled:   enabled,
		interval:  interval,
		batchSize: batchSize,
		done:      make(chan struct{}),
	}
}

func (w *ImageMirrorWorker) Start(parent context.Context) {
	if w == nil || !w.enabled || w.repo == nil || w.upload == nil {
		return
	}

	w.once.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		w.cancel = cancel

		go func() {
			defer close(w.done)

			w.logger.Info("recipe image mirror worker started", "interval", w.interval.String(), "batchSize", w.batchSize)
			w.runBatch(ctx)

			ticker := time.NewTicker(w.interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					w.logger.Info("recipe image mirror worker stopped")
					return
				case <-ticker.C:
					w.runBatch(ctx)
				}
			}
		}()
	})
}

func (w *ImageMirrorWorker) Stop() {
	if w == nil || w.cancel == nil {
		return
	}

	w.cancel()
	<-w.done
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

	original := cleanRecipeImageURLs(item.ImageURLs)
	mirrored, changed, err := mirrorRecipeImages(ctx, original, w.upload)
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

	for _, imageURL := range cleanRecipeImageURLs(item.ImageURLs) {
		if !uploadService.IsManagedImageURL(imageURL) && isRemoteImageURL(imageURL) {
			return true
		}
	}
	return false
}

func mirrorRecipeImages(ctx context.Context, imageURLs []string, uploadService *upload.Service) ([]string, bool, error) {
	next := make([]string, 0, len(imageURLs))
	changed := false

	for _, imageURL := range cleanRecipeImageURLs(imageURLs) {
		if uploadService.IsManagedImageURL(imageURL) || !isRemoteImageURL(imageURL) {
			next = append(next, imageURL)
			continue
		}

		image, err := uploadService.SaveRemoteImage(ctx, imageURL)
		if err != nil {
			return nil, false, err
		}

		next = append(next, strings.TrimSpace(image.URL))
		changed = true
	}

	return cleanRecipeImageURLs(next), changed, nil
}

func isRemoteImageURL(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
