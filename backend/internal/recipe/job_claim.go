package recipe

import (
	"context"
	"errors"
	"time"
)

var (
	ErrStaleJobResult          = errors.New("stale recipe job result")
	ErrAutoParseContentChanged = errors.New("recipe content changed while auto-parse was running")
)

func startJobLeaseRenewal(
	parent context.Context,
	leaseDuration time.Duration,
	renew func(context.Context, string) error,
	onError func(error),
) func() {
	if parent == nil || leaseDuration <= 0 || renew == nil {
		return func() {}
	}

	interval := leaseDuration / 3
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				expiresAt := now.Add(leaseDuration).Format(time.RFC3339Nano)
				if err := renew(ctx, expiresAt); err != nil && onError != nil {
					onError(err)
				}
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}
