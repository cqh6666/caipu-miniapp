package app

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/appsettings"
	"github.com/cqh6666/caipu-miniapp/backend/internal/dietassistant"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
)

type runtimeProviderStub struct {
	summary   appsettings.SummaryAIConfig
	title     appsettings.TitleAIConfig
	flowchart appsettings.FlowchartAIConfig
	sidecar   appsettings.LinkparseSidecarConfig
}

func (s runtimeProviderStub) SummaryAI(context.Context) appsettings.SummaryAIConfig {
	return s.summary
}

func (s runtimeProviderStub) TitleAI(context.Context) appsettings.TitleAIConfig {
	return s.title
}

func (s runtimeProviderStub) FlowchartAI(context.Context) appsettings.FlowchartAIConfig {
	return s.flowchart
}

func (s runtimeProviderStub) LinkparseSidecar(context.Context) appsettings.LinkparseSidecarConfig {
	return s.sidecar
}

func TestBuildRuntimeConfigLoaders(t *testing.T) {
	provider := runtimeProviderStub{
		summary: appsettings.SummaryAIConfig{
			BaseURL: "https://summary.example.com/v1",
			APIKey:  "summary-secret",
			Model:   "summary-model",
			Timeout: 12 * time.Second,
		},
		title: appsettings.TitleAIConfig{
			Enabled:     true,
			BaseURL:     "https://title.example.com/v1",
			APIKey:      "title-secret",
			Model:       "title-model",
			Stream:      true,
			Temperature: 0.3,
			MaxTokens:   64,
			Timeout:     4 * time.Second,
		},
		flowchart: appsettings.FlowchartAIConfig{
			BaseURL:        "https://flowchart.example.com/v1",
			APIKey:         "flowchart-secret",
			Model:          "flowchart-model",
			EndpointMode:   "images_generations",
			ResponseFormat: "b64_json",
			Timeout:        45 * time.Second,
		},
		sidecar: appsettings.LinkparseSidecarConfig{
			Enabled: true,
			BaseURL: "https://sidecar.example.com",
			APIKey:  "sidecar-secret",
			Timeout: 20 * time.Second,
		},
	}

	linkConfig := buildLinkParseRuntimeConfigLoader(provider)(context.Background())
	if linkConfig.SummaryAI.BaseURL != provider.summary.BaseURL ||
		linkConfig.SummaryAI.APIKey != provider.summary.APIKey ||
		linkConfig.SummaryAI.Model != provider.summary.Model ||
		linkConfig.SummaryAI.Timeout != provider.summary.Timeout {
		t.Fatalf("summary config mismatch: %#v", linkConfig.SummaryAI)
	}
	if linkConfig.TitleAI.BaseURL != provider.title.BaseURL ||
		linkConfig.TitleAI.APIKey != provider.title.APIKey ||
		linkConfig.TitleAI.Model != provider.title.Model ||
		linkConfig.TitleAI.Enabled != provider.title.Enabled ||
		linkConfig.TitleAI.Stream != provider.title.Stream ||
		linkConfig.TitleAI.Temperature != provider.title.Temperature ||
		linkConfig.TitleAI.MaxTokens != provider.title.MaxTokens ||
		linkConfig.TitleAI.Timeout != provider.title.Timeout {
		t.Fatalf("title config mismatch: %#v", linkConfig.TitleAI)
	}
	if !reflect.DeepEqual(linkConfig.LinkparseSidecar, linkparse.LinkparseSidecarConfig{
		Enabled: true,
		BaseURL: "https://sidecar.example.com",
		APIKey:  "sidecar-secret",
		Timeout: 20 * time.Second,
	}) {
		t.Fatalf("sidecar config mismatch: %#v", linkConfig.LinkparseSidecar)
	}

	flowchartConfig := buildFlowchartRuntimeConfigLoader(provider)(context.Background())
	if !reflect.DeepEqual(flowchartConfig, recipe.FlowchartRuntimeConfig{
		BaseURL:        provider.flowchart.BaseURL,
		APIKey:         provider.flowchart.APIKey,
		Model:          provider.flowchart.Model,
		EndpointMode:   provider.flowchart.EndpointMode,
		ResponseFormat: provider.flowchart.ResponseFormat,
		Timeout:        provider.flowchart.Timeout,
	}) {
		t.Fatalf("flowchart config mismatch: %#v", flowchartConfig)
	}
}

func TestBuildAIRoutingCompatibilityLoader(t *testing.T) {
	provider := runtimeProviderStub{
		summary: appsettings.SummaryAIConfig{
			BaseURL: "https://summary.example.com/v1",
			APIKey:  "summary-secret-1234",
			Model:   "summary-model",
			Timeout: 9 * time.Second,
		},
		title: appsettings.TitleAIConfig{
			Enabled:     true,
			Stream:      true,
			Temperature: 0.2,
			MaxTokens:   48,
			Timeout:     3 * time.Second,
		},
		flowchart: appsettings.FlowchartAIConfig{
			BaseURL:        "https://flowchart.example.com/v1",
			APIKey:         "flowchart-secret",
			Model:          "image-model",
			EndpointMode:   "images_generations",
			ResponseFormat: "b64_json",
			Timeout:        30 * time.Second,
		},
	}
	loader := buildAIRoutingCompatibilityLoader(provider)

	summary := loader(context.Background(), airouter.SceneSummary)
	if !summary.Enabled || len(summary.Providers) != 1 {
		t.Fatalf("unexpected summary compatibility config: %#v", summary)
	}
	if got := summary.Providers[0].APIKeyMasked; got != "summ...1234" {
		t.Fatalf("unexpected summary secret mask: %q", got)
	}

	title := loader(context.Background(), airouter.SceneTitle)
	if !title.Enabled || title.Strategy != airouter.StrategyRoundRobinFailover {
		t.Fatalf("unexpected title compatibility config: %#v", title)
	}
	if got := title.Providers[0]; got.BaseURL != provider.summary.BaseURL ||
		got.APIKey != provider.summary.APIKey || got.Model != provider.summary.Model {
		t.Fatalf("title fallback mismatch: %#v", got)
	}
	if !title.RequestOptions.Stream || title.RequestOptions.MaxTokens != 48 {
		t.Fatalf("title request options mismatch: %#v", title.RequestOptions)
	}

	flowchart := loader(context.Background(), airouter.SceneFlowchart)
	if got := flowchart.Providers[0]; got.EndpointMode != airouter.EndpointModeImagesGenerations ||
		got.ResponseFormat != airouter.ResponseFormatB64JSON {
		t.Fatalf("flowchart protocol options mismatch: %#v", got)
	}

	unknown := loader(context.Background(), airouter.Scene("unknown"))
	if unknown.Enabled || len(unknown.Providers) != 0 || unknown.MaxAttempts != 1 {
		t.Fatalf("unexpected unknown scene config: %#v", unknown)
	}
}

func TestBuildAIRoutingTestInputBuilder(t *testing.T) {
	builder := buildAIRoutingTestInputBuilder()
	for _, scene := range airouter.AllScenes() {
		input, ok := builder(scene)
		if !ok || len(input.Messages) == 0 {
			t.Fatalf("scene %s has no route test input: %#v", scene, input)
		}
	}
	if _, ok := builder(airouter.Scene("unknown")); ok {
		t.Fatal("unknown scene should not have route test input")
	}
}

type dietAssistantRecipeServiceStub struct {
	items       []recipe.Recipe
	item        recipe.Recipe
	listFilter  recipe.ListFilter
	createInput recipe.CreateInput
}

func (s *dietAssistantRecipeServiceStub) ListByKitchenID(_ context.Context, _, _ int64, filter recipe.ListFilter) ([]recipe.Recipe, error) {
	s.listFilter = filter
	return s.items, nil
}

func (s *dietAssistantRecipeServiceStub) CreateFromInput(_ context.Context, _, _ int64, input recipe.CreateInput) (recipe.Recipe, error) {
	s.createInput = input
	return s.item, nil
}

func (s *dietAssistantRecipeServiceStub) GetByID(context.Context, int64, string) (recipe.Recipe, error) {
	return s.item, nil
}

type dietAssistantLinkParserStub struct {
	outcome linkparse.RecipeParseOutcome
}

func (s dietAssistantLinkParserStub) ParseRecipeLink(context.Context, string) (linkparse.RecipeParseOutcome, error) {
	return s.outcome, nil
}

func TestBuildDietAssistantRecipeTools(t *testing.T) {
	recipeService := &dietAssistantRecipeServiceStub{
		items: []recipe.Recipe{
			{ID: "recipe-1", Title: "红烧排骨"},
			{ID: "recipe-2", Title: "番茄炒蛋"},
		},
		item: recipe.Recipe{
			ID:        "recipe-created",
			KitchenID: 7,
			Title:     "链接菜谱",
		},
	}
	parser := dietAssistantLinkParserStub{outcome: linkparse.RecipeParseOutcome{
		Source:       "xiaohongshu",
		SourceDetail: "  note  ",
		SummaryMode:  "  ai  ",
		Warnings:     []string{" warning ", "warning"},
		RecipeDraft: linkparse.RecipeDraft{
			Link:      "https://example.com/recipe",
			ImageURL:  "https://img.example.com/main.jpg",
			ImageURLs: []string{"https://img.example.com/a.jpg", "https://img.example.com/a.jpg"},
			ParsedContent: linkparse.ParsedContent{
				MainIngredients:      []string{" 牛肉 ", "牛肉"},
				SecondaryIngredients: []string{"盐"},
				Steps: []linkparse.ParsedStep{
					{Title: " 备料 ", Detail: " 切块 "},
					{},
				},
			},
		},
	}}
	tools := buildDietAssistantRecipeTools(recipeService, parser)

	count, err := tools.countRecipes(context.Background(), dietassistant.RecipeCountInput{
		UserID: 1, KitchenID: 7, MealType: "dinner", Status: "want",
	})
	if err != nil || count != 2 {
		t.Fatalf("count recipes: count=%d err=%v", count, err)
	}
	if recipeService.listFilter.MealType != "dinner" || recipeService.listFilter.Status != "want" {
		t.Fatalf("count filter mismatch: %#v", recipeService.listFilter)
	}

	searched, err := tools.searchRecipes(context.Background(), dietassistant.RecipeSearchInput{
		UserID: 1, KitchenID: 7, Keyword: " 排骨 ", SearchScope: "title", Limit: 1,
	})
	if err != nil || len(searched) != 1 || searched[0].ID != "recipe-1" {
		t.Fatalf("search recipes mismatch: %#v err=%v", searched, err)
	}
	if recipeService.listFilter.TitleKeyword != "排骨" {
		t.Fatalf("search filter mismatch: %#v", recipeService.listFilter)
	}

	created, err := tools.createFromURL(context.Background(), dietassistant.RecipeFromURLInput{
		UserID: 1, KitchenID: 7, URL: "https://example.com/recipe", MealType: "dinner", Status: "want",
	})
	if err != nil {
		t.Fatalf("create from URL: %v", err)
	}
	if recipeService.createInput.Title != "链接菜谱" || recipeService.createInput.ParsedContentEdited == nil || *recipeService.createInput.ParsedContentEdited {
		t.Fatalf("create input mismatch: %#v", recipeService.createInput)
	}
	if !reflect.DeepEqual(recipeService.createInput.ImageURLs, []string{
		"https://img.example.com/a.jpg",
		"https://img.example.com/main.jpg",
	}) {
		t.Fatalf("image URL normalization mismatch: %#v", recipeService.createInput.ImageURLs)
	}
	if len(recipeService.createInput.ParsedContent.Steps) != 1 ||
		recipeService.createInput.ParsedContent.Steps[0].Title != "备料" {
		t.Fatalf("parsed content mismatch: %#v", recipeService.createInput.ParsedContent)
	}
	if created.SourceDetail != "note" || created.SummaryMode != "ai" ||
		len(created.Warnings) != 1 || created.StepsCount != 1 {
		t.Fatalf("create result mismatch: %#v", created)
	}

	recipeService.item.KitchenID = 8
	_, err = tools.getRecipeByID(context.Background(), dietassistant.RecipeGetInput{
		UserID: 1, KitchenID: 7, RecipeID: "recipe-created",
	})
	if err == nil || !strings.Contains(err.Error(), "current kitchen") {
		t.Fatalf("expected kitchen isolation error, got %v", err)
	}
}
