package appsettings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	_ "modernc.org/sqlite"
)

func TestRuntimeProviderConcurrentUpdatesRejectStaleVersion(t *testing.T) {
	provider := newRuntimeProviderForTest(t)
	const groupName = "miniapp.features"

	start := make(chan struct{})
	errorsCh := make(chan error, 2)
	var wg sync.WaitGroup
	for _, enabled := range []bool{true, false} {
		wg.Add(1)
		go func(enabled bool) {
			defer wg.Done()
			<-start
			_, err := provider.UpdateRuntimeGroup(context.Background(), "tester", "req-cas", groupName, 0, map[string]any{
				"diet_assistant_enabled": enabled,
			}, nil)
			errorsCh <- err
		}(enabled)
	}
	close(start)
	wg.Wait()
	close(errorsCh)

	succeeded := 0
	conflicted := 0
	for err := range errorsCh {
		if err == nil {
			succeeded++
			continue
		}
		var appErr *common.AppError
		if errors.As(err, &appErr) && appErr.HTTPStatus == http.StatusConflict {
			conflicted++
			continue
		}
		t.Fatalf("UpdateRuntimeGroup() unexpected error = %v", err)
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("success/conflict = %d/%d, want 1/1", succeeded, conflicted)
	}
	group, err := provider.GetRuntimeGroup(context.Background(), groupName)
	if err != nil {
		t.Fatalf("GetRuntimeGroup() error = %v", err)
	}
	if group.Version != 1 {
		t.Fatalf("group.Version = %d, want 1", group.Version)
	}
}

type appSettingsHTTPDoerFunc func(*http.Request) (*http.Response, error)

func (f appSettingsHTTPDoerFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestRuntimeProviderUpdateRuntimeGroupPrefersExplicitSecretValueOverClear(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	ctx := context.Background()

	first, err := provider.UpdateRuntimeGroup(ctx, "tester", "req-1", "sidecar.linkparse", 0, map[string]any{
		"api_key": "old-secret",
	}, nil)
	if err != nil {
		t.Fatalf("initial UpdateRuntimeGroup() error = %v", err)
	}

	if _, err := provider.UpdateRuntimeGroup(ctx, "tester", "req-2", "sidecar.linkparse", first.Version, map[string]any{
		"api_key": "new-secret",
	}, []string{"api_key"}); err != nil {
		t.Fatalf("conflicting UpdateRuntimeGroup() error = %v", err)
	}

	if got := provider.LinkparseSidecar(ctx).APIKey; got != "new-secret" {
		t.Fatalf("provider.LinkparseSidecar().APIKey = %q, want %q", got, "new-secret")
	}
}

func TestRuntimeProviderTestRuntimeGroupPrefersExplicitValueOverClear(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/health" {
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
		if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer live-secret" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer live-secret")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"test"}`))
	}))
	defer server.Close()

	result, err := provider.TestRuntimeGroup(ctx, "tester", "req-3", "sidecar.linkparse", map[string]any{
		"base_url": server.URL,
		"api_key":  "live-secret",
	}, []string{"base_url", "api_key"})
	if err != nil {
		t.Fatalf("TestRuntimeGroup() error = %v", err)
	}
	if !result.OK {
		t.Fatalf("TestRuntimeGroup().OK = false, message = %q", result.Message)
	}
}

func TestRuntimeProviderProbeUsesInjectedDoerHeadersAndDeadline(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTestWithOptions(t, RuntimeProviderOptions{
		HTTPDoer: appSettingsHTTPDoerFunc(func(request *http.Request) (*http.Response, error) {
			if request.Method != http.MethodGet || request.URL.String() != "https://sidecar.example/v1/health" {
				t.Fatalf("request=%s %s", request.Method, request.URL.String())
			}
			if got := request.Header.Get("Authorization"); got != "Bearer injected-secret" {
				t.Fatalf("Authorization=%q", got)
			}
			deadline, ok := request.Context().Deadline()
			if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 6*time.Second {
				t.Fatalf("request deadline=%v ok=%t", deadline, ok)
			}
			return appSettingsResponse(http.StatusOK, `{"ok":true}`), nil
		}),
	})

	result, err := provider.TestRuntimeGroup(context.Background(), "tester", "req-http-doer", "sidecar.linkparse", map[string]any{
		"base_url":        "https://sidecar.example",
		"api_key":         "injected-secret",
		"timeout_seconds": 5,
	}, nil)
	if err != nil || !result.OK {
		t.Fatalf("result=%#v error=%v", result, err)
	}
}

func TestRuntimeProviderProbeInjectedDoerBoundsResponseAndHonorsCancellation(t *testing.T) {
	t.Parallel()

	t.Run("oversized response", func(t *testing.T) {
		result := testSidecarHealth(
			context.Background(),
			appSettingsHTTPDoerFunc(func(*http.Request) (*http.Response, error) {
				return appSettingsResponse(http.StatusOK, strings.Repeat("x", int(maxRuntimeProbeResponseBytes)+1)), nil
			}),
			"https://sidecar.example",
			"secret",
			time.Second,
		)
		if result.OK || !strings.Contains(result.Message, "响应超过大小限制") {
			t.Fatalf("result=%#v", result)
		}
	})

	t.Run("caller cancellation", func(t *testing.T) {
		observed := make(chan error, 1)
		provider := newRuntimeProviderForTestWithOptions(t, RuntimeProviderOptions{
			HTTPDoer: appSettingsHTTPDoerFunc(func(request *http.Request) (*http.Response, error) {
				<-request.Context().Done()
				observed <- request.Context().Err()
				return nil, request.Context().Err()
			}),
		})
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		result, err := provider.TestRuntimeGroup(ctx, "tester", "req-cancel", "sidecar.linkparse", map[string]any{
			"base_url":        "https://sidecar.example",
			"timeout_seconds": 5,
		}, nil)
		if err != nil || result.OK || !strings.Contains(result.Message, "网络和超时配置") {
			t.Fatalf("result=%#v error=%v", result, err)
		}
		select {
		case observedErr := <-observed:
			if observedErr == nil {
				t.Fatal("injected doer did not observe cancellation")
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for injected doer cancellation")
		}
	})
}

func TestRuntimeProviderTestRuntimeGroupUsesImageGenerationEndpointForFlowchart(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
		if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer flowchart-secret" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer flowchart-secret")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body error = %v", err)
		}
		if _, ok := payload["quality"]; ok {
			t.Fatalf("request body unexpectedly contains quality: %#v", payload["quality"])
		}
		if got := payload["output_format"]; got != "png" {
			t.Fatalf("request output_format = %#v, want %q", got, "png")
		}
		if _, ok := payload["response_format"]; ok {
			t.Fatalf("request body unexpectedly contains response_format for gpt-image model: %#v", payload["response_format"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aF9sAAAAASUVORK5CYII="}]}`))
	}))
	defer server.Close()

	result := testFlowchartCompatible(
		ctx,
		nil,
		server.URL,
		"flowchart-secret",
		"gpt-image-2",
		"images_generations",
		"b64_json",
		0,
	)
	if !result.OK {
		t.Fatalf("testFlowchartCompatible().OK = false, message = %q", result.Message)
	}
}

func TestRuntimeProbesDoNotExposeUpstreamErrorBodies(t *testing.T) {
	t.Parallel()

	const rawBody = `<html>api_key=sk-provider-secret host=internal.provider.example</html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(rawBody))
	}))
	defer server.Close()

	results := []GroupTestResult{
		testOpenAICompatible(context.Background(), nil, server.URL, "secret", "model", time.Second),
		testFlowchartCompatible(context.Background(), nil, server.URL, "secret", "model", "chat_completions", "auto", time.Second),
		testSidecarHealth(context.Background(), nil, server.URL, "secret", time.Second),
	}
	for _, result := range results {
		if result.OK || !strings.Contains(result.Message, "502") {
			t.Fatalf("unexpected probe result: %#v", result)
		}
		for _, sensitive := range []string{"sk-provider-secret", "internal.provider.example", "<html>", "api_key"} {
			if strings.Contains(result.Message, sensitive) {
				t.Fatalf("probe message leaked %q: %q", sensitive, result.Message)
			}
		}
	}
}

func TestRuntimeProbeDoesNotExposeTransportTarget(t *testing.T) {
	t.Parallel()

	result := testOpenAICompatible(context.Background(), nil, "http://127.0.0.1:1/private/provider", "secret", "model", 100*time.Millisecond)
	if result.OK || !strings.Contains(result.Message, "请检查地址、网络和超时配置") {
		t.Fatalf("unexpected probe result: %#v", result)
	}
	if strings.Contains(result.Message, "127.0.0.1") || strings.Contains(result.Message, "/private/provider") {
		t.Fatalf("probe message exposed target: %q", result.Message)
	}
}

func TestRuntimeProviderListRuntimeGroupsHidesLegacySingleAIConfig(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	groups, err := provider.ListRuntimeGroups(context.Background())
	if err != nil {
		t.Fatalf("ListRuntimeGroups() error = %v", err)
	}

	names := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		names[group.Name] = struct{}{}
	}

	for _, hidden := range []string{"ai.summary", "ai.title", "ai.flowchart"} {
		if _, ok := names[hidden]; ok {
			t.Fatalf("ListRuntimeGroups() includes hidden legacy group %q", hidden)
		}
	}
	for _, visible := range []string{"ai.provider_alert", "sidecar.linkparse", "miniapp.features"} {
		if _, ok := names[visible]; !ok {
			t.Fatalf("ListRuntimeGroups() missing visible group %q", visible)
		}
	}

	if got := provider.SummaryAI(context.Background()).BaseURL; got != "https://default.example.com/v1" {
		t.Fatalf("SummaryAI().BaseURL = %q, want default compatibility value", got)
	}
}

func TestRuntimeProviderMiniProgramFeaturesDefaultsAndUpdates(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	ctx := context.Background()

	if got := provider.MiniProgramFeatures(ctx).DietAssistantEnabled; got {
		t.Fatal("MiniProgramFeatures().DietAssistantEnabled = true, want false default")
	}

	if _, err := provider.UpdateRuntimeGroup(ctx, "tester", "req-miniapp-features", "miniapp.features", 0, map[string]any{
		"diet_assistant_enabled": true,
	}, nil); err != nil {
		t.Fatalf("UpdateRuntimeGroup(miniapp.features) error = %v", err)
	}

	if got := provider.MiniProgramFeatures(ctx).DietAssistantEnabled; !got {
		t.Fatal("MiniProgramFeatures().DietAssistantEnabled = false, want true after update")
	}
}

func TestRuntimeProviderRejectsLegacySingleAIAdminMutation(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	ctx := context.Background()

	if _, err := provider.UpdateRuntimeGroup(ctx, "tester", "req-hidden-update", "ai.summary", 0, map[string]any{
		"api_key": "secret",
	}, nil); err == nil {
		t.Fatal("UpdateRuntimeGroup(ai.summary) error = nil, want not found")
	}

	if _, err := provider.TestRuntimeGroup(ctx, "tester", "req-hidden-test", "ai.flowchart", map[string]any{
		"base_url": "https://example.com/v1",
		"model":    "gpt-test",
	}, nil); err == nil {
		t.Fatal("TestRuntimeGroup(ai.flowchart) error = nil, want not found")
	}
}

func TestRuntimeProviderTestRuntimeGroupSendsAIProviderAlertTestEmail(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	sender := &fakeAlertSender{}
	provider.SetAIAlertSender(sender)

	result, err := provider.TestRuntimeGroup(context.Background(), "tester", "req-4", "ai.provider_alert", map[string]any{
		"enabled":           true,
		"failure_threshold": 3,
		"smtp_host":         "smtp.qq.com",
		"smtp_port":         587,
		"smtp_username":     "bot@qq.com",
		"smtp_password":     "auth-code",
		"from_email":        "bot@qq.com",
		"to_emails":         "ops@qq.com",
	}, nil)
	if err != nil {
		t.Fatalf("TestRuntimeGroup(ai.provider_alert) error = %v", err)
	}
	if !result.OK {
		t.Fatalf("TestRuntimeGroup(ai.provider_alert).OK = false, message = %q", result.Message)
	}
	if len(sender.requests) != 1 {
		t.Fatalf("len(sender.requests) = %d, want 1", len(sender.requests))
	}
	if sender.requests[0].Config.ToEmails != "ops@qq.com" {
		t.Fatalf("sender.requests[0].Config.ToEmails = %q, want %q", sender.requests[0].Config.ToEmails, "ops@qq.com")
	}
}

func TestRuntimeProviderListSettingAuditsSupportsAdvancedFilters(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	ctx := context.Background()

	records := []settingAuditRecord{
		{
			GroupName:       "ai.routing.summary",
			SettingKey:      "ai.routing.summary.scene",
			Action:          "update",
			OldValueMasked:  `{"enabled":"false"}`,
			NewValueMasked:  `{"enabled":"true"}`,
			OperatorSubject: "alice",
			RequestID:       "req-1",
			CreatedAt:       "2026-04-25T10:00:00Z",
		},
		{
			GroupName:       "ai.routing.summary",
			SettingKey:      "ai.routing.summary.provider.primary",
			Action:          "test",
			OldValueMasked:  "",
			NewValueMasked:  "timeout",
			OperatorSubject: "bob",
			RequestID:       "req-2",
			CreatedAt:       "2026-04-25T11:00:00Z",
		},
		{
			GroupName:       "ai.routing.flowchart",
			SettingKey:      "ai.routing.flowchart.provider.fallback",
			Action:          "update",
			OldValueMasked:  "",
			NewValueMasked:  `{"model":"gpt-image-2"}`,
			OperatorSubject: "alice-admin",
			RequestID:       "req-3",
			CreatedAt:       "2026-04-25T12:00:00Z",
		},
	}

	for _, record := range records {
		if err := provider.repo.InsertSettingAudit(ctx, record); err != nil {
			t.Fatalf("InsertSettingAudit(%q) error = %v", record.SettingKey, err)
		}
	}

	result, err := provider.ListSettingAudits(ctx, SettingAuditFilter{
		GroupName:       "ai.routing.summary",
		Action:          "test",
		OperatorSubject: "bo",
		SettingKey:      "provider",
		TimeFrom:        "2026-04-25T10:30:00Z",
		TimeTo:          "2026-04-25T11:30:00Z",
		Page:            1,
		PageSize:        50,
	})
	if err != nil {
		t.Fatalf("ListSettingAudits() error = %v", err)
	}

	if result.Total != 1 {
		t.Fatalf("ListSettingAudits().Total = %d, want 1", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("len(ListSettingAudits().Items) = %d, want 1", len(result.Items))
	}
	if got := result.Items[0].RequestID; got != "req-2" {
		t.Fatalf("ListSettingAudits().Items[0].RequestID = %q, want %q", got, "req-2")
	}
	if got := result.PageSize; got != 50 {
		t.Fatalf("ListSettingAudits().PageSize = %d, want 50", got)
	}
}

func newRuntimeProviderForTest(t *testing.T) *RuntimeProvider {
	return newRuntimeProviderForTestWithOptions(t, RuntimeProviderOptions{})
}

func newRuntimeProviderForTestWithOptions(t *testing.T, opts RuntimeProviderOptions) *RuntimeProvider {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	statements := []string{
		`CREATE TABLE app_runtime_setting_groups (
			group_name TEXT PRIMARY KEY,
			version INTEGER NOT NULL DEFAULT 1,
			updated_by_subject TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE app_runtime_settings (
			key TEXT PRIMARY KEY,
			group_name TEXT NOT NULL,
			value_text TEXT NOT NULL DEFAULT '',
			value_ciphertext TEXT NOT NULL DEFAULT '',
			value_type TEXT NOT NULL DEFAULT 'string',
			is_secret INTEGER NOT NULL DEFAULT 0,
			is_restart_required INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT '',
			updated_by_subject TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE app_setting_audits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			group_name TEXT NOT NULL DEFAULT '',
			setting_key TEXT NOT NULL,
			action TEXT NOT NULL,
			old_value_masked TEXT NOT NULL DEFAULT '',
			new_value_masked TEXT NOT NULL DEFAULT '',
			operator_subject TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("db.Exec(%q) error = %v", statement, err)
		}
	}

	repo := NewRepository(db)
	return NewRuntimeProviderWithOptions(repo, "test-secret", config.Config{
		AIBaseURL:                  "https://default.example.com/v1",
		AIModel:                    "default-model",
		AITimeoutSeconds:           30,
		AIFlowchartTimeoutSeconds:  45,
		AITitleTimeoutSeconds:      3,
		AITitleMaxTokens:           64,
		AIAlertFailureThreshold:    3,
		AIAlertSMTPHost:            "smtp.qq.com",
		AIAlertSMTPPort:            587,
		LinkparseSidecarTimeoutSec: 150,
	}, opts)
}

type fakeAlertSender struct {
	requests []aialert.SendRequest
}

func appSettingsResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func (f *fakeAlertSender) Send(_ context.Context, request aialert.SendRequest) error {
	f.requests = append(f.requests, request)
	return nil
}
