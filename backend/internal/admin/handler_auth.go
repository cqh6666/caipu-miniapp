package admin

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := h.loginGuard.CheckIP(r); err != nil {
		common.WriteError(w, err)
		return
	}
	var req loginRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}
	if err := h.loginGuard.CheckSubject(req.Username); err != nil {
		common.WriteError(w, err)
		return
	}

	token, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	http.SetCookie(w, h.auth.BuildSessionCookie(token))
	common.WriteData(w, http.StatusOK, map[string]any{
		"username":  strings.TrimSpace(req.Username),
		"csrfToken": h.auth.CSRFToken(token),
	})
}

func (h *Handler) Logout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, h.auth.BuildLogoutCookie())
	common.WriteData(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	subject, err := h.auth.CurrentSubject(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"username":  subject,
		"csrfToken": h.auth.CSRFToken(readAdminToken(r)),
	})
}
