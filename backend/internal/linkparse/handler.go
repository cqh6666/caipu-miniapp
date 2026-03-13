package linkparse

import (
	"net/http"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ParseBilibili(w http.ResponseWriter, r *http.Request) {
	if _, ok := common.CurrentUserID(r.Context()); !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	var req parseBilibiliRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	result, err := h.service.ParseBilibili(r.Context(), req.URL)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}
