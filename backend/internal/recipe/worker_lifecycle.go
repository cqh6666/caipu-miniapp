package recipe

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type workerTicker interface {
	C() <-chan time.Time
	Stop()
}

type realWorkerTicker struct {
	*time.Ticker
}

func (t realWorkerTicker) C() <-chan time.Time {
	return t.Ticker.C
}

type workerLifecycle struct {
	mu            sync.Mutex
	startCalled   bool
	cancel        context.CancelFunc
	done          chan struct{}
	tickerFactory func(time.Duration) workerTicker
}

func newWorkerLifecycle() *workerLifecycle {
	return &workerLifecycle{
		tickerFactory: func(interval time.Duration) workerTicker {
			return realWorkerTicker{Ticker: time.NewTicker(interval)}
		},
	}
}

func (l *workerLifecycle) Start(
	parent context.Context,
	name string,
	interval time.Duration,
	run func(context.Context),
	onStart func(),
	onStop func(),
) error {
	if l == nil {
		return fmt.Errorf("%s worker lifecycle is nil", name)
	}
	if interval <= 0 {
		return fmt.Errorf("%s worker interval must be positive", name)
	}
	if run == nil {
		return fmt.Errorf("%s worker run function is nil", name)
	}
	if parent == nil {
		parent = context.Background()
	}

	l.mu.Lock()
	if l.startCalled {
		l.mu.Unlock()
		return nil
	}
	l.startCalled = true
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	l.cancel = cancel
	l.done = done
	tickerFactory := l.tickerFactory
	l.mu.Unlock()

	go func() {
		defer close(done)
		if onStart != nil {
			onStart()
		}
		if onStop != nil {
			defer onStop()
		}

		if ctx.Err() == nil {
			run(ctx)
		}
		if ctx.Err() != nil {
			return
		}

		ticker := tickerFactory(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C():
				run(ctx)
			}
		}
	}()

	return nil
}

func (l *workerLifecycle) Stop(ctx context.Context, name string) error {
	if l == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	l.mu.Lock()
	if !l.startCalled {
		l.mu.Unlock()
		return nil
	}
	cancel := l.cancel
	done := l.done
	l.mu.Unlock()

	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop %s worker: %w", name, ctx.Err())
	}
}
