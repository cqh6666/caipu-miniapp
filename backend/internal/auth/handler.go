package auth

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/ratelimit"
)

type Handler struct {
	service    *Service
	loginGuard *ratelimit.Guard
}

func (h *Handler) SetLoginGuard(guard *ratelimit.Guard) {
	if h != nil {
		h.loginGuard = guard
	}
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) WechatLogin(w http.ResponseWriter, r *http.Request) {
	if err := h.loginGuard.CheckIP(r); err != nil {
		common.WriteError(w, err)
		return
	}
	var req wechatLoginRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}
	if err := h.loginGuard.CheckSubject(strings.TrimSpace(req.AppID) + ":" + strings.TrimSpace(req.Code)); err != nil {
		common.WriteError(w, err)
		return
	}

	session, err := h.service.LoginWithWechatCode(r.Context(), req.Code, req.AppID, req.Nickname, req.AvatarURL)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, session)
}

func (h *Handler) DevLogin(w http.ResponseWriter, r *http.Request) {
	if err := h.loginGuard.CheckIP(r); err != nil {
		common.WriteError(w, err)
		return
	}
	var req devLoginRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}
	if err := h.loginGuard.CheckSubject(req.Identity); err != nil {
		common.WriteError(w, err)
		return
	}

	session, err := h.service.LoginForDev(r.Context(), req.Identity)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, session)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	session, err := h.service.CurrentSession(r.Context(), userID)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, session)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}
	if err := h.service.Logout(r.Context(), userID); err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := common.CurrentUserID(r.Context())
	if !ok {
		common.WriteError(w, common.ErrUnauthorized)
		return
	}

	var req updateProfileRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, req.Nickname, req.AvatarURL)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, map[string]any{
		"user": user,
	})
}
