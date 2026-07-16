package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/go-chi/chi/v5"
)

func TestHandlerAuthDomainContract(t *testing.T) {
	t.Parallel()

	auth := &handlerAuthStub{token: "session-token", subject: "root-admin"}
	handler := NewHandler(auth, nil, nil, nil, nil, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/api/admin/auth/login", bytes.NewBufferString(`{"username":" root-admin ","password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.Login(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != AdminCookieName || cookies[0].Value != "session-token" {
		t.Fatalf("cookies=%#v", cookies)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"username":"root-admin"`)) ||
		!bytes.Contains(response.Body.Bytes(), []byte(`"csrfToken":"csrf-session-token"`)) {
		t.Fatalf("body=%s", response.Body.String())
	}
}

func TestHandlerDashboardDomainUsesNarrowAuditContract(t *testing.T) {
	t.Parallel()

	auditStub := &handlerAuditStub{overview: audit.DashboardOverview{WindowHours: 12, TaskTotal: 7}}
	handler := NewHandler(&handlerAuthStub{}, auditStub, nil, nil, nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/overview?windowHours=12", nil)
	response := httptest.NewRecorder()

	handler.DashboardOverview(response, request)

	if response.Code != http.StatusOK || auditStub.windowHours != 12 {
		t.Fatalf("status=%d windowHours=%d body=%s", response.Code, auditStub.windowHours, response.Body.String())
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"taskTotal":7`)) {
		t.Fatalf("body=%s", response.Body.String())
	}
}

func TestHandlerRuntimeDomainUsesNarrowRuntimeContract(t *testing.T) {
	t.Parallel()

	runtimeStub := &handlerRuntimeStub{}
	handler := NewHandler(&handlerAuthStub{subject: "root-admin"}, nil, runtimeStub, nil, nil, nil, nil)
	router := chi.NewRouter()
	router.Put("/{group}", handler.UpdateRuntimeGroup)
	request := httptest.NewRequest(http.MethodPut, "/miniapp.features", bytes.NewBufferString(`{"expectedVersion":0,"values":{"diet_assistant_enabled":true},"clearKeys":[]}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || runtimeStub.groupName != "miniapp.features" || runtimeStub.subject != "root-admin" || runtimeStub.expectedVersion != 0 {
		t.Fatalf("status=%d subject=%q group=%q expectedVersion=%d body=%s", response.Code, runtimeStub.subject, runtimeStub.groupName, runtimeStub.expectedVersion, response.Body.String())
	}
	if value, ok := runtimeStub.values["diet_assistant_enabled"].(bool); !ok || !value {
		t.Fatalf("values=%#v", runtimeStub.values)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte(`"name":"miniapp.features"`)) {
		t.Fatalf("body=%s", response.Body.String())
	}
}

func TestHandlerRuntimeDomainRequiresExpectedVersion(t *testing.T) {
	t.Parallel()

	runtimeStub := &handlerRuntimeStub{}
	handler := NewHandler(&handlerAuthStub{subject: "root-admin"}, nil, runtimeStub, nil, nil, nil, nil)
	router := chi.NewRouter()
	router.Put("/{group}", handler.UpdateRuntimeGroup)
	request := httptest.NewRequest(http.MethodPut, "/miniapp.features", bytes.NewBufferString(`{"values":{"diet_assistant_enabled":true}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || runtimeStub.groupName != "" {
		t.Fatalf("status=%d group=%q body=%s", response.Code, runtimeStub.groupName, response.Body.String())
	}
}

func TestHandlerAIRoutingDomainUsesNarrowRoutingContract(t *testing.T) {
	t.Parallel()

	routingStub := &handlerAIRoutingStub{}
	alertStub := &handlerAlertNoteStub{}
	handler := NewHandler(&handlerAuthStub{subject: "root-admin"}, nil, nil, nil, nil, routingStub, alertStub)
	router := chi.NewRouter()
	router.Put("/{scene}", handler.UpdateAIRoutingScene)
	request := httptest.NewRequest(http.MethodPut, "/summary", bytes.NewBufferString(`{"version":7,"enabled":true,"strategy":"priority_failover","providers":[]}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || routingStub.scene != airouter.SceneSummary || routingStub.subject != "root-admin" {
		t.Fatalf("status=%d subject=%q scene=%q body=%s", response.Code, routingStub.subject, routingStub.scene, response.Body.String())
	}
	if !routingStub.input.Enabled || routingStub.input.Version != 7 || alertStub.scene != "summary" || alertStub.subject != "root-admin" {
		t.Fatalf("routing=%#v alert=%#v", routingStub, alertStub)
	}
	var payload struct {
		Data struct {
			Scene airouter.SceneConfig `json:"scene"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Scene.Scene != airouter.SceneSummary {
		t.Fatalf("response scene=%q", payload.Data.Scene.Scene)
	}
}

type handlerAuthStub struct {
	token   string
	subject string
}

func (s *handlerAuthStub) Login(context.Context, string, string) (string, error) { return s.token, nil }
func (s *handlerAuthStub) CurrentSubject(context.Context) (string, error)        { return s.subject, nil }
func (s *handlerAuthStub) BuildSessionCookie(token string) *http.Cookie {
	return &http.Cookie{Name: AdminCookieName, Value: token, Path: "/api/admin", HttpOnly: true}
}
func (s *handlerAuthStub) BuildLogoutCookie() *http.Cookie {
	return &http.Cookie{Name: AdminCookieName, Path: "/api/admin", MaxAge: -1}
}
func (s *handlerAuthStub) CSRFToken(token string) string { return "csrf-" + token }

type handlerAuditStub struct {
	overview    audit.DashboardOverview
	windowHours int
}

func (s *handlerAuditStub) Overview(_ context.Context, windowHours int) (audit.DashboardOverview, error) {
	s.windowHours = windowHours
	return s.overview, nil
}
func (s *handlerAuditStub) ListJobs(context.Context, audit.JobListFilter) (audit.PaginationResult[audit.JobRunRecord], error) {
	return audit.PaginationResult[audit.JobRunRecord]{}, nil
}
func (s *handlerAuditStub) Trends(context.Context, string) ([]audit.TrendBucket, error) {
	return nil, nil
}
func (s *handlerAuditStub) GetJobDetail(context.Context, int64) (audit.JobRunRecord, []audit.CallLogRecord, error) {
	return audit.JobRunRecord{}, nil, nil
}
func (s *handlerAuditStub) ListCalls(context.Context, audit.CallListFilter) (audit.PaginationResult[audit.CallLogRecord], error) {
	return audit.PaginationResult[audit.CallLogRecord]{}, nil
}

type handlerRuntimeStub struct {
	subject         string
	groupName       string
	expectedVersion int
	values          map[string]any
}

func (s *handlerRuntimeStub) ListRuntimeGroups(context.Context) ([]appsettings.RuntimeSettingGroupView, error) {
	return nil, nil
}
func (s *handlerRuntimeStub) UpdateRuntimeGroup(_ context.Context, subject, _ string, groupName string, expectedVersion int, values map[string]any, _ []string) (appsettings.RuntimeSettingGroupView, error) {
	s.subject, s.groupName, s.expectedVersion, s.values = subject, groupName, expectedVersion, values
	return appsettings.RuntimeSettingGroupView{Name: groupName}, nil
}
func (s *handlerRuntimeStub) TestRuntimeGroup(context.Context, string, string, string, map[string]any, []string) (appsettings.GroupTestResult, error) {
	return appsettings.GroupTestResult{}, nil
}
func (s *handlerRuntimeStub) ListSettingAudits(context.Context, appsettings.SettingAuditFilter) (appsettings.SettingAuditList, error) {
	return appsettings.SettingAuditList{}, nil
}

type handlerAIRoutingStub struct {
	subject string
	scene   airouter.Scene
	input   airouter.SceneConfig
}

func (s *handlerAIRoutingStub) ListScenes(context.Context) ([]airouter.SceneSummaryView, error) {
	return nil, nil
}
func (s *handlerAIRoutingStub) GetScene(context.Context, airouter.Scene) (airouter.SceneConfig, error) {
	return airouter.SceneConfig{}, nil
}
func (s *handlerAIRoutingStub) SaveScene(_ context.Context, subject, _ string, scene airouter.Scene, input airouter.SceneConfig) (airouter.SceneConfig, error) {
	s.subject, s.scene, s.input = subject, scene, input
	input.Scene = scene
	return input, nil
}
func (s *handlerAIRoutingStub) TestScene(context.Context, string, string, airouter.Scene, airouter.SceneConfig) (airouter.TestResult, error) {
	return airouter.TestResult{}, nil
}

type handlerAlertNoteStub struct {
	subject string
	scene   string
}

func (s *handlerAlertNoteStub) Overview(context.Context) (aialert.Overview, error) {
	return aialert.Overview{}, nil
}
func (s *handlerAlertNoteStub) Retest(context.Context, string, string) (aialert.MutationResult, error) {
	return aialert.MutationResult{}, nil
}
func (s *handlerAlertNoteStub) Archive(context.Context, string, string, string) (aialert.MutationResult, error) {
	return aialert.MutationResult{}, nil
}
func (s *handlerAlertNoteStub) Mute(context.Context, string, string, int, string) (aialert.MutationResult, error) {
	return aialert.MutationResult{}, nil
}
func (s *handlerAlertNoteStub) Unmute(context.Context, string, string) (aialert.MutationResult, error) {
	return aialert.MutationResult{}, nil
}
func (s *handlerAlertNoteStub) NoteSceneConfigChanged(_ context.Context, subject, scene string) error {
	s.subject, s.scene = subject, scene
	return nil
}
