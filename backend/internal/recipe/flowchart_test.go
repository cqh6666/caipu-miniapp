package recipe

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func TestExtractFlowchartImageURLSupportsDataURL(t *testing.T) {
	t.Parallel()

	content := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aF9sAAAAASUVORK5CYII="
	if got := extractFlowchartImageURL(content); got != content {
		t.Fatalf("extractFlowchartImageURL(dataURL) = %q, want %q", got, content)
	}
}

func TestFlowchartClientGenerateSupportsImageGenerationsB64(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aF9sAAAAASUVORK5CYII="}]}`))
	}))
	defer server.Close()

	client := (&FlowchartGenerator{
		defaultConfig: FlowchartRuntimeConfig{
			BaseURL:        server.URL,
			Model:          "gpt-image-2",
			EndpointMode:   "images_generations",
			ResponseFormat: "b64_json",
		},
	}).clientFor(context.Background())
	if client == nil {
		t.Fatal("clientFor() = nil")
	}

	content, err := client.generate(context.Background(), "测试流程图 prompt")
	if err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	if got := extractFlowchartImageURL(content); got == "" {
		t.Fatalf("extractFlowchartImageURL(generate()) = empty, content = %q", content)
	}
}
