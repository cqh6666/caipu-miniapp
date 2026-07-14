package audit

import (
	"math"
	"strings"
	"time"
)

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
