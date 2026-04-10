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
