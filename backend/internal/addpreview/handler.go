package addpreview

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

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

	kitchenID, err := parseKitchenID(r)
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

func parseKitchenID(r *http.Request) (int64, error) {
	kitchenID := strings.TrimSpace(chi.URLParam(r, "kitchenID"))
	if kitchenID == "" {
		return 0, common.NewAppError(common.CodeBadRequest, "kitchenID is required", http.StatusBadRequest)
	}

	value, err := strconv.ParseInt(kitchenID, 10, 64)
	if err != nil || value <= 0 {
		return 0, common.NewAppError(common.CodeBadRequest, "invalid kitchenID", http.StatusBadRequest)
	}

	return value, nil
}
