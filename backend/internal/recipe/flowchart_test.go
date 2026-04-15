package recipe

import (
	"context"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

func TestFlowchartGeneratorIsConfiguredRequiresAvailableRoute(t *testing.T) {
	t.Parallel()

	generator := NewFlowchartGenerator(FlowchartOptions{
		AIRouter: airouter.NewService(nil, "test-secret", func(context.Context, airouter.Scene) airouter.SceneConfig {
			return airouter.SceneConfig{
				Scene:    airouter.SceneFlowchart,
				Enabled:  false,
				Strategy: airouter.StrategyPriorityFailover,
			}
		}, nil, nil),
	}, upload.NewService(t.TempDir(), "https://static.example.com", 10))

	if generator.IsConfigured() {
		t.Fatalf("IsConfigured() = true, want false")
	}
}

func TestFlowchartGeneratorIsConfiguredIgnoresEmptyRuntimeLoader(t *testing.T) {
	t.Parallel()

	generator := NewFlowchartGenerator(FlowchartOptions{
		RuntimeConfigLoader: func(context.Context) FlowchartRuntimeConfig {
			return FlowchartRuntimeConfig{}
		},
	}, upload.NewService(t.TempDir(), "https://static.example.com", 10))

	if generator.IsConfigured() {
		t.Fatalf("IsConfigured() = true, want false")
	}
}
