package audit

import (
	"context"
	"database/sql"
	"log/slog"
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
