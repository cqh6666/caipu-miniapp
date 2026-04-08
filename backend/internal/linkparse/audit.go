package linkparse

import (
	"context"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func (s *Service) startTrackedJob(ctx context.Context, scene, rawInput, defaultTargetType string, meta map[string]any) (context.Context, int64, audit.FinishFunc) {
	if s == nil || s.tracker == nil {
		return ctx, 0, func(context.Context, audit.JobResult) error { return nil }
	}

	requestMeta, ok := audit.CurrentRequestMeta(ctx)
	if !ok {
		requestMeta = audit.RequestMeta{
			TriggerSource: "manual",
			TargetType:    defaultTargetType,
			TargetID:      audit.HashTargetID(rawInput),
		}
	}
	if strings.TrimSpace(requestMeta.TriggerSource) == "" {
		requestMeta.TriggerSource = "manual"
	}
	if strings.TrimSpace(requestMeta.TargetType) == "" {
		requestMeta.TargetType = defaultTargetType
	}
	if strings.TrimSpace(requestMeta.TargetID) == "" {
		requestMeta.TargetID = audit.HashTargetID(rawInput)
	}

	jobMeta := cloneMeta(requestMeta.Meta)
	for key, value := range meta {
		jobMeta[key] = value
	}

	jobID, finish, err := s.tracker.StartJob(ctx, audit.JobInput{
		Scene:         scene,
		TargetType:    requestMeta.TargetType,
		TargetID:      requestMeta.TargetID,
		TriggerSource: requestMeta.TriggerSource,
		RequestID:     common.RequestID(ctx),
		Meta:          jobMeta,
	})
	if err != nil {
		return ctx, 0, func(context.Context, audit.JobResult) error { return nil }
	}
	return audit.WithJobContext(ctx, scene, jobID), jobID, finish
}

func cloneMeta(meta map[string]any) map[string]any {
	if len(meta) == 0 {
		return make(map[string]any)
	}
	cloned := make(map[string]any, len(meta))
	for key, value := range meta {
		cloned[key] = value
	}
	return cloned
}
