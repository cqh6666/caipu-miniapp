package recipe

import (
	"testing"
	"time"
)

func TestEstimatePendingQueueWaitSecondsIncludesPickupDelay(t *testing.T) {
	t.Parallel()

	got := estimatePendingQueueWaitSeconds(queueEstimateConfig{
		enabled:         true,
		interval:        10 * time.Second,
		batchSize:       3,
		averageDuration: 45 * time.Second,
	}, 0)

	if got != 50 {
		t.Fatalf("estimatePendingQueueWaitSeconds() = %d, want %d", got, 50)
	}
}

func TestEstimatePendingQueueWaitSecondsIncludesJobsAhead(t *testing.T) {
	t.Parallel()

	got := estimatePendingQueueWaitSeconds(queueEstimateConfig{
		enabled:         true,
		interval:        10 * time.Second,
		batchSize:       2,
		averageDuration: 30 * time.Second,
	}, 3)

	if got != 125 {
		t.Fatalf("estimatePendingQueueWaitSeconds() = %d, want %d", got, 125)
	}
}

func TestEstimateProcessingQueueWaitSecondsUsesHalfAverageDuration(t *testing.T) {
	t.Parallel()

	got := estimateProcessingQueueWaitSeconds(queueEstimateConfig{
		enabled:         true,
		interval:        10 * time.Second,
		batchSize:       1,
		averageDuration: 80 * time.Second,
	})

	if got != 40 {
		t.Fatalf("estimateProcessingQueueWaitSeconds() = %d, want %d", got, 40)
	}
}
