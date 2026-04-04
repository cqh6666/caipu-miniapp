package mealplan

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

	store, err := h.service.ListStoreByKitchenID(r.Context(), userID, kitchenID)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"store": store,
	})
}

func (h *Handler) SaveDraft(w http.ResponseWriter, r *http.Request) {
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

	planDate := strings.TrimSpace(chi.URLParam(r, "planDate"))
	if planDate == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "planDate is required", http.StatusBadRequest))
		return
	}

	var req savePlanRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	store, err := h.service.SaveDraft(r.Context(), userID, kitchenID, planDate, req)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"store": store,
	})
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
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

	planDate := strings.TrimSpace(chi.URLParam(r, "planDate"))
	if planDate == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "planDate is required", http.StatusBadRequest))
		return
	}

	var req savePlanRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	store, err := h.service.Submit(r.Context(), userID, kitchenID, planDate, req)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"store": store,
	})
}

func (h *Handler) CreateDraftFromSubmitted(w http.ResponseWriter, r *http.Request) {
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

	planDate := strings.TrimSpace(chi.URLParam(r, "planDate"))
	if planDate == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "planDate is required", http.StatusBadRequest))
		return
	}

	store, err := h.service.CreateDraftFromSubmitted(r.Context(), userID, kitchenID, planDate)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"store": store,
	})
}

func (h *Handler) DeleteDraft(w http.ResponseWriter, r *http.Request) {
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

	planDate := strings.TrimSpace(chi.URLParam(r, "planDate"))
	if planDate == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "planDate is required", http.StatusBadRequest))
		return
	}

	store, err := h.service.DeleteDraft(r.Context(), userID, kitchenID, planDate)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"store": store,
	})
}

func (h *Handler) DeleteSubmitted(w http.ResponseWriter, r *http.Request) {
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

	planDate := strings.TrimSpace(chi.URLParam(r, "planDate"))
	if planDate == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "planDate is required", http.StatusBadRequest))
		return
	}

	store, err := h.service.DeleteSubmitted(r.Context(), userID, kitchenID, planDate)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"store": store,
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
