package recipe

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
)

const defaultAutoParseJobTimeout = 90 * time.Second

type AutoParseWorker struct {
	logger    *slog.Logger
	repo      *Repository
	parser    *linkparse.Service
	enabled   bool
	interval  time.Duration
	batchSize int

	cancel func()
	done   chan struct{}
	once   sync.Once
}

func NewAutoParseWorker(logger *slog.Logger, repo *Repository, parser *linkparse.Service, enabled bool, interval time.Duration, batchSize int) *AutoParseWorker {
	return &AutoParseWorker{
		logger:    logger,
		repo:      repo,
		parser:    parser,
		enabled:   enabled,
		interval:  interval,
		batchSize: batchSize,
		done:      make(chan struct{}),
	}
}

func (w *AutoParseWorker) Start(parent context.Context) {
	if w == nil || !w.enabled || w.parser == nil || w.repo == nil {
		return
	}

	w.once.Do(func() {
		ctx, cancel := context.WithCancel(parent)
		w.cancel = cancel

		go func() {
			defer close(w.done)

			w.logger.Info("recipe auto-parse worker started", "interval", w.interval.String(), "batchSize", w.batchSize)
			w.runBatch(ctx)

			ticker := time.NewTicker(w.interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					w.logger.Info("recipe auto-parse worker stopped")
					return
				case <-ticker.C:
					w.runBatch(ctx)
				}
			}
		}()
	})
}

func (w *AutoParseWorker) Stop() {
	if w == nil || w.cancel == nil {
		return
	}

	w.cancel()
	<-w.done
}

func (w *AutoParseWorker) runBatch(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, defaultAutoParseJobTimeout)
	defer cancel()

	w.enqueueLegacyCandidates(ctx)

	items, err := w.repo.ListPendingAutoParse(ctx, w.batchSize)
	if err != nil {
		w.logger.Error("list pending recipe auto-parse jobs failed", "error", err)
		return
	}

	for _, item := range items {
		if err := w.processOne(parent, item); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Error("process recipe auto-parse job failed", "recipeID", item.ID, "error", err)
		}
	}
}

func (w *AutoParseWorker) enqueueLegacyCandidates(ctx context.Context) {
	scanLimit := int(math.Max(float64(w.batchSize*10), 50))
	items, err := w.repo.ListLegacyAutoParseCandidates(ctx, scanLimit)
	if err != nil {
		w.logger.Error("list legacy recipe auto-parse candidates failed", "error", err)
		return
	}

	queued := 0
	now := time.Now().Format(time.RFC3339)
	for _, item := range items {
		if !shouldQueueAutoParse(item.Link, item.ParsedContent, item.MealType, item.Title, item.Ingredient) {
			continue
		}

		marked, err := w.repo.MarkAutoParsePending(ctx, item.ID, "bilibili", now)
		if err != nil {
			w.logger.Error("mark legacy recipe auto-parse pending failed", "recipeID", item.ID, "error", err)
			continue
		}
		if marked {
			queued++
		}
	}

	if queued > 0 {
		w.logger.Info("requeued legacy recipe auto-parse jobs", "count", queued)
	}
}

func (w *AutoParseWorker) processOne(parent context.Context, item Recipe) error {
	ctx, cancel := context.WithTimeout(parent, defaultAutoParseJobTimeout)
	defer cancel()

	marked, err := w.repo.MarkAutoParseProcessing(ctx, item.ID, "bilibili")
	if err != nil {
		return err
	}
	if !marked {
		return nil
	}

	result, err := w.parser.ParseBilibili(ctx, item.Link)
	if err != nil {
		finishedAt := time.Now().Format(time.RFC3339)
		if markErr := w.repo.MarkAutoParseFailed(ctx, item.ID, "bilibili", err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe auto-parse failed state failed", "recipeID", item.ID, "error", markErr)
		}
		return err
	}

	finishedAt := time.Now().Format(time.RFC3339)
	parseSource := result.Source
	if strings.TrimSpace(result.SummaryMode) != "" {
		parseSource += ":" + strings.TrimSpace(result.SummaryMode)
	}

	if err := w.repo.ApplyAutoParseResult(ctx, item.ID, parseSource, finishedAt, Recipe{
		Ingredient: result.RecipeDraft.Ingredient,
		ParsedContent: ParsedContent{
			Ingredients: result.RecipeDraft.ParsedContent.Ingredients,
			Steps:       result.RecipeDraft.ParsedContent.Steps,
		},
	}); err != nil {
		if markErr := w.repo.MarkAutoParseFailed(ctx, item.ID, parseSource, err.Error(), finishedAt); markErr != nil {
			w.logger.Error("mark recipe auto-parse failure after apply error failed", "recipeID", item.ID, "error", markErr)
		}
		return err
	}

	w.logger.Info("recipe auto-parse completed", "recipeID", item.ID, "source", parseSource)
	return nil
}
