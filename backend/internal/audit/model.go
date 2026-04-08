package audit

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const (
	SceneParseSummary = "parse_summary"
	SceneFlowchart    = "flowchart"
	SceneTitleRefine  = "title_refine"

	JobStatusSuccess  = "success"
	JobStatusFailed   = "failed"
	JobStatusTimeout  = "timeout"
	JobStatusFallback = "fallback"

	CallStatusSuccess = "success"
	CallStatusFailed  = "failed"
	CallStatusTimeout = "timeout"
)

type FinishFunc func(context.Context, JobResult) error

type Tracker interface {
	StartJob(ctx context.Context, input JobInput) (int64, FinishFunc, error)
	LogCall(ctx context.Context, input CallLogInput) error
}

type JobInput struct {
	Scene         string
	TargetType    string
	TargetID      string
	TriggerSource string
	RequestID     string
	Meta          map[string]any
}

type JobResult struct {
	Status        string
	FinalProvider string
	FinalModel    string
	FallbackUsed  bool
	ErrorMessage  string
	FinishedAt    string
	Meta          map[string]any
}

type CallLogInput struct {
	JobRunID     int64
	Scene        string
	Provider     string
	Endpoint     string
	Model        string
	Status       string
	HTTPStatus   int
	LatencyMS    int64
	ErrorType    string
	ErrorMessage string
	RequestID    string
	CreatedAt    string
	Meta         map[string]any
}

type JobRunRecord struct {
	ID            int64  `json:"id"`
	Scene         string `json:"scene"`
	TargetType    string `json:"targetType"`
	TargetID      string `json:"targetId"`
	TriggerSource string `json:"triggerSource"`
	Status        string `json:"status"`
	FinalProvider string `json:"finalProvider"`
	FinalModel    string `json:"finalModel"`
	FallbackUsed  bool   `json:"fallbackUsed"`
	ErrorMessage  string `json:"errorMessage"`
	RequestID     string `json:"requestId"`
	StartedAt     string `json:"startedAt"`
	FinishedAt    string `json:"finishedAt"`
	DurationMS    int64  `json:"durationMs"`
	MetaJSON      string `json:"metaJson"`
}

type CallLogRecord struct {
	ID           int64  `json:"id"`
	JobRunID     int64  `json:"jobRunId"`
	Scene        string `json:"scene"`
	Provider     string `json:"provider"`
	Endpoint     string `json:"endpoint"`
	Model        string `json:"model"`
	Status       string `json:"status"`
	HTTPStatus   int    `json:"httpStatus"`
	LatencyMS    int64  `json:"latencyMs"`
	ErrorType    string `json:"errorType"`
	ErrorMessage string `json:"errorMessage"`
	RequestID    string `json:"requestId"`
	MetaJSON     string `json:"metaJson"`
	CreatedAt    string `json:"createdAt"`
}

type JobListFilter struct {
	Scene         string
	Status        string
	TriggerSource string
	TargetID      string
	TimeFrom      string
	TimeTo        string
	Page          int
	PageSize      int
}

type CallListFilter struct {
	Scene     string
	Status    string
	Provider  string
	Model     string
	RequestID string
	TimeFrom  string
	TimeTo    string
	Page      int
	PageSize  int
}

type PaginationResult[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type OverviewMetric struct {
	Name        string  `json:"name"`
	Total       int     `json:"total"`
	SuccessRate float64 `json:"successRate"`
}

type DashboardOverview struct {
	WindowHours     int              `json:"windowHours"`
	TaskTotal       int              `json:"taskTotal"`
	TaskSuccessRate float64          `json:"taskSuccessRate"`
	APITotal        int              `json:"apiTotal"`
	APISuccessRate  float64          `json:"apiSuccessRate"`
	TimeoutRate     float64          `json:"timeoutRate"`
	AvgDurationMS   int64            `json:"avgDurationMs"`
	P95DurationMS   int64            `json:"p95DurationMs"`
	ByScene         []OverviewMetric `json:"byScene"`
	ByModel         []OverviewMetric `json:"byModel"`
	ByProvider      []OverviewMetric `json:"byProvider"`
	RecentFailures  []JobRunRecord   `json:"recentFailures"`
}

type TrendBucket struct {
	Bucket          string  `json:"bucket"`
	Label           string  `json:"label"`
	TaskTotal       int     `json:"taskTotal"`
	TaskSuccessRate float64 `json:"taskSuccessRate"`
	APITotal        int     `json:"apiTotal"`
	APISuccessRate  float64 `json:"apiSuccessRate"`
	AvgDurationMS   int64   `json:"avgDurationMs"`
}

func TruncateMessage(message string, limit int) string {
	message = strings.TrimSpace(message)
	if limit <= 0 || len([]rune(message)) <= limit {
		return message
	}

	runes := []rune(message)
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func JobStatusFromError(err error) string {
	if IsTimeoutError(err) {
		return JobStatusTimeout
	}
	return JobStatusFailed
}

func CallStatusFromError(err error) string {
	if IsTimeoutError(err) {
		return CallStatusTimeout
	}
	return CallStatusFailed
}

func ErrorTypeFromError(err error) string {
	if err == nil {
		return ""
	}
	if IsTimeoutError(err) {
		return "timeout"
	}

	var appErr *common.AppError
	if errors.As(err, &appErr) {
		switch {
		case appErr.HTTPStatus == http.StatusUnauthorized || appErr.HTTPStatus == http.StatusForbidden:
			return "auth"
		case appErr.HTTPStatus >= 500:
			return "upstream"
		case appErr.HTTPStatus >= 400:
			return "bad_request"
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return "network"
	}

	return "unknown"
}

func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
