package linkparse

import (
	"net/http"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) PreviewLink(w http.ResponseWriter, r *http.Request) {
	if _, ok := common.CurrentUserID(r.Context()); !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	var req parseLinkRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	ctx := audit.WithRequestMeta(r.Context(), audit.RequestMeta{
		TriggerSource: "preview",
		TargetType:    "preview_link",
		TargetID:      audit.HashTargetID(req.URL),
	})
	result, err := h.service.PreviewLink(ctx, req.URL)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

func (h *Handler) ParseBilibili(w http.ResponseWriter, r *http.Request) {
	if _, ok := common.CurrentUserID(r.Context()); !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	var req parseLinkRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	ctx := audit.WithRequestMeta(r.Context(), audit.RequestMeta{
		TriggerSource: "manual",
		TargetType:    "manual_link",
		TargetID:      audit.HashTargetID(req.URL),
	})
	result, err := h.service.ParseBilibili(ctx, req.URL)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

func (h *Handler) ParseXiaohongshu(w http.ResponseWriter, r *http.Request) {
	if _, ok := common.CurrentUserID(r.Context()); !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	var req parseLinkRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	ctx := audit.WithRequestMeta(r.Context(), audit.RequestMeta{
		TriggerSource: "manual",
		TargetType:    "manual_link",
		TargetID:      audit.HashTargetID(req.URL),
	})
	result, err := h.service.ParseXiaohongshu(ctx, req.URL)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}
