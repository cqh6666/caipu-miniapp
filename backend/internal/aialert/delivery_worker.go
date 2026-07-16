package aialert

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	defaultDeliveryBatchSize = 10
	defaultDeliveryTimeout   = 15 * time.Second
	defaultDeliveryRetryBase = 30 * time.Second
	maxDeliveryRetryDelay    = 15 * time.Minute
)

type deliveryWorker struct {
	service  *Service
	interval time.Duration

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	done    chan struct{}
}

func newDeliveryWorker(service *Service, interval time.Duration) *deliveryWorker {
	return &deliveryWorker{service: service, interval: interval}
}

func (s *Service) Start(parent context.Context) error {
	if s == nil || s.deliveryWorker == nil || s.repo == nil {
		return nil
	}
	return s.deliveryWorker.Start(parent)
}

func (s *Service) Stop(ctx context.Context) error {
	if s == nil || s.deliveryWorker == nil {
		return nil
	}
	return s.deliveryWorker.Stop(ctx)
}

func (w *deliveryWorker) Start(parent context.Context) error {
	if w == nil || w.service == nil {
		return nil
	}
	if w.interval <= 0 {
		return fmt.Errorf("AI alert delivery worker interval must be positive")
	}
	if parent == nil {
		parent = context.Background()
	}

	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return nil
	}
	w.started = true
	ctx, cancel := context.WithCancel(parent)
	w.cancel = cancel
	w.done = make(chan struct{})
	done := w.done
	w.mu.Unlock()

	go func() {
		defer close(done)
		w.service.logInfo("AI alert delivery worker started", "interval", w.interval.String())
		defer w.service.logInfo("AI alert delivery worker stopped")

		w.run(ctx)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.run(ctx)
			}
		}
	}()
	return nil
}

func (w *deliveryWorker) run(ctx context.Context) {
	if _, err := w.service.DispatchPending(ctx, defaultDeliveryBatchSize); err != nil && ctx.Err() == nil {
		w.service.logWarn("dispatch queued provider alerts failed", Event{}, err)
	}
}

func (w *deliveryWorker) Stop(ctx context.Context) error {
	if w == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	w.mu.Lock()
	if !w.started {
		w.mu.Unlock()
		return nil
	}
	cancel := w.cancel
	done := w.done
	w.mu.Unlock()

	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop AI alert delivery worker: %w", ctx.Err())
	}
}

func (s *Service) DispatchPending(ctx context.Context, limit int) (int, error) {
	if s == nil || s.repo == nil {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if limit <= 0 {
		limit = defaultDeliveryBatchSize
	}

	dispatched := 0
	for dispatched < limit {
		delivery, found, err := s.repo.ClaimNextDelivery(ctx, deliveryClaimLease)
		if err != nil {
			return dispatched, err
		}
		if !found {
			return dispatched, nil
		}
		if err := s.dispatchDelivery(ctx, delivery); err != nil {
			return dispatched, err
		}
		dispatched++
	}
	return dispatched, nil
}

func (s *Service) dispatchDelivery(ctx context.Context, delivery Delivery) error {
	state, found, err := s.repo.GetState(ctx, delivery.ProviderID)
	if err != nil {
		return s.releaseDelivery(ctx, delivery, err)
	}
	if !found || state.FailureStreakID != delivery.FailureStreakID || state.ConsecutiveFailures == 0 || state.LastStatus != "failed" {
		return s.repo.MarkDeliveryCancelled(ctx, delivery, "failure streak is no longer active")
	}

	cfg := Config{}
	if s.configProvider != nil {
		cfg = s.configProvider.AIProviderAlert(ctx)
	}
	cfg = cfg.Normalized()
	if !cfg.Enabled {
		return s.releaseDelivery(ctx, delivery, fmt.Errorf("AI provider alert delivery is disabled"))
	}
	if err := cfg.ValidateForSend(); err != nil {
		return s.releaseDelivery(ctx, delivery, err)
	}
	if s.sender == nil {
		return s.releaseDelivery(ctx, delivery, fmt.Errorf("provider alert sender is not configured"))
	}

	recentFailures, err := s.repo.ListRecentFailures(ctx, state.ProviderID, 3)
	if err != nil {
		s.logWarn("list provider recent failures failed", Event{ProviderID: state.ProviderID, Scene: state.Scene}, err)
		recentFailures = nil
	}
	event := Event{
		Scene:         delivery.Scene,
		ProviderID:    delivery.ProviderID,
		ProviderName:  state.ProviderName,
		Model:         state.Model,
		HTTPStatus:    state.LastHTTPStatus,
		ErrorType:     state.LastErrorType,
		ErrorMessage:  state.LastErrorMessage,
		RequestID:     delivery.RequestID,
		TriggerSource: delivery.TriggerSource,
		TargetType:    delivery.TargetType,
		TargetID:      delivery.TargetID,
	}
	subject, body := BuildFailureAlertMessage(state, event, recentFailures, cfg.FailureThreshold)
	sendCtx, cancel := context.WithTimeout(ctx, defaultDeliveryTimeout)
	defer cancel()
	if err := s.sender.Send(sendCtx, SendRequest{Config: cfg, Subject: subject, Body: body}); err != nil {
		s.logWarn("send provider alert email failed", event, err)
		return s.releaseDelivery(ctx, delivery, err)
	}
	if err := s.repo.MarkDeliverySent(ctx, delivery, state.ConsecutiveFailures, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("persist provider alert delivery result: %w", err)
	}
	return nil
}

func (s *Service) releaseDelivery(ctx context.Context, delivery Delivery, cause error) error {
	delay := deliveryRetryDelay(delivery.AttemptCount)
	if err := s.repo.MarkDeliveryFailed(ctx, delivery, cause, delay); err != nil {
		return fmt.Errorf("release provider alert delivery: %w", err)
	}
	return nil
}

func deliveryRetryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return defaultDeliveryRetryBase
	}
	delay := defaultDeliveryRetryBase
	for current := 1; current < attempt && delay < maxDeliveryRetryDelay; current++ {
		delay *= 2
		if delay > maxDeliveryRetryDelay {
			return maxDeliveryRetryDelay
		}
	}
	return delay
}

func (s *Service) logInfo(message string, args ...any) {
	if s != nil && s.logger != nil {
		s.logger.Info(message, args...)
	}
}
