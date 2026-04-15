package airouter

import "testing"

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
			valid:   `![test](https://example.com/test.png)`,
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

	service := NewService(nil, "unit-test-secret", nil, nil)
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
