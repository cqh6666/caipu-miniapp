package airouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
)

func TestBuildSceneTestInputUsesSceneSpecificValidator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		scene   Scene
		valid   string
		invalid string
	}{
		{
			name:    "summary",
			scene:   SceneSummary,
			valid:   `{"title":"西红柿炒鸡蛋","steps":[{"title":"备料","detail":"切番茄打蛋"}]}`,
			invalid: `{"title":"西红柿炒鸡蛋","steps":[]}`,
		},
		{
			name:    "title",
			scene:   SceneTitle,
			valid:   `{"title":"西红柿炒鸡蛋"}`,
			invalid: `{"title":""}`,
		},
		{
			name:    "flowchart",
			scene:   SceneFlowchart,
			valid:   `data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aF9sAAAAASUVORK5CYII=`,
			invalid: `{"message":"no image"}`,
		},
	}

	for _, item := range cases {
		item := item
		t.Run(item.name, func(t *testing.T) {
			t.Parallel()

			input := buildSceneTestInput(item.scene)
			if input.ValidateContent == nil {
				t.Fatalf("buildSceneTestInput(%q).ValidateContent = nil", item.scene)
			}
			if err := input.ValidateContent(item.valid); err != nil {
				t.Fatalf("ValidateContent(valid) error = %v", err)
			}
			if err := input.ValidateContent(item.invalid); err == nil {
				t.Fatalf("ValidateContent(invalid) error = nil, want non-nil")
			}
		})
	}
}

func TestBuildSceneTestInputUsesLargerSummaryTokenBudget(t *testing.T) {
	t.Parallel()

	input := buildSceneTestInput(SceneSummary)
	if input.MaxTokens == nil {
		t.Fatal("buildSceneTestInput(summary).MaxTokens = nil")
	}
	if got := *input.MaxTokens; got != 1024 {
		t.Fatalf("buildSceneTestInput(summary).MaxTokens = %d, want 1024", got)
	}
}

func TestBuildSceneConfigRetainsEncryptedAPIKeyForRuntimeCalls(t *testing.T) {
	t.Parallel()

	service := NewService(nil, "unit-test-secret", nil, nil, nil)
	ciphertext, err := service.cipherBox.Encrypt("sk-test-12345678")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	config, err := service.buildSceneConfig(sceneRecord{
		Scene:       SceneSummary,
		Enabled:     true,
		Strategy:    StrategyPriorityFailover,
		MaxAttempts: 1,
	}, []providerRecord{
		{
			ID:           "summary-primary",
			Scene:        SceneSummary,
			Name:         "summary-primary",
			Adapter:      AdapterOpenAICompatible,
			Enabled:      true,
			Priority:     10,
			BaseURL:      "https://example.com/v1",
			APIKeyCipher: ciphertext,
			Model:        "gpt-test",
		},
	})
	if err != nil {
		t.Fatalf("buildSceneConfig() error = %v", err)
	}
	if len(config.Providers) != 1 {
		t.Fatalf("buildSceneConfig() providers = %d, want 1", len(config.Providers))
	}
	if got := config.Providers[0].APIKey; got != ciphertext {
		t.Fatalf("provider.APIKey = %q, want encrypted runtime value", got)
	}
	if !config.Providers[0].HasAPIKey {
		t.Fatal("provider.HasAPIKey = false, want true")
	}
	if got := config.Providers[0].APIKeyMasked; got == "" {
		t.Fatal("provider.APIKeyMasked = empty, want masked secret")
	}
}

func TestSceneTestInputUsesInjectedBuilder(t *testing.T) {
	t.Parallel()

	service := NewService(nil, "unit-test-secret", nil, nil, nil)
	service.SetTestInputBuilder(func(scene Scene) (ChatCompletionInput, bool) {
		if scene != SceneSummary {
			return ChatCompletionInput{}, false
		}
		return ChatCompletionInput{
			ContentKind: "custom-route-test",
		}, true
	})

	input := service.sceneTestInput(SceneSummary)
	if got := input.ContentKind; got != "custom-route-test" {
		t.Fatalf("sceneTestInput(summary).ContentKind = %q, want %q", got, "custom-route-test")
	}
}

func TestSceneUsesCompatibilityWhenNoRuntimeProviderIsAvailable(t *testing.T) {
	t.Parallel()

	config := SceneConfig{
		Scene:    SceneSummary,
		Enabled:  true,
		Strategy: StrategyPriorityFailover,
		Providers: []ProviderConfig{
			{
				ID:      "summary-primary",
				Enabled: true,
			},
		},
	}

	if !sceneUsesCompatibility(config) {
		t.Fatalf("sceneUsesCompatibility() = false, want true")
	}
}

func TestRouteChatRouteTestSkipsProviderAlerts(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"title\":\"西红柿炒鸡蛋\",\"ingredient\":\"鸡蛋 番茄\",\"summary\":\"家常快手菜\",\"mainIngredients\":[\"番茄\"],\"secondaryIngredients\":[\"鸡蛋\"],\"steps\":[{\"title\":\"备料\",\"detail\":\"切番茄，打蛋。\"}],\"note\":\"\"}"}}]}`))
	}))
	defer server.Close()

	alerts := &fakeAlertTracker{}
	service := NewService(nil, "test-secret", func(context.Context, Scene) SceneConfig {
		return SceneConfig{}
	}, nil, alerts)

	_, err := service.routeChat(context.Background(), SceneConfig{
		Scene:       SceneSummary,
		Enabled:     true,
		Strategy:    StrategyPriorityFailover,
		MaxAttempts: 1,
		RetryOn:     DefaultRetryOn(),
		Breaker:     DefaultBreakerConfig(),
		Providers: []ProviderConfig{
			{
				ID:             "summary-main",
				Name:           "主节点",
				Adapter:        AdapterOpenAICompatible,
				Enabled:        true,
				Priority:       10,
				BaseURL:        server.URL,
				Model:          "gpt-test",
				TimeoutSeconds: 5,
			},
		},
	}, buildSceneTestInput(SceneSummary))
	if err != nil {
		t.Fatalf("routeChat() error = %v", err)
	}
	if alerts.successCount != 0 || alerts.failureCount != 0 {
		t.Fatalf("alerts = %+v, want zero", alerts)
	}
}

func TestRouteChatFlowchartUsesMessageImagesWhenContentIsEmpty(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":null,"images":[{"type":"image_url","image_url":{"url":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aF9sAAAAASUVORK5CYII="}}]}}]}`))
	}))
	defer server.Close()

	service := NewService(nil, "test-secret", func(context.Context, Scene) SceneConfig {
		return SceneConfig{}
	}, nil, nil)

	result, err := service.routeChat(context.Background(), SceneConfig{
		Scene:       SceneFlowchart,
		Enabled:     true,
		Strategy:    StrategyPriorityFailover,
		MaxAttempts: 1,
		RetryOn:     DefaultRetryOn(),
		Breaker:     DefaultBreakerConfig(),
		Providers: []ProviderConfig{
			{
				ID:             "flowchart-main",
				Name:           "主节点",
				Adapter:        AdapterOpenAICompatible,
				Enabled:        true,
				Priority:       10,
				BaseURL:        server.URL,
				Model:          "gpt-test",
				TimeoutSeconds: 5,
				Scene:          SceneFlowchart,
			},
		},
	}, buildSceneTestInput(SceneFlowchart))
	if err != nil {
		t.Fatalf("routeChat() error = %v", err)
	}
	if !strings.HasPrefix(result.Content, "data:image/png;base64,") {
		t.Fatalf("routeChat() content = %q, want data image url", result.Content)
	}
}

type fakeAlertTracker struct {
	successCount int
	failureCount int
}

func (f *fakeAlertTracker) RecordSuccess(context.Context, aialert.Event) {
	f.successCount++
}

func (f *fakeAlertTracker) RecordFailure(context.Context, aialert.Event) {
	f.failureCount++
}
