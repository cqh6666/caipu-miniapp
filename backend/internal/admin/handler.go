package admin

import (
	"context"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/ratelimit"
)

type AIAlertService interface {
	Overview(ctx context.Context) (aialert.Overview, error)
	Retest(ctx context.Context, providerID, subject string) (aialert.MutationResult, error)
	Archive(ctx context.Context, providerID, subject, reason string) (aialert.MutationResult, error)
	Mute(ctx context.Context, providerID, subject string, durationHours int, reason string) (aialert.MutationResult, error)
	Unmute(ctx context.Context, providerID, subject string) (aialert.MutationResult, error)
	NoteSceneConfigChanged(ctx context.Context, subject, scene string) error
}

type AuditService interface {
	Overview(context.Context, int) (audit.DashboardOverview, error)
	ListJobs(context.Context, audit.JobListFilter) (audit.PaginationResult[audit.JobRunRecord], error)
	Trends(context.Context, string) ([]audit.TrendBucket, error)
	GetJobDetail(context.Context, int64) (audit.JobRunRecord, []audit.CallLogRecord, error)
	ListCalls(context.Context, audit.CallListFilter) (audit.PaginationResult[audit.CallLogRecord], error)
}

type RuntimeSettingsService interface {
	ListRuntimeGroups(context.Context) ([]appsettings.RuntimeSettingGroupView, error)
	UpdateRuntimeGroup(context.Context, string, string, string, int, map[string]any, []string) (appsettings.RuntimeSettingGroupView, error)
	TestRuntimeGroup(context.Context, string, string, string, map[string]any, []string) (appsettings.GroupTestResult, error)
	ListSettingAudits(context.Context, appsettings.SettingAuditFilter) (appsettings.SettingAuditList, error)
}

type BilibiliSettingsService interface {
	CurrentBilibiliSessionSetting(context.Context) (appsettings.BilibiliSessionSetting, error)
	UpdateBilibiliSessionBySubject(context.Context, string, *int64, string) (appsettings.BilibiliSessionSetting, error)
	UpdateBilibiliSessionBySubjectIfVersion(context.Context, string, *int64, int, string) (appsettings.BilibiliSessionSetting, error)
	ClearBilibiliSessionBySubject(context.Context, string, *int64) (appsettings.BilibiliSessionSetting, error)
	ClearBilibiliSessionBySubjectIfVersion(context.Context, string, *int64, int) (appsettings.BilibiliSessionSetting, error)
	TestBilibiliSession(context.Context, string, string) (appsettings.GroupTestResult, error)
	CurrentBilibiliSessdata(context.Context) (string, error)
}

type ServerHealthReader interface {
	Overview(context.Context) (ServerHealthOverview, error)
}

type AIRoutingService interface {
	ListScenes(context.Context) ([]airouter.SceneSummaryView, error)
	GetScene(context.Context, airouter.Scene) (airouter.SceneConfig, error)
	SaveScene(context.Context, string, string, airouter.Scene, airouter.SceneConfig) (airouter.SceneConfig, error)
	TestScene(context.Context, string, string, airouter.Scene, airouter.SceneConfig) (airouter.TestResult, error)
}

type Handler struct {
	auth           AuthService
	audit          AuditService
	runtime        RuntimeSettingsService
	appSettingsSvc BilibiliSettingsService
	serverHealth   ServerHealthReader
	aiRouting      AIRoutingService
	aiAlert        AIAlertService
	loginGuard     *ratelimit.Guard
}

func NewHandler(
	auth AuthService,
	auditService AuditService,
	runtimeProvider RuntimeSettingsService,
	appSettingsService BilibiliSettingsService,
	serverHealthService ServerHealthReader,
	aiRoutingService AIRoutingService,
	aiAlertService AIAlertService,
) *Handler {
	return &Handler{
		auth:           auth,
		audit:          auditService,
		runtime:        runtimeProvider,
		appSettingsSvc: appSettingsService,
		serverHealth:   serverHealthService,
		aiRouting:      aiRoutingService,
		aiAlert:        aiAlertService,
	}
}

func (h *Handler) SetLoginGuard(guard *ratelimit.Guard) {
	if h != nil {
		h.loginGuard = guard
	}
}
