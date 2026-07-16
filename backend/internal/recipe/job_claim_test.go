package recipe

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestStartJobLeaseRenewalRenewsUntilStopped(t *testing.T) {
	t.Parallel()

	var renewals atomic.Int32
	renewed := make(chan struct{}, 1)
	stop := startJobLeaseRenewal(
		context.Background(),
		30*time.Millisecond,
		func(_ context.Context, expiresAt string) error {
			if _, err := time.Parse(time.RFC3339Nano, expiresAt); err != nil {
				t.Errorf("lease expiration = %q: %v", expiresAt, err)
			}
			renewals.Add(1)
			select {
			case renewed <- struct{}{}:
			default:
			}
			return nil
		},
		nil,
	)

	select {
	case <-renewed:
	case <-time.After(time.Second):
		t.Fatal("lease was not renewed")
	}
	stop()
	countAfterStop := renewals.Load()
	time.Sleep(40 * time.Millisecond)
	if got := renewals.Load(); got != countAfterStop {
		t.Fatalf("renewals continued after stop: before=%d after=%d", countAfterStop, got)
	}
}
