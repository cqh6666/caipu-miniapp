package recipe

import (
	"context"
	"errors"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

const tinyFlowchartPNGDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aF9sAAAAASUVORK5CYII="

type fakeFlowchartRouter struct {
	available bool
	result    airouter.ChatCompletionResult
	err       error
	scene     airouter.Scene
	input     airouter.ChatCompletionInput
}

func (f *fakeFlowchartRouter) IsSceneAvailable(context.Context, airouter.Scene) bool {
	return f != nil && f.available
}

func (f *fakeFlowchartRouter) RouteChat(_ context.Context, scene airouter.Scene, input airouter.ChatCompletionInput) (airouter.ChatCompletionResult, error) {
	f.scene = scene
	f.input = input
	if f.err != nil {
		return f.result, f.err
	}
	if input.ValidateContent != nil {
		if err := input.ValidateContent(f.result.Content); err != nil {
			return f.result, err
		}
	}
	return f.result, nil
}

func TestFlowchartGeneratorIsConfiguredRequiresAvailableRoute(t *testing.T) {
	t.Parallel()

	generator := NewFlowchartGenerator(FlowchartOptions{
		AIRouter: &fakeFlowchartRouter{available: false},
	}, upload.NewService(t.TempDir(), "https://static.example.com", 10))

	if generator.IsConfigured() {
		t.Fatalf("IsConfigured() = true, want false")
	}
}

func TestFlowchartGeneratorIsConfiguredRequiresRouter(t *testing.T) {
	t.Parallel()

	generator := NewFlowchartGenerator(FlowchartOptions{}, upload.NewService(t.TempDir(), "https://static.example.com", 10))
	if generator.IsConfigured() {
		t.Fatalf("IsConfigured() = true, want false")
	}
}

func TestFlowchartGeneratorGenerateUsesSameRouter(t *testing.T) {
	t.Parallel()

	router := &fakeFlowchartRouter{
		available: true,
		result: airouter.ChatCompletionResult{
			Content:         tinyFlowchartPNGDataURL,
			ProviderID:      "image-provider",
			Model:           "image-model",
			Strategy:        airouter.StrategyPriorityFailover,
			StartedProvider: "image-provider",
			AttemptCount:    1,
		},
	}
	generator := NewFlowchartGenerator(FlowchartOptions{AIRouter: router}, upload.NewService(t.TempDir(), "https://static.example.com", 10))

	result, err := generator.Generate(context.Background(), Recipe{
		Title:   "番茄牛腩",
		Summary: "酸甜软烂",
		ParsedContent: ParsedContent{
			MainIngredients: []string{"牛腩 500克", "番茄 3个"},
			Steps: []ParsedStep{
				{Title: "焯水", Detail: "牛腩冷水下锅焯水。"},
				{Title: "炒香", Detail: "番茄炒出汁后加入牛腩。"},
				{Title: "炖煮", Detail: "小火炖至牛腩软烂。"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.ImageURL == "" || result.Provider != "image-provider" || result.Model != "image-model" {
		t.Fatalf("Generate() result = %#v", result)
	}
	if router.scene != airouter.SceneFlowchart || router.input.ContentKind != "flowchart" {
		t.Fatalf("unexpected route call: scene=%q input=%#v", router.scene, router.input)
	}
}

func TestFlowchartGeneratorGeneratePreservesRouteFailureMetadata(t *testing.T) {
	t.Parallel()

	router := &fakeFlowchartRouter{
		available: true,
		result: airouter.ChatCompletionResult{
			ProviderID:   "failed-provider",
			Model:        "failed-model",
			AttemptCount: 2,
		},
		err: errors.New("route failed"),
	}
	generator := NewFlowchartGenerator(FlowchartOptions{AIRouter: router}, upload.NewService(t.TempDir(), "https://static.example.com", 10))

	result, err := generator.Generate(context.Background(), Recipe{
		Title: "番茄牛腩",
		ParsedContent: ParsedContent{Steps: []ParsedStep{
			{Title: "焯水", Detail: "牛腩焯水。"},
			{Title: "炒香", Detail: "番茄炒香。"},
			{Title: "炖煮", Detail: "小火炖煮。"},
		}},
	})
	if !errors.Is(err, router.err) {
		t.Fatalf("Generate() error = %v, want %v", err, router.err)
	}
	if result.Provider != "failed-provider" || result.AttemptCount != 2 {
		t.Fatalf("Generate() failure result = %#v", result)
	}
}

func TestExtractFlowchartImageURLSupportsDataURL(t *testing.T) {
	t.Parallel()

	if got := extractFlowchartImageURL(tinyFlowchartPNGDataURL); got != tinyFlowchartPNGDataURL {
		t.Fatalf("extractFlowchartImageURL(dataURL) = %q, want %q", got, tinyFlowchartPNGDataURL)
	}
}
