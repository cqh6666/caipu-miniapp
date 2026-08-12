package addpreview

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

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	kitchenID, err := common.PositiveInt64URLParam(r, "kitchenID")
	if err != nil {
		common.WriteError(w, err)
		return
	}

	var req PreviewRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	result, err := h.service.Preview(r.Context(), userID, kitchenID, req)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}
