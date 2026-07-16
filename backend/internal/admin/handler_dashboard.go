package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) DashboardOverview(w http.ResponseWriter, r *http.Request) {
	windowHours := 0
	if raw := r.URL.Query().Get("windowHours"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			windowHours = parsed
		}
	}
	overview, err := h.audit.Overview(r.Context(), windowHours)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"overview": overview})
}

func (h *Handler) DashboardFailures(w http.ResponseWriter, r *http.Request) {
	failed, err := h.audit.ListJobs(r.Context(), audit.JobListFilter{
		Status: audit.JobStatusFailed, Page: parseIntQuery(r, "page", 1), PageSize: parseIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": failed})
}

func (h *Handler) DashboardTrends(w http.ResponseWriter, r *http.Request) {
	items, err := h.audit.Trends(r.Context(), r.URL.Query().Get("range"))
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ServerHealthOverview(w http.ResponseWriter, r *http.Request) {
	if h.serverHealth == nil {
		common.WriteError(w, common.ErrInternal)
		return
	}
	overview, err := h.serverHealth.Overview(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"overview": overview})
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	result, err := h.audit.ListJobs(r.Context(), audit.JobListFilter{
		Scene:         r.URL.Query().Get("scene"),
		Status:        r.URL.Query().Get("status"),
		TriggerSource: r.URL.Query().Get("triggerSource"),
		TargetID:      r.URL.Query().Get("targetId"),
		TimeFrom:      r.URL.Query().Get("timeFrom"),
		TimeTo:        r.URL.Query().Get("timeTo"),
		Page:          parseIntQuery(r, "page", 1),
		PageSize:      parseIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": result})
}

func (h *Handler) GetJobDetail(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
	if err != nil || jobID <= 0 {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "invalid job id", http.StatusBadRequest))
		return
	}
	job, calls, err := h.audit.GetJobDetail(r.Context(), jobID)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"job": job, "calls": calls})
}

func (h *Handler) ListCalls(w http.ResponseWriter, r *http.Request) {
	result, err := h.audit.ListCalls(r.Context(), audit.CallListFilter{
		Scene:     r.URL.Query().Get("scene"),
		Status:    r.URL.Query().Get("status"),
		Provider:  r.URL.Query().Get("provider"),
		Model:     r.URL.Query().Get("model"),
		RequestID: r.URL.Query().Get("requestId"),
		TimeFrom:  r.URL.Query().Get("timeFrom"),
		TimeTo:    r.URL.Query().Get("timeTo"),
		Page:      parseIntQuery(r, "page", 1),
		PageSize:  parseIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": result})
}

func parseIntQuery(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
