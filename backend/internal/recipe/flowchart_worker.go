package recipe

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

const defaultFlowchartJobTimeout = 3 * time.Minute

type FlowchartWorker struct {
	logger    *slog.Logger
	repo      *Repository
	generator *FlowchartGenerator
	enabled   bool
	interval  time.Duration
	batchSize int

	cancel func()
	done   chan struct{}
	once   sync.Once
}

func NewFlowchartWorker(logger *slog.Logger, repo *Repository, generator *FlowchartGenerator, enabled bool, interval time.Duration, batchSize int) *FlowchartWorker {
	return &FlowchartWorker{
		logger:    logger,
		repo:      repo,
		generator: generator,
		enabled:   enabled,
		interval:  interval,
		batchSize: batchSize,
		done:      make(chan struct{}),
	}
}

func (w *FlowchartWorker) Start(parent context.Context) {
	if w == nil || !w.enabled || w.repo == nil || w.generator == nil || !w.generator.IsConfigured() {
		return
	}

	w.once.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		w.cancel = cancel

		go func() {
			defer close(w.done)

			w.logger.Info("recipe flowchart worker started", "interval", w.interval.String(), "batchSize", w.batchSize)
			w.runBatch(ctx)

			ticker := time.NewTicker(w.interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					w.logger.Info("recipe flowchart worker stopped")
					return
				case <-ticker.C:
					w.runBatch(ctx)
				}
			}
		}()
	})
}

func (w *FlowchartWorker) Stop() {
	if w == nil || w.cancel == nil {
		return
	}

	w.cancel()
	<-w.done
}

func (w *FlowchartWorker) runBatch(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, defaultFlowchartJobTimeout)
	defer cancel()

	items, err := w.repo.ListPendingFlowcharts(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("list pending recipe flowchart jobs failed", "error", err)
		return
	}

	for _, item := range items {
		if err := w.processOne(parent, item.ID); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Error("process recipe flowchart job failed", "recipeID", item.ID, "error", err)
		}
	}
}

func (w *FlowchartWorker) processOne(parent context.Context, recipeID string) error {
	ctx, cancel := context.WithTimeout(parent, defaultFlowchartJobTimeout)
	defer cancel()

	marked, err := w.repo.MarkFlowchartProcessing(ctx, recipeID)
	if err != nil {
		return err
	}
	if !marked {
		return nil
	}

	item, err := w.repo.FindByID(ctx, recipeID)
	if err != nil {
		finishedAt := time.Now().Format(time.RFC3339)
		if markErr := w.repo.MarkFlowchartFailed(ctx, recipeID, err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe flowchart failed after load error", "recipeID", recipeID, "error", markErr)
		}
		return err
	}

	if !canGenerateFlowchartForRecipe(item) {
		err = errors.New("please complete key recipe steps before generating a flowchart")
		finishedAt := time.Now().Format(time.RFC3339)
		if markErr := w.repo.MarkFlowchartFailed(ctx, recipeID, err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe flowchart invalid-input failed", "recipeID", recipeID, "error", markErr)
		}
		return err
	}

	result, err := w.generator.Generate(ctx, item)
	if err != nil {
		finishedAt := time.Now().Format(time.RFC3339)
		if markErr := w.repo.MarkFlowchartFailed(ctx, recipeID, err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe flowchart failed state failed", "recipeID", recipeID, "error", markErr)
		}
		return err
	}

	finishedAt := time.Now().Format(time.RFC3339)
	if err := w.repo.ApplyFlowchartResult(ctx, recipeID, result.ImageURL, result.SourceHash, finishedAt); err != nil {
		if markErr := w.repo.MarkFlowchartFailed(ctx, recipeID, err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe flowchart apply failure failed", "recipeID", recipeID, "error", markErr)
		}
		return err
	}

	w.logger.Info("recipe flowchart completed", "recipeID", recipeID)
	return nil
}
