package kitchen

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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	items, err := h.service.ListByUserID(r.Context(), userID)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	var req createKitchenRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	item, err := h.service.CreateKitchen(r.Context(), userID, req.Name)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusCreated, map[string]any{
		"kitchen": item,
	})
}
