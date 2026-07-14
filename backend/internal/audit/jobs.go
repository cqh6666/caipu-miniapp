package audit

import "context"

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
