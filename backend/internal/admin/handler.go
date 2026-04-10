package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	auth           AuthService
	audit          *audit.Service
	runtime        *appsettings.RuntimeProvider
	appSettingsSvc *appsettings.Service
	serverHealth   *ServerHealthService
	aiRouting      *airouter.Service
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type runtimeGroupMutationRequest struct {
	Values    map[string]any `json:"values"`
	ClearKeys []string       `json:"clearKeys"`
}

func NewHandler(
	auth AuthService,
	auditService *audit.Service,
	runtimeProvider *appsettings.RuntimeProvider,
	appSettingsService *appsettings.Service,
	serverHealthService *ServerHealthService,
	aiRoutingService *airouter.Service,
) *Handler {
	return &Handler{
		auth:           auth,
		audit:          auditService,
		runtime:        runtimeProvider,
		appSettingsSvc: appSettingsService,
		serverHealth:   serverHealthService,
		aiRouting:      aiRoutingService,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := common.DecodeJSON(r, &req); err != nil {
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
		"username": strings.TrimSpace(req.Username),
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, h.auth.BuildLogoutCookie())
	common.WriteData(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	subject, err := h.auth.CurrentSubject(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"username": subject,
	})
}

func (h *Handler) DashboardOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.audit.Overview(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"overview": overview,
	})
}

func (h *Handler) DashboardFailures(w http.ResponseWriter, r *http.Request) {
	failed, err := h.audit.ListJobs(r.Context(), audit.JobListFilter{
		Status:   audit.JobStatusFailed,
		Page:     parseIntQuery(r, "page", 1),
		PageSize: parseIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"result": failed,
	})
}

func (h *Handler) DashboardTrends(w http.ResponseWriter, r *http.Request) {
	items, err := h.audit.Trends(r.Context(), r.URL.Query().Get("range"))
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

func (h *Handler) ServerHealthOverview(w http.ResponseWriter, r *http.Request) {
	if h.serverHealth == nil {
		common.WriteError(w, common.ErrInternal)
		return
	}

	overview, err := h.serverHealth.Overview(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"overview": overview,
	})
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	result, err := h.audit.ListJobs(r.Context(), audit.JobListFilter{
		Scene:         r.URL.Query().Get("scene"),
		Status:        r.URL.Query().Get("status"),
		TriggerSource: r.URL.Query().Get("triggerSource"),
		TargetID:      r.URL.Query().Get("targetId"),
		TimeFrom:      r.URL.Query().Get("timeFrom"),
		TimeTo:        r.URL.Query().Get("timeTo"),
		Page:          parseIntQuery(r, "page", 1),
		PageSize:      parseIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

func (h *Handler) GetJobDetail(w http.ResponseWriter, r *http.Request) {
	jobID, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, "id")), 10, 64)
	if err != nil || jobID <= 0 {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "invalid job id", http.StatusBadRequest))
		return
	}

	job, calls, err := h.audit.GetJobDetail(r.Context(), jobID)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"job":   job,
		"calls": calls,
	})
}

func (h *Handler) ListCalls(w http.ResponseWriter, r *http.Request) {
	result, err := h.audit.ListCalls(r.Context(), audit.CallListFilter{
		Scene:     r.URL.Query().Get("scene"),
		Status:    r.URL.Query().Get("status"),
		Provider:  r.URL.Query().Get("provider"),
		Model:     r.URL.Query().Get("model"),
		RequestID: r.URL.Query().Get("requestId"),
		TimeFrom:  r.URL.Query().Get("timeFrom"),
		TimeTo:    r.URL.Query().Get("timeTo"),
		Page:      parseIntQuery(r, "page", 1),
		PageSize:  parseIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

func (h *Handler) ListRuntimeSettings(w http.ResponseWriter, r *http.Request) {
	groups, err := h.runtime.ListRuntimeGroups(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	bilibiliGroup, err := h.buildBilibiliGroup(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}
	groups = append(groups, bilibiliGroup)
	common.WriteData(w, http.StatusOK, map[string]any{
		"groups": groups,
	})
}

func (h *Handler) UpdateRuntimeGroup(w http.ResponseWriter, r *http.Request) {
	subject, err := h.auth.CurrentSubject(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}

	groupName := strings.TrimSpace(chi.URLParam(r, "group"))
	var req runtimeGroupMutationRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	if groupName == "bilibili.session" {
		group, err := h.updateBilibiliGroup(r.Context(), subject, req)
		if err != nil {
			common.WriteError(w, err)
			return
		}
		common.WriteData(w, http.StatusOK, map[string]any{
			"group": group,
		})
		return
	}

	group, err := h.runtime.UpdateRuntimeGroup(r.Context(), subject, common.RequestID(r.Context()), groupName, req.Values, req.ClearKeys)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"group": group,
	})
}

func (h *Handler) TestRuntimeGroup(w http.ResponseWriter, r *http.Request) {
	subject, err := h.auth.CurrentSubject(r.Context())
	if err != nil {
		common.WriteError(w, err)
		return
	}

	groupName := strings.TrimSpace(chi.URLParam(r, "group"))
	var req runtimeGroupMutationRequest
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	if groupName == "bilibili.session" {
		result, err := h.testBilibiliGroup(r.Context(), subject, req)
		if err != nil {
			common.WriteError(w, err)
			return
		}
		common.WriteData(w, http.StatusOK, map[string]any{
			"result": result,
		})
		return
	}

	result, err := h.runtime.TestRuntimeGroup(r.Context(), subject, common.RequestID(r.Context()), groupName, req.Values, req.ClearKeys)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

func (h *Handler) ListRuntimeAudits(w http.ResponseWriter, r *http.Request) {
	result, err := h.runtime.ListSettingAudits(r.Context(), appsettings.SettingAuditFilter{
		GroupName: r.URL.Query().Get("group"),
		Action:    r.URL.Query().Get("action"),
		Page:      parseIntQuery(r, "page", 1),
		PageSize:  parseIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

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
	common.WriteData(w, http.StatusOK, map[string]any{
		"items": items,
	})
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
	common.WriteData(w, http.StatusOK, map[string]any{
		"scene": config,
	})
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
	common.WriteData(w, http.StatusOK, map[string]any{
		"scene": config,
	})
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
	common.WriteData(w, http.StatusOK, map[string]any{
		"result": result,
	})
}

func (h *Handler) buildBilibiliGroup(ctx context.Context) (appsettings.RuntimeSettingGroupView, error) {
	setting, err := h.appSettingsSvc.CurrentBilibiliSessionSetting(ctx)
	if err != nil {
		return appsettings.RuntimeSettingGroupView{}, err
	}

	return appsettings.RuntimeSettingGroupView{
		Name:        "bilibili.session",
		Title:       "B 站字幕配置",
		Description: "复用现有的全局 B 站 SESSDATA 配置。",
		Fields: []appsettings.RuntimeSettingFieldView{
			{
				Key:         "sessdata",
				Label:       "SESSDATA",
				Description: "用于 B 站字幕命中的登录态 Cookie。",
				ValueType:   "string",
				IsSecret:    true,
				HasValue:    setting.Configured,
				MaskedValue: setting.MaskedSessdata,
				Source:      sourceForBilibili(setting.Configured),
				UpdatedAt:   setting.UpdatedAt,
			},
		},
	}, nil
}

func (h *Handler) updateBilibiliGroup(ctx context.Context, subject string, req runtimeGroupMutationRequest) (appsettings.RuntimeSettingGroupView, error) {
	if containsString(req.ClearKeys, "sessdata") {
		if _, err := h.appSettingsSvc.ClearBilibiliSessionBySubject(ctx, subject, 0); err != nil {
			return appsettings.RuntimeSettingGroupView{}, err
		}
		return h.buildBilibiliGroup(ctx)
	}

	raw, ok := req.Values["sessdata"]
	if !ok {
		return appsettings.RuntimeSettingGroupView{}, common.NewAppError(common.CodeBadRequest, "sessdata is required", http.StatusBadRequest)
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return appsettings.RuntimeSettingGroupView{}, common.NewAppError(common.CodeBadRequest, "sessdata is required", http.StatusBadRequest)
	}
	if _, err := h.appSettingsSvc.UpdateBilibiliSessionBySubject(ctx, subject, 0, value); err != nil {
		return appsettings.RuntimeSettingGroupView{}, err
	}
	return h.buildBilibiliGroup(ctx)
}

func (h *Handler) testBilibiliGroup(ctx context.Context, subject string, req runtimeGroupMutationRequest) (appsettings.GroupTestResult, error) {
	if containsString(req.ClearKeys, "sessdata") {
		return appsettings.GroupTestResult{OK: false, Message: "已清空的 sessdata 无法测试"}, nil
	}

	if raw, ok := req.Values["sessdata"]; ok {
		value, ok := raw.(string)
		if !ok || strings.TrimSpace(value) == "" {
			return appsettings.GroupTestResult{}, common.NewAppError(common.CodeBadRequest, "sessdata is required", http.StatusBadRequest)
		}
		return h.appSettingsSvc.TestBilibiliSession(ctx, subject, value)
	}

	current, err := h.appSettingsSvc.CurrentBilibiliSessdata(ctx)
	if err != nil {
		return appsettings.GroupTestResult{}, err
	}
	if strings.TrimSpace(current) == "" {
		return appsettings.GroupTestResult{OK: false, Message: "当前没有已保存的 sessdata，无法测试"}, nil
	}
	return h.appSettingsSvc.TestBilibiliSession(ctx, subject, current)
}

func parseIntQuery(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func containsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func sourceForBilibili(configured bool) string {
	if configured {
		return "db"
	}
	return "none"
}
