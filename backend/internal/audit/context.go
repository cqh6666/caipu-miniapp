package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

type requestMetaContextKey string

const (
	requestMetaKey requestMetaContextKey = "auditRequestMeta"
	jobContextKey  requestMetaContextKey = "auditJobContext"
)

type RequestMeta struct {
	TriggerSource string
	TargetType    string
	TargetID      string
	Meta          map[string]any
}

type JobContext struct {
	Scene    string
	JobRunID int64
}

func WithRequestMeta(ctx context.Context, meta RequestMeta) context.Context {
	return context.WithValue(ctx, requestMetaKey, meta)
}

func CurrentRequestMeta(ctx context.Context) (RequestMeta, bool) {
	meta, ok := ctx.Value(requestMetaKey).(RequestMeta)
	return meta, ok
}

func WithJobContext(ctx context.Context, scene string, jobRunID int64) context.Context {
	return context.WithValue(ctx, jobContextKey, JobContext{
		Scene:    strings.TrimSpace(scene),
		JobRunID: jobRunID,
	})
}

func CurrentJobContext(ctx context.Context) (JobContext, bool) {
	meta, ok := ctx.Value(jobContextKey).(JobContext)
	return meta, ok
}

func HashTargetID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:24]
}

func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func EncodeMeta(meta map[string]any) string {
	if len(meta) == 0 {
		return "{}"
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return "{}"
	}

	return string(data)
}
