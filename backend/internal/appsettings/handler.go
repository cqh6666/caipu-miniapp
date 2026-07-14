package appsettings

import (
	"net/http"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Handler struct {
	service *Service
	runtime *RuntimeProvider
}

func NewHandler(service *Service, runtime *RuntimeProvider) *Handler {
	return &Handler{service: service, runtime: runtime}
}

func (h *Handler) GetPublicAppConfig(w http.ResponseWriter, r *http.Request) {
	features := MiniProgramFeatureConfig{}
	if h.runtime != nil {
		features = h.runtime.MiniProgramFeatures(r.Context())
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"features": features,
	})
}

func (h *Handler) GetBilibiliSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	setting, err := h.service.GetBilibiliSession(r.Context(), userID)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"setting": setting,
	})
}

func (h *Handler) UpdateBilibiliSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	var req updateBilibiliSessionRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	setting, err := h.service.UpdateBilibiliSession(r.Context(), userID, req.Sessdata)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"setting": setting,
	})
}

func (h *Handler) ClearBilibiliSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	setting, err := h.service.ClearBilibiliSession(r.Context(), userID)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"setting": setting,
	})
}
