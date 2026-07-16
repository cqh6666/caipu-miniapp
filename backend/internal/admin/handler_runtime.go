package admin

import (
	"context"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/go-chi/chi/v5"
)

type runtimeGroupMutationRequest struct {
	ExpectedVersion *int           `json:"expectedVersion"`
	Values          map[string]any `json:"values"`
	ClearKeys       []string       `json:"clearKeys"`
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
	common.WriteData(w, http.StatusOK, map[string]any{"groups": append(groups, bilibiliGroup)})
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
	if req.ExpectedVersion == nil || *req.ExpectedVersion < 0 {
		common.WriteError(w, common.NewAppError(common.CodeBadRequest, "expectedVersion is required", http.StatusBadRequest))
		return
	}
	if groupName == "bilibili.session" {
		group, err := h.updateBilibiliGroup(r.Context(), subject, req)
		if err != nil {
			common.WriteError(w, err)
			return
		}
		common.WriteData(w, http.StatusOK, map[string]any{"group": group})
		return
	}
	group, err := h.runtime.UpdateRuntimeGroup(r.Context(), subject, common.RequestID(r.Context()), groupName, *req.ExpectedVersion, req.Values, req.ClearKeys)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"group": group})
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
		common.WriteData(w, http.StatusOK, map[string]any{"result": result})
		return
	}
	result, err := h.runtime.TestRuntimeGroup(r.Context(), subject, common.RequestID(r.Context()), groupName, req.Values, req.ClearKeys)
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": result})
}

func (h *Handler) ListRuntimeAudits(w http.ResponseWriter, r *http.Request) {
	result, err := h.runtime.ListSettingAudits(r.Context(), appsettings.SettingAuditFilter{
		GroupName:       r.URL.Query().Get("group"),
		Action:          r.URL.Query().Get("action"),
		OperatorSubject: r.URL.Query().Get("operator"),
		SettingKey:      r.URL.Query().Get("settingKey"),
		TimeFrom:        r.URL.Query().Get("timeFrom"),
		TimeTo:          r.URL.Query().Get("timeTo"),
		Page:            parseIntQuery(r, "page", 1),
		PageSize:        parseIntQuery(r, "pageSize", 20),
	})
	if err != nil {
		common.WriteError(w, err)
		return
	}
	common.WriteData(w, http.StatusOK, map[string]any{"result": result})
}

func (h *Handler) buildBilibiliGroup(ctx context.Context) (appsettings.RuntimeSettingGroupView, error) {
	setting, err := h.appSettingsSvc.CurrentBilibiliSessionSetting(ctx)
	if err != nil {
		return appsettings.RuntimeSettingGroupView{}, err
	}
	return appsettings.RuntimeSettingGroupView{
		Name: "bilibili.session", Version: setting.Version, Title: "B 站字幕配置", Description: "复用现有的全局 B 站 SESSDATA 配置。",
		Fields: []appsettings.RuntimeSettingFieldView{{
			Key: "sessdata", Label: "SESSDATA", Description: "用于 B 站字幕命中的登录态 Cookie。",
			ValueType: "string", IsSecret: true, HasValue: setting.Configured,
			MaskedValue: setting.MaskedSessdata, Source: sourceForBilibili(setting.Configured), UpdatedAt: setting.UpdatedAt,
		}},
	}, nil
}

func (h *Handler) updateBilibiliGroup(ctx context.Context, subject string, req runtimeGroupMutationRequest) (appsettings.RuntimeSettingGroupView, error) {
	if req.ExpectedVersion == nil || *req.ExpectedVersion < 0 {
		return appsettings.RuntimeSettingGroupView{}, common.NewAppError(common.CodeBadRequest, "expectedVersion is required", http.StatusBadRequest)
	}
	if containsString(req.ClearKeys, "sessdata") {
		if _, err := h.appSettingsSvc.ClearBilibiliSessionBySubjectIfVersion(ctx, "admin:"+subject, nil, *req.ExpectedVersion); err != nil {
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
	if _, err := h.appSettingsSvc.UpdateBilibiliSessionBySubjectIfVersion(ctx, "admin:"+subject, nil, *req.ExpectedVersion, value); err != nil {
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
