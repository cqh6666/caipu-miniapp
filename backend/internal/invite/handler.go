package invite

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

	var req createInviteRequest
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
		"invite": item,
	})
}

func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if token == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "invite token is required", http.StatusBadRequest))
		return
	}

	item, err := h.service.Preview(r.Context(), token)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"invite": item,
	})
}

func (h *Handler) PreviewByCode(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(chi.URLParam(r, "code"))
	if code == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "invite code is required", http.StatusBadRequest))
		return
	}

	item, err := h.service.PreviewByCode(r.Context(), code)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"invite": item,
	})
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	token := strings.TrimSpace(chi.URLParam(r, "token"))
	if token == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "invite token is required", http.StatusBadRequest))
		return
	}

	result, err := h.service.Accept(r.Context(), userID, token)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, result)
}

func (h *Handler) AcceptByCode(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	code := strings.TrimSpace(chi.URLParam(r, "code"))
	if code == "" {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "invite code is required", http.StatusBadRequest))
		return
	}

	result, err := h.service.AcceptByCode(r.Context(), userID, code)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, result)
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
