package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
)

func TestHandlerGetAIRoutingAlertsOverview(t *testing.T) {
	t.Parallel()

	handler := NewHandler(
		stubAuthService{subject: "admin"},
		nil,
		nil,
		nil,
		nil,
		nil,
		stubAlertOverviewProvider{
			overview: aialert.Overview{
				GeneratedAt:       "2026-04-25T10:00:00Z",
				Enabled:           true,
				FailureThreshold:  3,
				HasDeliveryConfig: true,
				ActiveAlertCount:  1,
				LatestAlertedAt:   "2026-04-25T09:58:00Z",
				Items: []aialert.OverviewItem{
					{
						ProviderID:          "summary-main",
						ProviderName:        "主节点",
						Scene:               "summary",
						Model:               "gpt-test",
						ConsecutiveFailures: 4,
						LastStatus:          "failed",
						LastErrorType:       "timeout",
						LastErrorMessage:    "request timeout",
						LastRequestID:       "req-1",
						LastFailedAt:        "2026-04-25T09:57:00Z",
						LastRecoveredAt:     "",
						LastAlertedAt:       "2026-04-25T09:58:00Z",
						UpdatedAt:           "2026-04-25T09:59:00Z",
						ThresholdReached:    true,
					},
				},
			},
		},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/admin/ai-routing/alerts/overview", nil)
	recorder := httptest.NewRecorder()

	handler.GetAIRoutingAlertsOverview(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("recorder.Code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Overview aialert.Overview `json:"overview"`
		} `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if response.Code != 0 || response.Message != "ok" {
		t.Fatalf("response envelope = %+v, want code=0 message=ok", response)
	}
	if response.Data.Overview.GeneratedAt != "2026-04-25T10:00:00Z" {
		t.Fatalf("GeneratedAt = %q, want %q", response.Data.Overview.GeneratedAt, "2026-04-25T10:00:00Z")
	}
	if response.Data.Overview.ActiveAlertCount != 1 {
		t.Fatalf("ActiveAlertCount = %d, want 1", response.Data.Overview.ActiveAlertCount)
	}
	if len(response.Data.Overview.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(response.Data.Overview.Items))
	}
	item := response.Data.Overview.Items[0]
	if item.ProviderID != "summary-main" || !item.ThresholdReached {
		t.Fatalf("item = %+v, want providerID=summary-main thresholdReached=true", item)
	}
}

type stubAuthService struct {
	subject string
}

func (s stubAuthService) Login(context.Context, string, string) (string, error) {
	return "", nil
}

func (s stubAuthService) CurrentSubject(context.Context) (string, error) {
	return s.subject, nil
}

func (s stubAuthService) BuildSessionCookie(string) *http.Cookie {
	return &http.Cookie{}
}

func (s stubAuthService) BuildLogoutCookie() *http.Cookie {
	return &http.Cookie{}
}

type stubAlertOverviewProvider struct {
	overview aialert.Overview
	err      error
}

func (p stubAlertOverviewProvider) Overview(context.Context) (aialert.Overview, error) {
	return p.overview, p.err
}
