package recipe

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const defaultFlowchartJobTimeout = 3 * time.Minute
const staleFlowchartProcessingThreshold = 10 * time.Minute

type FlowchartWorker struct {
	logger             *slog.Logger
	repo               *Repository
	generator          *FlowchartGenerator
	tracker            audit.Tracker
	enabled            bool
	autoEnqueueEnabled bool
	interval           time.Duration
	batchSize          int

	cancel func()
	done   chan struct{}
	once   sync.Once
}

func NewFlowchartWorker(logger *slog.Logger, repo *Repository, generator *FlowchartGenerator, tracker audit.Tracker, enabled bool, autoEnqueueEnabled bool, interval time.Duration, batchSize int) *FlowchartWorker {
	return &FlowchartWorker{
		logger:             logger,
		repo:               repo,
		generator:          generator,
		tracker:            tracker,
		enabled:            enabled,
		autoEnqueueEnabled: autoEnqueueEnabled,
		interval:           interval,
		batchSize:          batchSize,
		done:               make(chan struct{}),
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

			w.logger.Info(
				"recipe flowchart worker started",
				"interval",
				w.interval.String(),
				"batchSize",
				w.batchSize,
				"autoEnqueueEnabled",
				w.autoEnqueueEnabled,
			)
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

	w.requeueStaleProcessing(ctx)

	items, err := w.repo.ListPendingFlowcharts(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("list pending recipe flowchart jobs failed", "error", err)
		return
	}
	if len(items) == 0 {
		processing, err := w.repo.CountProcessingFlowcharts(ctx)
		if err != nil {
			w.logger.Error("count processing recipe flowchart jobs failed", "error", err)
			return
		}
		if processing == 0 {
			w.enqueueAutoCandidates(ctx)
			items, err = w.repo.ListPendingFlowcharts(ctx, w.batchSize)
			if err != nil {
				w.logger.Error("reload pending recipe flowchart jobs failed", "error", err)
				return
			}
		}
	}

	for _, item := range items {
		if err := w.processOne(parent, item.ID); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Error("process recipe flowchart job failed", "recipeID", item.ID, "error", err)
		}
	}
}

func (w *FlowchartWorker) enqueueAutoCandidates(ctx context.Context) {
	if w == nil || !w.autoEnqueueEnabled || w.repo == nil {
		return
	}

	scanLimit := w.batchSize * 10
	if scanLimit < 50 {
		scanLimit = 50
	}

	items, err := w.repo.ListAutoFlowchartCandidates(ctx, scanLimit)
	if err != nil {
		w.logger.Error("list auto recipe flowchart candidates failed", "error", err)
		return
	}

	requestedAt := time.Now().Format(time.RFC3339)
	for _, item := range items {
		if !canGenerateFlowchartForRecipe(item) {
			continue
		}

		marked, err := w.repo.MarkAutoFlowchartPending(ctx, item.ID, requestedAt)
		if err != nil {
			w.logger.Error("mark auto recipe flowchart pending failed", "recipeID", item.ID, "error", err)
			continue
		}
		if !marked {
			continue
		}

		w.logger.Info("queued auto recipe flowchart candidate", "recipeID", item.ID)
		return
	}
}

func (w *FlowchartWorker) requeueStaleProcessing(ctx context.Context) {
	if w == nil || w.repo == nil {
		return
	}

	// Keep the reclaim threshold comfortably above the normal job timeout so a
	// restarted instance does not steal a legitimately in-flight job.
	now := time.Now()
	staleBefore := now.Add(-staleFlowchartProcessingThreshold).Format(time.RFC3339)
	requeuedAt := now.Format(time.RFC3339)

	requeued, err := w.repo.RequeueStaleFlowcharts(ctx, staleBefore, requeuedAt)
	if err != nil {
		w.logger.Error("requeue stale recipe flowchart jobs failed", "error", err)
		return
	}
	if requeued > 0 {
		w.logger.Warn("requeued stale recipe flowchart jobs", "count", requeued, "staleBefore", staleBefore)
	}
}

func (w *FlowchartWorker) processOne(parent context.Context, recipeID string) error {
	ctx, cancel := context.WithTimeout(parent, defaultFlowchartJobTimeout)
	defer cancel()
	startedAt := time.Now()

	marked, err := w.repo.MarkFlowchartProcessing(ctx, recipeID)
	if err != nil {
		return err
	}
	if !marked {
		return nil
	}

	finish := audit.FinishFunc(func(context.Context, audit.JobResult) error { return nil })
	if w != nil && w.tracker != nil {
		jobID, startedFinish, trackErr := w.tracker.StartJob(ctx, audit.JobInput{
			Scene:         audit.SceneFlowchart,
			TargetType:    "recipe",
			TargetID:      recipeID,
			TriggerSource: "worker",
			RequestID:     common.RequestID(ctx),
			Meta: map[string]any{
				"recipe_id": recipeID,
			},
		})
		if trackErr == nil && jobID > 0 {
			ctx = audit.WithJobContext(ctx, audit.SceneFlowchart, jobID)
			finish = startedFinish
		}
	}

	finishJob := func(status, provider, model string, err error, meta map[string]any) {
		jobResult := audit.JobResult{
			Status:        status,
			FinalProvider: provider,
			FinalModel:    model,
			FinishedAt:    audit.NowRFC3339(),
			Meta:          meta,
		}
		if err != nil {
			jobResult.ErrorMessage = err.Error()
		}
		_ = finish(ctx, jobResult)
	}

	w.logger.Info("recipe flowchart job started", "recipeID", recipeID)

	item, err := w.repo.FindByID(ctx, recipeID)
	if err != nil {
		finishedAt := time.Now().Format(time.RFC3339)
		if markErr := w.repo.MarkFlowchartFailed(ctx, recipeID, err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe flowchart failed after load error", "recipeID", recipeID, "error", markErr)
		}
		w.logFailure(recipeID, "load", startedAt, err)
		finishJob(audit.JobStatusFailed, "", "", err, map[string]any{"stage": "load"})
		return err
	}

	if !canGenerateFlowchartForRecipe(item) {
		err = errors.New("please complete key recipe steps before generating a flowchart")
		finishedAt := time.Now().Format(time.RFC3339)
		if markErr := w.repo.MarkFlowchartFailed(ctx, recipeID, err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe flowchart invalid-input failed", "recipeID", recipeID, "error", markErr)
		}
		w.logFailure(recipeID, "validate", startedAt, err)
		finishJob(audit.JobStatusFailed, "", "", err, map[string]any{"stage": "validate"})
		return err
	}

	result, err := w.generator.Generate(ctx, item)
	if err != nil {
		finishedAt := time.Now().Format(time.RFC3339)
		if markErr := w.repo.MarkFlowchartFailed(ctx, recipeID, err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe flowchart failed state failed", "recipeID", recipeID, "error", markErr)
		}
		w.logFailure(recipeID, "generate", startedAt, err)
		finishJob(audit.JobStatusFromError(err), "", "", err, map[string]any{"stage": "generate"})
		return err
	}

	finishedAt := time.Now().Format(time.RFC3339)
	if err := w.repo.ApplyFlowchartResult(ctx, recipeID, result.ImageURL, result.SourceHash, finishedAt); err != nil {
		if markErr := w.repo.MarkFlowchartFailed(ctx, recipeID, err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe flowchart apply failure failed", "recipeID", recipeID, "error", markErr)
		}
		w.logFailure(recipeID, "persist", startedAt, err)
		finishJob(audit.JobStatusFailed, result.Provider, result.Model, err, map[string]any{"stage": "persist"})
		return err
	}

	w.logger.Info(
		"recipe flowchart completed",
		"recipeID",
		recipeID,
		"duration",
		time.Since(startedAt).String(),
		"imageURL",
		result.ImageURL,
	)
	finishJob(audit.JobStatusSuccess, result.Provider, result.Model, nil, map[string]any{
		"stage":       "completed",
		"image_url":   result.ImageURL,
		"source_hash": result.SourceHash,
	})
	return nil
}

func (w *FlowchartWorker) logFailure(recipeID string, stage string, startedAt time.Time, err error) {
	if w == nil || w.logger == nil {
		return
	}

	w.logger.Error(
		"recipe flowchart failed",
		"recipeID",
		recipeID,
		"stage",
		stage,
		"duration",
		time.Since(startedAt).String(),
		"error",
		err,
		"cause",
		flowchartErrorCause(err),
	)
}
