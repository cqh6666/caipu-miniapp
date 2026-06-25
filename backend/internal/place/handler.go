package place

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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
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

	items, err := h.service.ListByKitchenID(r.Context(), userID, kitchenID, ListFilter{
		Status:  r.URL.Query().Get("status"),
		Keyword: r.URL.Query().Get("keyword"),
	})
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

	kitchenID, err := parseKitchenID(r)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	var req placeRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	item, err := h.service.Create(r.Context(), userID, kitchenID, req)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusCreated, map[string]any{
		"place": item,
	})
}

func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	placeID, err := parsePlaceID(r)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	item, err := h.service.GetByID(r.Context(), userID, placeID)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"place": item,
	})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	placeID, err := parsePlaceID(r)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	var req placeRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	item, err := h.service.Update(r.Context(), userID, placeID, req)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"place": item,
	})
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	placeID, err := parsePlaceID(r)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	var req updateStatusRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	item, err := h.service.UpdateStatus(r.Context(), userID, placeID, statusUpdateInput{
		Status:           req.Status,
		VisitedAt:        req.VisitedAt,
		RevisitRating:    req.RevisitRating,
		RecommendedItems: req.RecommendedItems,
	})
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"place": item,
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	placeID, err := parsePlaceID(r)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), userID, placeID); err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"deleted": true,
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

func parsePlaceID(r *http.Request) (string, error) {
	placeID := strings.TrimSpace(chi.URLParam(r, "placeID"))
	if placeID == "" {
		return "", common.NewAppError(common.CodeBadRequest, "placeID is required", http.StatusBadRequest)
	}
	return placeID, nil
}
