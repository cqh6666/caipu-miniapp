package audit

import "context"

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
