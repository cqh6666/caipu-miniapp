package appsettings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerGetPublicAppConfigReturnsMiniProgramFeatures(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	if _, err := provider.UpdateRuntimeGroup(context.Background(), "tester", "req-public-config", "miniapp.features", 0, map[string]any{
		"diet_assistant_enabled": false,
	}, nil); err != nil {
		t.Fatalf("UpdateRuntimeGroup(miniapp.features) error = %v", err)
	}

	handler := NewHandler(nil, provider)
	request := httptest.NewRequest(http.MethodGet, "/api/public/app-config", nil)
	recorder := httptest.NewRecorder()

	handler.GetPublicAppConfig(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("recorder.Code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Code int `json:"code"`
		Data struct {
			Features MiniProgramFeatureConfig `json:"features"`
		} `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("response.Code = %d, want 0", response.Code)
	}
	if response.Data.Features.DietAssistantEnabled {
		t.Fatal("DietAssistantEnabled = true, want false")
	}
}

func TestHandlerGetPublicAppConfigDefaultsDietAssistantToDisabled(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/public/app-config", nil)
	recorder := httptest.NewRecorder()

	handler.GetPublicAppConfig(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("recorder.Code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Data struct {
			Features MiniProgramFeatureConfig `json:"features"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.Data.Features.DietAssistantEnabled {
		t.Fatal("DietAssistantEnabled = true, want false without runtime provider")
	}
}
