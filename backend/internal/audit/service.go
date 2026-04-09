package audit

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"
)

type Service struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewService(db *sql.DB, logger *slog.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

func (s *Service) StartJob(ctx context.Context, input JobInput) (int64, FinishFunc, error) {
	if s == nil || s.db == nil {
		return 0, func(context.Context, JobResult) error { return nil }, nil
	}

	startedAt := NowRFC3339()
	result, err := s.db.ExecContext(ctx, `
INSERT INTO ai_job_runs (
	scene,
	target_type,
	target_id,
	trigger_source,
	status,
	final_provider,
	final_model,
	fallback_used,
	error_message,
	request_id,
	started_at,
	finished_at,
	duration_ms,
	meta_json
) VALUES (?, ?, ?, ?, '', '', '', 0, '', ?, ?, '', 0, ?)
`,
		strings.TrimSpace(input.Scene),
		strings.TrimSpace(input.TargetType),
		strings.TrimSpace(input.TargetID),
		strings.TrimSpace(input.TriggerSource),
		strings.TrimSpace(input.RequestID),
		startedAt,
		EncodeMeta(input.Meta),
	)
	if err != nil {
		return 0, nil, err
	}

	jobID, err := result.LastInsertId()
	if err != nil {
		return 0, nil, err
	}

	finish := func(ctx context.Context, result JobResult) error {
		if s == nil || s.db == nil {
			return nil
		}

		finishedAt := strings.TrimSpace(result.FinishedAt)
		if finishedAt == "" {
			finishedAt = NowRFC3339()
		}

		startedTime, parseErr := time.Parse(time.RFC3339, startedAt)
		if parseErr != nil {
			startedTime = time.Now().UTC()
		}
		finishedTime, parseErr := time.Parse(time.RFC3339, finishedAt)
		if parseErr != nil {
			finishedTime = time.Now().UTC()
			finishedAt = finishedTime.Format(time.RFC3339)
		}
		durationMS := finishedTime.Sub(startedTime).Milliseconds()
		if durationMS < 0 {
			durationMS = 0
		}

		_, err := s.db.ExecContext(ctx, `
UPDATE ai_job_runs
SET
	status = ?,
	final_provider = ?,
	final_model = ?,
	fallback_used = ?,
	error_message = ?,
	finished_at = ?,
	duration_ms = ?,
	meta_json = ?
WHERE id = ?
`,
			strings.TrimSpace(result.Status),
			strings.TrimSpace(result.FinalProvider),
			strings.TrimSpace(result.FinalModel),
			boolToInt(result.FallbackUsed),
			TruncateMessage(result.ErrorMessage, 240),
			finishedAt,
			durationMS,
			EncodeMeta(result.Meta),
			jobID,
		)
		return err
	}

	return jobID, finish, nil
}

func (s *Service) LogCall(ctx context.Context, input CallLogInput) error {
	if s == nil || s.db == nil {
		return nil
	}

	createdAt := strings.TrimSpace(input.CreatedAt)
	if createdAt == "" {
		createdAt = NowRFC3339()
	}

	var jobRunID any
	if input.JobRunID > 0 {
		jobRunID = input.JobRunID
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO ai_call_logs (
	job_run_id,
	scene,
	provider,
	endpoint,
	model,
	status,
	http_status,
	latency_ms,
	error_type,
	error_message,
	request_id,
	meta_json,
	created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		jobRunID,
		strings.TrimSpace(input.Scene),
		strings.TrimSpace(input.Provider),
		strings.TrimSpace(input.Endpoint),
		strings.TrimSpace(input.Model),
		strings.TrimSpace(input.Status),
		input.HTTPStatus,
		input.LatencyMS,
		strings.TrimSpace(input.ErrorType),
		TruncateMessage(input.ErrorMessage, 240),
		strings.TrimSpace(input.RequestID),
		EncodeMeta(input.Meta),
		createdAt,
	)
	return err
}

func (s *Service) ListJobs(ctx context.Context, filter JobListFilter) (PaginationResult[JobRunRecord], error) {
	filter.Page, filter.PageSize = normalizePagination(filter.Page, filter.PageSize)
	where, args := buildJobWhere(filter)

	total, err := s.queryCount(ctx, "SELECT COUNT(*) FROM ai_job_runs"+where, args...)
	if err != nil {
		return PaginationResult[JobRunRecord]{}, err
	}

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.db.QueryContext(ctx, `
SELECT
	id,
	scene,
	target_type,
	target_id,
	trigger_source,
	status,
	final_provider,
	final_model,
	fallback_used,
	error_message,
	request_id,
	started_at,
	finished_at,
	duration_ms,
	meta_json
FROM ai_job_runs`+where+`
ORDER BY started_at DESC, id DESC
LIMIT ? OFFSET ?
`, args...)
	if err != nil {
		return PaginationResult[JobRunRecord]{}, err
	}
	defer rows.Close()

	items := make([]JobRunRecord, 0, filter.PageSize)
	for rows.Next() {
		var item JobRunRecord
		var fallbackUsed int
		if err := rows.Scan(
			&item.ID,
			&item.Scene,
			&item.TargetType,
			&item.TargetID,
			&item.TriggerSource,
			&item.Status,
			&item.FinalProvider,
			&item.FinalModel,
			&fallbackUsed,
			&item.ErrorMessage,
			&item.RequestID,
			&item.StartedAt,
			&item.FinishedAt,
			&item.DurationMS,
			&item.MetaJSON,
		); err != nil {
			return PaginationResult[JobRunRecord]{}, err
		}
		item.FallbackUsed = fallbackUsed == 1
		items = append(items, item)
	}

	return PaginationResult[JobRunRecord]{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, rows.Err()
}

func (s *Service) GetJobDetail(ctx context.Context, id int64) (JobRunRecord, []CallLogRecord, error) {
	var item JobRunRecord
	var fallbackUsed int
	err := s.db.QueryRowContext(ctx, `
SELECT
	id,
	scene,
	target_type,
	target_id,
	trigger_source,
	status,
	final_provider,
	final_model,
	fallback_used,
	error_message,
	request_id,
	started_at,
	finished_at,
	duration_ms,
	meta_json
FROM ai_job_runs
WHERE id = ?
LIMIT 1
`, id).Scan(
		&item.ID,
		&item.Scene,
		&item.TargetType,
		&item.TargetID,
		&item.TriggerSource,
		&item.Status,
		&item.FinalProvider,
		&item.FinalModel,
		&fallbackUsed,
		&item.ErrorMessage,
		&item.RequestID,
		&item.StartedAt,
		&item.FinishedAt,
		&item.DurationMS,
		&item.MetaJSON,
	)
	if err != nil {
		return JobRunRecord{}, nil, err
	}
	item.FallbackUsed = fallbackUsed == 1

	rows, err := s.db.QueryContext(ctx, `
SELECT
	id,
	COALESCE(job_run_id, 0),
	scene,
	provider,
	endpoint,
	model,
	status,
	http_status,
	latency_ms,
	error_type,
	error_message,
	request_id,
	meta_json,
	created_at
FROM ai_call_logs
WHERE job_run_id = ?
ORDER BY created_at ASC, id ASC
`, id)
	if err != nil {
		return JobRunRecord{}, nil, err
	}
	defer rows.Close()

	calls := make([]CallLogRecord, 0, 4)
	for rows.Next() {
		var item CallLogRecord
		if err := rows.Scan(
			&item.ID,
			&item.JobRunID,
			&item.Scene,
			&item.Provider,
			&item.Endpoint,
			&item.Model,
			&item.Status,
			&item.HTTPStatus,
			&item.LatencyMS,
			&item.ErrorType,
			&item.ErrorMessage,
			&item.RequestID,
			&item.MetaJSON,
			&item.CreatedAt,
		); err != nil {
			return JobRunRecord{}, nil, err
		}
		calls = append(calls, item)
	}
	return item, calls, rows.Err()
}

func (s *Service) ListCalls(ctx context.Context, filter CallListFilter) (PaginationResult[CallLogRecord], error) {
	filter.Page, filter.PageSize = normalizePagination(filter.Page, filter.PageSize)
	where, args := buildCallWhere(filter)

	total, err := s.queryCount(ctx, "SELECT COUNT(*) FROM ai_call_logs"+where, args...)
	if err != nil {
		return PaginationResult[CallLogRecord]{}, err
	}

	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.db.QueryContext(ctx, `
SELECT
	id,
	COALESCE(job_run_id, 0),
	scene,
	provider,
	endpoint,
	model,
	status,
	http_status,
	latency_ms,
	error_type,
	error_message,
	request_id,
	meta_json,
	created_at
FROM ai_call_logs`+where+`
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?
`, args...)
	if err != nil {
		return PaginationResult[CallLogRecord]{}, err
	}
	defer rows.Close()

	items := make([]CallLogRecord, 0, filter.PageSize)
	for rows.Next() {
		var item CallLogRecord
		if err := rows.Scan(
			&item.ID,
			&item.JobRunID,
			&item.Scene,
			&item.Provider,
			&item.Endpoint,
			&item.Model,
			&item.Status,
			&item.HTTPStatus,
			&item.LatencyMS,
			&item.ErrorType,
			&item.ErrorMessage,
			&item.RequestID,
			&item.MetaJSON,
			&item.CreatedAt,
		); err != nil {
			return PaginationResult[CallLogRecord]{}, err
		}
		items = append(items, item)
	}

	return PaginationResult[CallLogRecord]{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, rows.Err()
}

func (s *Service) Overview(ctx context.Context) (DashboardOverview, error) {
	since := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	overview := DashboardOverview{
		WindowHours: 24,
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

func buildJobWhere(filter JobListFilter) (string, []any) {
	parts := make([]string, 0, 5)
	args := make([]any, 0, 5)
	if value := strings.TrimSpace(filter.Scene); value != "" {
		parts = append(parts, "scene = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.Status); value != "" {
		parts = append(parts, "status = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.TriggerSource); value != "" {
		parts = append(parts, "trigger_source = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.TargetID); value != "" {
		parts = append(parts, "target_id = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.TimeFrom); value != "" {
		parts = append(parts, "started_at >= ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.TimeTo); value != "" {
		parts = append(parts, "started_at <= ?")
		args = append(args, value)
	}
	if len(parts) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func buildCallWhere(filter CallListFilter) (string, []any) {
	parts := make([]string, 0, 6)
	args := make([]any, 0, 6)
	if value := strings.TrimSpace(filter.Scene); value != "" {
		parts = append(parts, "scene = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.Status); value != "" {
		parts = append(parts, "status = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.Provider); value != "" {
		parts = append(parts, "provider = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.Model); value != "" {
		parts = append(parts, "model = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.RequestID); value != "" {
		parts = append(parts, "request_id = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.TimeFrom); value != "" {
		parts = append(parts, "created_at >= ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.TimeTo); value != "" {
		parts = append(parts, "created_at <= ?")
		args = append(args, value)
	}
	if len(parts) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(parts, " AND "), args
}

func normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func percentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}
	index := int(float64(len(values)-1) * p)
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func roundDurationMS(value float64) int64 {
	if value <= 0 {
		return 0
	}
	return int64(math.Round(value))
}

type trendRange struct {
	since       string
	jobField    string
	jobSelect   string
	callField   string
	callSelect  string
	formatLabel func(string) string
}

func normalizeTrendRange(window string) trendRange {
	switch strings.TrimSpace(strings.ToLower(window)) {
	case "7d":
		since := time.Now().UTC().Add(-7 * 24 * time.Hour)
		return trendRange{
			since:       since.Format(time.RFC3339),
			jobField:    "started_at",
			jobSelect:   "strftime('%Y-%m-%d', started_at)",
			callField:   "created_at",
			callSelect:  "strftime('%Y-%m-%d', created_at)",
			formatLabel: func(value string) string { return value },
		}
	case "30d":
		since := time.Now().UTC().Add(-30 * 24 * time.Hour)
		return trendRange{
			since:       since.Format(time.RFC3339),
			jobField:    "started_at",
			jobSelect:   "strftime('%Y-%m-%d', started_at)",
			callField:   "created_at",
			callSelect:  "strftime('%Y-%m-%d', created_at)",
			formatLabel: func(value string) string { return value },
		}
	default:
		since := time.Now().UTC().Add(-24 * time.Hour)
		return trendRange{
			since:      since.Format(time.RFC3339),
			jobField:   "started_at",
			jobSelect:  "strftime('%Y-%m-%dT%H:00:00Z', started_at)",
			callField:  "created_at",
			callSelect: "strftime('%Y-%m-%dT%H:00:00Z', created_at)",
			formatLabel: func(value string) string {
				timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
				if err != nil {
					return value
				}
				return timestamp.Format("01-02 15:04")
			},
		}
	}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
