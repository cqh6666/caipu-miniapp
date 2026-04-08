package linkparse

import (
	"context"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func (c *aiClient) logCall(ctx context.Context, startedAt time.Time, endpoint string, status string, httpStatus int, err error, meta map[string]any) {
	if c == nil || c.tracker == nil {
		return
	}
	jobCtx, ok := audit.CurrentJobContext(ctx)
	if !ok || jobCtx.JobRunID <= 0 {
		return
	}
	_ = c.tracker.LogCall(ctx, audit.CallLogInput{
		JobRunID:     jobCtx.JobRunID,
		Scene:        jobCtx.Scene,
		Provider:     "openai-compatible",
		Endpoint:     endpoint,
		Model:        c.model,
		Status:       status,
		HTTPStatus:   httpStatus,
		LatencyMS:    time.Since(startedAt).Milliseconds(),
		ErrorType:    audit.ErrorTypeFromError(err),
		ErrorMessage: callErrorMessage(err),
		RequestID:    common.RequestID(ctx),
		Meta:         meta,
	})
}

func callErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
