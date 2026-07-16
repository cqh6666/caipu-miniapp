package admin

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/go-chi/chi/v5"
)

type alertArchiveRequest struct {
	Reason string `json:"reason"`
}

type alertMuteRequest struct {
	DurationHours int    `json:"durationHours"`
	Reason        string `json:"reason"`
}

func (h *Handler) GetAIRoutingAlertsOverview(w http.ResponseWriter, r *http.Request) {
	if h.aiAlert == nil {
		common.WriteError(w, common.ErrInternal)
		return
	}
	if _, err := h.auth.CurrentSubject(r.Context()); err != nil {
		common.WriteError(w, err)
		return
	}
	overview, err := h.aiAlert.Overview(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"overview": overview})
}

func (h *Handler) RetestAIRoutingAlert(w http.ResponseWriter, r *http.Request) {
	subject, providerID, ok := h.alertActionContext(w, r)
	if !ok {
		return
	}
	result, err := h.aiAlert.Retest(r.Context(), providerID, subject)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": result})
}

func (h *Handler) ArchiveAIRoutingAlert(w http.ResponseWriter, r *http.Request) {
	subject, providerID, ok := h.alertActionContext(w, r)
	if !ok {
		return
	}
	var req alertArchiveRequest
	if err := common.DecodeJSONAllowEmpty(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}
	result, err := h.aiAlert.Archive(r.Context(), providerID, subject, req.Reason)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": result})
}

func (h *Handler) MuteAIRoutingAlert(w http.ResponseWriter, r *http.Request) {
	subject, providerID, ok := h.alertActionContext(w, r)
	if !ok {
		return
	}
	var req alertMuteRequest
	if err := common.DecodeJSONAllowEmpty(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}
	result, err := h.aiAlert.Mute(r.Context(), providerID, subject, req.DurationHours, req.Reason)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": result})
}

func (h *Handler) UnmuteAIRoutingAlert(w http.ResponseWriter, r *http.Request) {
	subject, providerID, ok := h.alertActionContext(w, r)
	if !ok {
		return
	}
	result, err := h.aiAlert.Unmute(r.Context(), providerID, subject)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": result})
}

func (h *Handler) alertActionContext(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	if h.aiAlert == nil {
		common.WriteError(w, common.ErrInternal)
		return "", "", false
	}
	subject, err := h.auth.CurrentSubject(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return "", "", false
	}
	providerID := strings.TrimSpace(chi.URLParam(r, "providerId"))
	if providerID == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "providerId is required", http.StatusBadRequest))
		return "", "", false
	}
	return subject, providerID, true
}
