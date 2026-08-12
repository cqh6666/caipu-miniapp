package recipe

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upload"
)

type FlowchartAIRouter interface {
	IsSceneAvailable(context.Context, airouter.Scene) bool
	RouteChat(context.Context, airouter.Scene, airouter.ChatCompletionInput) (airouter.ChatCompletionResult, error)
}

type FlowchartOptions struct {
	AIRouter FlowchartAIRouter
}

type FlowchartGenerator struct {
	aiRouter FlowchartAIRouter
	uploader *upload.Service
}

type FlowchartResult struct {
	ImageURL        string
	SourceHash      string
	Provider        string
	Model           string
	FallbackUsed    bool
	AttemptCount    int
	StartedProvider string
	RouteStrategy   string
}

func NewFlowchartGenerator(opts FlowchartOptions, uploader *upload.Service) *FlowchartGenerator {
	if uploader == nil {
		return nil
	}
	return &FlowchartGenerator{
		aiRouter: opts.AIRouter,
		uploader: uploader,
	}
}

func (g *FlowchartGenerator) IsConfigured() bool {
	return g != nil && g.uploader != nil && g.aiRouter != nil &&
		g.aiRouter.IsSceneAvailable(context.Background(), airouter.SceneFlowchart)
}

func (g *FlowchartGenerator) Generate(ctx context.Context, item Recipe) (FlowchartResult, error) {
	if g == nil || g.uploader == nil || g.aiRouter == nil ||
		!g.aiRouter.IsSceneAvailable(ctx, airouter.SceneFlowchart) {
		return FlowchartResult{}, common.NewAppError(common.CodeServiceUnavailable, "flowchart generation is not configured", http.StatusServiceUnavailable)
	}

	input, err := buildFlowchartPromptInput(item)
	if err != nil {
		return FlowchartResult{}, err
	}

	routeResult, routeErr := g.aiRouter.RouteChat(ctx, airouter.SceneFlowchart, airouter.ChatCompletionInput{
		Messages: []airouter.ChatMessage{
			{
				Role:    "system",
				Content: "你是一个料理流程图生成助手。请严格按用户要求生成手绘风格料理流程信息图，不要输出额外解释。",
			},
			{
				Role:    "user",
				Content: buildFlowchartPrompt(input),
			},
		},
		Temperature:    floatPtr(0.4),
		ContentKind:    "flowchart",
		AdditionalMeta: map[string]any{"content_kind": "flowchart"},
		ValidateContent: func(content string) error {
			if extractFlowchartImageURL(content) == "" {
				return fmt.Errorf("flowchart generation did not return an image")
			}
			return nil
		},
	})
	result := FlowchartResult{
		Provider:        routeResult.ProviderID,
		Model:           routeResult.Model,
		FallbackUsed:    routeResult.FallbackUsed,
		AttemptCount:    routeResult.AttemptCount,
		StartedProvider: routeResult.StartedProvider,
		RouteStrategy:   string(routeResult.Strategy),
	}
	if routeErr != nil {
		return result, routeErr
	}

	remoteURL := extractFlowchartImageURL(routeResult.Content)
	if remoteURL == "" {
		return FlowchartResult{}, common.NewAppError(common.CodeInternalServer, "flowchart generation did not return an image", http.StatusBadGateway)
	}

	image, err := g.uploader.SaveRemoteImage(ctx, remoteURL)
	if err != nil {
		return FlowchartResult{}, err
	}
	result.ImageURL = image.URL
	result.SourceHash = hashFlowchartPromptInput(input)
	return result, nil
}
