package admin

import (
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) ListAIRoutingScenes(w http.ResponseWriter, r *http.Request) {
	if h.aiRouting == nil {
		common.WriteError(w, common.ErrInternal)
		return
	}
	items, err := h.aiRouting.ListScenes(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetAIRoutingScene(w http.ResponseWriter, r *http.Request) {
	if h.aiRouting == nil {
		common.WriteError(w, common.ErrInternal)
		return
	}
	scene := airouter.Scene(strings.TrimSpace(chi.URLParam(r, "scene")))
	config, err := h.aiRouting.GetScene(r.Context(), scene)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"scene": config})
}

func (h *Handler) UpdateAIRoutingScene(w http.ResponseWriter, r *http.Request) {
	if h.aiRouting == nil {
		common.WriteError(w, common.ErrInternal)
		return
	}
	subject, err := h.auth.CurrentSubject(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	scene := airouter.Scene(strings.TrimSpace(chi.URLParam(r, "scene")))
	var req airouter.SceneConfig
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}
	config, err := h.aiRouting.SaveScene(r.Context(), subject, common.RequestID(r.Context()), scene, req)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	if h.aiAlert != nil {
		_ = h.aiAlert.NoteSceneConfigChanged(r.Context(), subject, string(scene))
	}
	common.WriteData(w, http.StatusOK, map[string]any{"scene": config})
}

func (h *Handler) TestAIRoutingScene(w http.ResponseWriter, r *http.Request) {
	if h.aiRouting == nil {
		common.WriteError(w, common.ErrInternal)
		return
	}
	subject, err := h.auth.CurrentSubject(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	scene := airouter.Scene(strings.TrimSpace(chi.URLParam(r, "scene")))
	var req airouter.SceneConfig
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}
	result, err := h.aiRouting.TestScene(r.Context(), subject, common.RequestID(r.Context()), scene, req)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": result})
}
