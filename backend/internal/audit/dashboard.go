package audit

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *Service) Overview(ctx context.Context, windowHours int) (DashboardOverview, error) {
	if windowHours <= 0 {
		windowHours = 7 * 24
	}
	if windowHours > 30*24 {
		windowHours = 30 * 24
	}
	since := time.Now().UTC().Add(-time.Duration(windowHours) * time.Hour).Format(time.RFC3339)
	overview := DashboardOverview{
		WindowHours: windowHours,
	}

	var (
		taskSuccesses, apiSuccesses, apiTimeouts int
		avgDurationMS                            float64
	)
	if err := s.db.QueryRowContext(ctx, `
SELECT
	COUNT(*),
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
	COALESCE(AVG(CASE WHEN duration_ms > 0 THEN duration_ms END), 0)
FROM ai_job_runs
WHERE started_at >= ?
`, JobStatusSuccess, since).Scan(&overview.TaskTotal, &taskSuccesses, &avgDurationMS); err != nil {
		return DashboardOverview{}, err
	}
	overview.AvgDurationMS = roundDurationMS(avgDurationMS)
	if overview.TaskTotal > 0 {
		overview.TaskSuccessRate = float64(taskSuccesses) / float64(overview.TaskTotal)
	}

	if err := s.db.QueryRowContext(ctx, `
SELECT
	COUNT(*),
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0),
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0)
FROM ai_call_logs
WHERE created_at >= ?
`, CallStatusSuccess, CallStatusTimeout, since).Scan(&overview.APITotal, &apiSuccesses, &apiTimeouts); err != nil {
		return DashboardOverview{}, err
	}
	if overview.APITotal > 0 {
		overview.APISuccessRate = float64(apiSuccesses) / float64(overview.APITotal)
		overview.TimeoutRate = float64(apiTimeouts) / float64(overview.APITotal)
	}

	durations, err := s.listDurations(ctx, since)
	if err != nil {
		return DashboardOverview{}, err
	}
	overview.P95DurationMS = percentile(durations, 0.95)

	if overview.ByScene, err = s.listDistribution(ctx, `
SELECT scene, COUNT(*) AS total, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS success_count
FROM ai_job_runs
WHERE started_at >= ?
GROUP BY scene
ORDER BY total DESC, scene ASC
`, JobStatusSuccess, since); err != nil {
		return DashboardOverview{}, err
	}

	if overview.ByModel, err = s.listDistribution(ctx, `
SELECT COALESCE(model, '') AS name, COUNT(*) AS total, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS success_count
FROM ai_call_logs
WHERE created_at >= ?
GROUP BY model
ORDER BY total DESC, model ASC
LIMIT 10
`, CallStatusSuccess, since); err != nil {
		return DashboardOverview{}, err
	}

	if overview.ByProvider, err = s.listDistribution(ctx, `
SELECT COALESCE(provider, '') AS name, COUNT(*) AS total, COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS success_count
FROM ai_call_logs
WHERE created_at >= ?
GROUP BY provider
ORDER BY total DESC, provider ASC
LIMIT 10
`, CallStatusSuccess, since); err != nil {
		return DashboardOverview{}, err
	}

	failures, err := s.ListJobs(ctx, JobListFilter{
		Page:     1,
		PageSize: 10,
		Status:   JobStatusFailed,
		TimeFrom: since,
	})
	if err != nil {
		return DashboardOverview{}, err
	}

	timeouts, err := s.ListJobs(ctx, JobListFilter{
		Page:     1,
		PageSize: 10,
		Status:   JobStatusTimeout,
		TimeFrom: since,
	})
	if err != nil {
		return DashboardOverview{}, err
	}
	overview.RecentFailures = append(failures.Items, timeouts.Items...)
	sort.SliceStable(overview.RecentFailures, func(i, j int) bool {
		return overview.RecentFailures[i].StartedAt > overview.RecentFailures[j].StartedAt
	})
	if len(overview.RecentFailures) > 10 {
		overview.RecentFailures = overview.RecentFailures[:10]
	}

	return overview, nil
}

func (s *Service) Trends(ctx context.Context, window string) ([]TrendBucket, error) {
	rangeSpec := normalizeTrendRange(window)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT
	%s AS bucket,
	COUNT(*) AS total,
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS success_count,
	COALESCE(AVG(CASE WHEN duration_ms > 0 THEN duration_ms END), 0) AS avg_duration_ms
FROM ai_job_runs
WHERE started_at >= ?
GROUP BY bucket
ORDER BY bucket ASC
`, rangeSpec.jobSelect), JobStatusSuccess, rangeSpec.since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobMap := make(map[string]TrendBucket)
	for rows.Next() {
		var bucket string
		var total, successCount int
		var avgDuration float64
		if err := rows.Scan(&bucket, &total, &successCount, &avgDuration); err != nil {
			return nil, err
		}
		item := jobMap[bucket]
		item.Bucket = bucket
		item.Label = rangeSpec.formatLabel(bucket)
		item.TaskTotal = total
		item.AvgDurationMS = roundDurationMS(avgDuration)
		if total > 0 {
			item.TaskSuccessRate = float64(successCount) / float64(total)
		}
		jobMap[bucket] = item
	}

	callRows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT
	%s AS bucket,
	COUNT(*) AS total,
	COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS success_count
FROM ai_call_logs
WHERE created_at >= ?
GROUP BY bucket
ORDER BY bucket ASC
`, rangeSpec.callSelect), CallStatusSuccess, rangeSpec.since)
	if err != nil {
		return nil, err
	}
	defer callRows.Close()

	for callRows.Next() {
		var bucket string
		var total, successCount int
		if err := callRows.Scan(&bucket, &total, &successCount); err != nil {
			return nil, err
		}
		item := jobMap[bucket]
		item.Bucket = bucket
		item.Label = rangeSpec.formatLabel(bucket)
		item.APITotal = total
		if total > 0 {
			item.APISuccessRate = float64(successCount) / float64(total)
		}
		jobMap[bucket] = item
	}

	items := make([]TrendBucket, 0, len(jobMap))
	for _, item := range jobMap {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Bucket < items[j].Bucket
	})

	return items, nil
}

func (s *Service) queryCount(ctx context.Context, query string, args ...any) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

func (s *Service) listDurations(ctx context.Context, since string) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT duration_ms
FROM ai_job_runs
WHERE started_at >= ? AND duration_ms > 0
ORDER BY duration_ms ASC
`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]int64, 0, 32)
	for rows.Next() {
		var value int64
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	return items, rows.Err()
}

func (s *Service) listDistribution(ctx context.Context, query string, successStatus string, since string) ([]OverviewMetric, error) {
	rows, err := s.db.QueryContext(ctx, query, successStatus, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OverviewMetric, 0, 8)
	for rows.Next() {
		var name string
		var total int
		var successCount int
		if err := rows.Scan(&name, &total, &successCount); err != nil {
			return nil, err
		}
		item := OverviewMetric{
			Name:  strings.TrimSpace(name),
			Total: total,
		}
		if item.Name == "" {
			item.Name = "(empty)"
		}
		if total > 0 {
			item.SuccessRate = float64(successCount) / float64(total)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
