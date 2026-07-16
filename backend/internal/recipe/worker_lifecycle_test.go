package recipe

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type manualWorkerTicker struct {
	ticks     chan time.Time
	stopOnce  sync.Once
	stoppedCh chan struct{}
}

func newManualWorkerTicker() *manualWorkerTicker {
	return &manualWorkerTicker{
		ticks:     make(chan time.Time, 1),
		stoppedCh: make(chan struct{}),
	}
}

func (t *manualWorkerTicker) C() <-chan time.Time { return t.ticks }
func (t *manualWorkerTicker) Stop() {
	t.stopOnce.Do(func() { close(t.stoppedCh) })
}

func TestWorkerLifecycleRunsImmediatelyTicksAndStops(t *testing.T) {
	lifecycle := newWorkerLifecycle()
	ticker := newManualWorkerTicker()
	tickerCreated := make(chan struct{})
	lifecycle.tickerFactory = func(time.Duration) workerTicker {
		close(tickerCreated)
		return ticker
	}

	runs := make(chan struct{}, 2)
	if err := lifecycle.Start(
		context.Background(),
		"test",
		time.Second,
		func(context.Context) { runs <- struct{}{} },
		nil,
		nil,
	); err != nil {
		t.Fatal(err)
	}
	waitWorkerSignal(t, runs, "immediate run")
	waitWorkerSignal(t, tickerCreated, "ticker creation")
	ticker.ticks <- time.Now()
	waitWorkerSignal(t, runs, "tick run")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := lifecycle.Stop(ctx, "test"); err != nil {
		t.Fatal(err)
	}
	waitWorkerSignal(t, ticker.stoppedCh, "ticker stop")
}

func TestWorkerLifecycleRejectsInvalidIntervalWithoutStarting(t *testing.T) {
	lifecycle := newWorkerLifecycle()
	var runs atomic.Int32
	err := lifecycle.Start(
		context.Background(),
		"invalid",
		0,
		func(context.Context) { runs.Add(1) },
		nil,
		nil,
	)
	if err == nil {
		t.Fatal("expected invalid interval error")
	}
	if runs.Load() != 0 {
		t.Fatalf("run count=%d, want=0", runs.Load())
	}
	if err := lifecycle.Stop(context.Background(), "invalid"); err != nil {
		t.Fatalf("stop unstarted lifecycle: %v", err)
	}
}

func TestWorkerLifecycleStopHonorsDeadlineWhenRunIgnoresCancellation(t *testing.T) {
	lifecycle := newWorkerLifecycle()
	started := make(chan struct{})
	release := make(chan struct{})
	if err := lifecycle.Start(
		context.Background(),
		"blocked",
		time.Second,
		func(context.Context) {
			close(started)
			<-release
		},
		nil,
		nil,
	); err != nil {
		t.Fatal(err)
	}
	waitWorkerSignal(t, started, "blocked run start")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	startedAt := time.Now()
	err := lifecycle.Stop(ctx, "blocked")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stop error=%v, want deadline exceeded", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 500*time.Millisecond {
		t.Fatalf("deadline stop took %s", elapsed)
	}

	close(release)
	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
	defer waitCancel()
	if err := lifecycle.Stop(waitCtx, "blocked"); err != nil {
		t.Fatalf("wait for released lifecycle: %v", err)
	}
}

func TestWorkerLifecycleConcurrentRepeatedStartStopIsIdempotent(t *testing.T) {
	lifecycle := newWorkerLifecycle()
	var runs atomic.Int32
	var starts atomic.Int32
	var stops atomic.Int32
	firstRun := make(chan struct{})
	var firstRunOnce sync.Once

	var startGroup sync.WaitGroup
	for range 32 {
		startGroup.Add(1)
		go func() {
			defer startGroup.Done()
			if err := lifecycle.Start(
				context.Background(),
				"concurrent",
				time.Hour,
				func(context.Context) {
					runs.Add(1)
					firstRunOnce.Do(func() { close(firstRun) })
				},
				func() { starts.Add(1) },
				func() { stops.Add(1) },
			); err != nil {
				t.Errorf("start: %v", err)
			}
		}()
	}
	startGroup.Wait()
	waitWorkerSignal(t, firstRun, "concurrent first run")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var stopGroup sync.WaitGroup
	for range 32 {
		stopGroup.Add(1)
		go func() {
			defer stopGroup.Done()
			if err := lifecycle.Stop(ctx, "concurrent"); err != nil {
				t.Errorf("stop: %v", err)
			}
		}()
	}
	stopGroup.Wait()

	if runs.Load() != 1 || starts.Load() != 1 || stops.Load() != 1 {
		t.Fatalf("runs=%d starts=%d stops=%d, want all 1", runs.Load(), starts.Load(), stops.Load())
	}
}

func waitWorkerSignal(t *testing.T, signal <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
}
