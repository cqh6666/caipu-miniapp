package recipe

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
		MealType: r.URL.Query().Get("mealType"),
		Status:   r.URL.Query().Get("status"),
		Keyword:  r.URL.Query().Get("keyword"),
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

	var req createRecipeRequest
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
		"recipe": item,
	})
}

func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	recipeID := strings.TrimSpace(chi.URLParam(r, "recipeID"))
	if recipeID == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "recipeID is required", http.StatusBadRequest))
		return
	}

	item, err := h.service.GetByID(r.Context(), userID, recipeID)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"recipe": item,
	})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	recipeID := strings.TrimSpace(chi.URLParam(r, "recipeID"))
	if recipeID == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "recipeID is required", http.StatusBadRequest))
		return
	}

	var req updateRecipeRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	item, err := h.service.Update(r.Context(), userID, recipeID, req)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"recipe": item,
	})
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	recipeID := strings.TrimSpace(chi.URLParam(r, "recipeID"))
	if recipeID == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "recipeID is required", http.StatusBadRequest))
		return
	}

	var req updateStatusRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	item, err := h.service.UpdateStatus(r.Context(), userID, recipeID, req.Status)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"recipe": item,
	})
}

func (h *Handler) UpdatePinned(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	recipeID := strings.TrimSpace(chi.URLParam(r, "recipeID"))
	if recipeID == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "recipeID is required", http.StatusBadRequest))
		return
	}

	var req updatePinnedRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	item, err := h.service.UpdatePinned(r.Context(), userID, recipeID, req.Pinned)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"recipe": item,
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	recipeID := strings.TrimSpace(chi.URLParam(r, "recipeID"))
	if recipeID == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "recipeID is required", http.StatusBadRequest))
		return
	}

	if err := h.service.Delete(r.Context(), userID, recipeID); err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"deleted": true,
	})
}

func (h *Handler) RequeueAutoParse(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	recipeID := strings.TrimSpace(chi.URLParam(r, "recipeID"))
	if recipeID == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "recipeID is required", http.StatusBadRequest))
		return
	}

	item, err := h.service.RequeueAutoParse(r.Context(), userID, recipeID)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"recipe": item,
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
