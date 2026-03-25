package recipe

import "time"

const (
	defaultAutoParseEstimatedDuration = 45 * time.Second
	defaultFlowchartEstimatedDuration = 75 * time.Second
	minEstimatedWaitDuration          = 5 * time.Second
	minEstimatedPickupDelay           = 5 * time.Second
)

type queueEstimateConfig struct {
	enabled         bool
	interval        time.Duration
	batchSize       int
	averageDuration time.Duration
}

func (c queueEstimateConfig) normalize() queueEstimateConfig {
	if c.batchSize <= 0 {
		c.batchSize = 1
	}
	if c.averageDuration <= 0 {
		c.averageDuration = 30 * time.Second
	}
	return c
}

func estimatePendingQueueWaitSeconds(cfg queueEstimateConfig, jobsAhead int) int {
	cfg = cfg.normalize()
	if !cfg.enabled {
		return 0
	}
	if jobsAhead < 0 {
		jobsAhead = 0
	}

	pickupDelay := estimatePickupDelay(cfg)
	serviceWait := time.Duration(jobsAhead+1) * cfg.averageDuration
	perBatchIdle := cfg.interval - time.Duration(cfg.batchSize)*cfg.averageDuration
	if perBatchIdle < 0 {
		perBatchIdle = 0
	}
	batchWait := time.Duration(jobsAhead/cfg.batchSize) * perBatchIdle

	return durationToEstimatedSeconds(pickupDelay + serviceWait + batchWait)
}

func estimateProcessingQueueWaitSeconds(cfg queueEstimateConfig) int {
	cfg = cfg.normalize()
	if !cfg.enabled {
		return 0
	}

	remaining := cfg.averageDuration / 2
	if remaining < minEstimatedWaitDuration {
		remaining = minEstimatedWaitDuration
	}
	return durationToEstimatedSeconds(remaining)
}

func estimatePickupDelay(cfg queueEstimateConfig) time.Duration {
	cfg = cfg.normalize()
	if cfg.interval <= 0 {
		return minEstimatedPickupDelay
	}

	delay := cfg.interval / 2
	if delay < minEstimatedPickupDelay {
		delay = minEstimatedPickupDelay
	}
	if delay > cfg.averageDuration {
		delay = cfg.averageDuration
	}
	return delay
}

func durationToEstimatedSeconds(value time.Duration) int {
	if value <= 0 {
		return 0
	}
	if value < minEstimatedWaitDuration {
		value = minEstimatedWaitDuration
	}
	return int((value + time.Second - 1) / time.Second)
}
