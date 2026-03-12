package auth

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

func (h *Handler) WechatLogin(w http.ResponseWriter, r *http.Request) {
	var req wechatLoginRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	session, err := h.service.LoginWithWechatCode(r.Context(), req.Code)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteData(w, http.StatusOK, session)
}

func (h *Handler) DevLogin(w http.ResponseWriter, r *http.Request) {
	var req devLoginRequest
	if err := common.DecodeJSON(r, &req); err != nil {
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
