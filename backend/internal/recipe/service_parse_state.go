package recipe

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
)

func applyCreateParseState(item *Recipe, req createRecipeRequest, now string) {
	if item == nil {
		return
	}

	platform := linkparse.DetectParsePlatform(req.Link)
	if platform != "" && shouldQueueAutoParse(req.Link, req.ParsedContent, req.MealType, req.Title, req.Ingredient) {
		item.ParseStatus = ParseStatusPending
		item.ParseSource = platform
		item.ParseError = ""
		item.ParseRequestedAt = now
		item.ParseFinishedAt = ""
		item.ParseAttempts = 0
		item.ParseNextAttemptAt = ""
		item.ParseLastErrorType = ""
		item.ParseProcessingStartedAt = ""
		return
	}

	item.ParseStatus = ParseStatusIdle
	item.ParseSource = ""
	item.ParseError = ""
	item.ParseRequestedAt = ""
	item.ParseFinishedAt = ""
	item.ParseAttempts = 0
	item.ParseNextAttemptAt = ""
	item.ParseLastErrorType = ""
	item.ParseProcessingStartedAt = ""
}

func applyUpdateParseState(item *Recipe, current Recipe, req updateRecipeRequest, now string) {
	if item == nil {
		return
	}

	linkChanged := strings.TrimSpace(req.Link) != strings.TrimSpace(current.Link)
	platform := linkparse.DetectParsePlatform(req.Link)
	switch {
	case platform != "" && (linkChanged || shouldQueueAutoParse(req.Link, req.ParsedContent, req.MealType, req.Title, req.Ingredient)):
		item.ParseStatus = ParseStatusPending
		item.ParseSource = platform
		item.ParseError = ""
		item.ParseRequestedAt = now
		item.ParseFinishedAt = ""
		item.ParseAttempts = 0
		item.ParseNextAttemptAt = ""
		item.ParseLastErrorType = ""
		item.ParseProcessingStartedAt = ""
	case platform != "":
		item.ParseStatus = current.ParseStatus
		item.ParseSource = current.ParseSource
		item.ParseError = current.ParseError
		item.ParseRequestedAt = current.ParseRequestedAt
		item.ParseFinishedAt = current.ParseFinishedAt
		item.ParseAttempts = current.ParseAttempts
		item.ParseNextAttemptAt = current.ParseNextAttemptAt
		item.ParseLastErrorType = current.ParseLastErrorType
		item.ParseProcessingStartedAt = current.ParseProcessingStartedAt
	default:
		item.ParseStatus = ParseStatusIdle
		item.ParseSource = ""
		item.ParseError = ""
		item.ParseRequestedAt = ""
		item.ParseFinishedAt = ""
		item.ParseAttempts = 0
		item.ParseNextAttemptAt = ""
		item.ParseLastErrorType = ""
		item.ParseProcessingStartedAt = ""
	}
}

func resolveCreateParsedContentEditedState(item Recipe, req createRecipeRequest) bool {
	if req.ParsedContentEdited != nil {
		return *req.ParsedContentEdited
	}

	return hasUserProvidedParsedContent(item.ParsedContent, item.MealType, item.Title, item.Ingredient)
}

func resolveUpdateParsedContentEditedState(current, next Recipe, req updateRecipeRequest) bool {
	if !parsedContentSlicesEqual(current.ParsedContent, next.ParsedContent) {
		if req.ParsedContentEdited != nil {
			return *req.ParsedContentEdited
		}
		return hasUserProvidedParsedContent(next.ParsedContent, next.MealType, next.Title, next.Ingredient)
	}

	return current.ParsedContentEdited
}

func (s *Service) RequeueAutoParse(ctx context.Context, userID int64, recipeID string) (Recipe, error) {
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return Recipe{}, err
	}

	platform := linkparse.DetectParsePlatform(current.Link)
	if platform == "" {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "only supported links can be reparsed", http.StatusBadRequest)
	}

	switch current.ParseStatus {
	case ParseStatusPending, ParseStatusProcessing:
		return s.decorateRecipeRuntimeState(ctx, current), nil
	}

	now := time.Now().Format(time.RFC3339)
	if err := s.repo.RequeueAutoParse(ctx, recipeID, userID, platform, now); errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	} else if err != nil {
		return Recipe{}, err
	}

	current.ParseStatus = ParseStatusPending
	current.ParseSource = platform
	current.ParseError = ""
	current.ParseRequestedAt = now
	current.ParseFinishedAt = ""
	current.ParseAttempts = 0
	current.ParseNextAttemptAt = ""
	current.ParseLastErrorType = ""
	current.ParseProcessingStartedAt = ""
	current.UpdatedBy = userID
	current.UpdatedAt = now
	return s.decorateRecipeRuntimeState(ctx, current), nil
}

func (s *Service) decorateRecipeRuntimeState(ctx context.Context, item Recipe) Recipe {
	item = s.decorateParseEstimate(ctx, item)
	item = s.decorateFlowchartEstimate(ctx, item)
	return item
}

func (s *Service) decorateParseEstimate(ctx context.Context, item Recipe) Recipe {
	if s == nil || s.repo == nil {
		return item
	}

	switch item.ParseStatus {
	case ParseStatusPending:
		if !s.autoParseEstimate.enabled {
			return item
		}
		ahead, err := s.repo.CountPendingAutoParseAhead(ctx, item)
		if err != nil {
			return item
		}
		processing, err := s.repo.CountProcessingAutoParse(ctx)
		if err != nil {
			return item
		}
		item.ParseQueueAhead = ahead + processing
		item.ParseEstimatedWait = estimatePendingQueueWaitSeconds(s.autoParseEstimate, item.ParseQueueAhead)
	case ParseStatusProcessing:
		if !s.autoParseEstimate.enabled {
			return item
		}
		item.ParseQueueAhead = 0
		item.ParseEstimatedWait = estimateProcessingQueueWaitSeconds(s.autoParseEstimate)
	default:
		item.ParseQueueAhead = 0
		item.ParseEstimatedWait = 0
	}

	return item
}

func (s *Service) decorateFlowchartEstimate(ctx context.Context, item Recipe) Recipe {
	if s == nil || s.repo == nil {
		return item
	}

	switch item.FlowchartStatus {
	case FlowchartStatusPending:
		if !s.flowchartEstimate.enabled {
			return item
		}
		ahead, err := s.repo.CountPendingFlowchartAhead(ctx, item)
		if err != nil {
			return item
		}
		processing, err := s.repo.CountProcessingFlowcharts(ctx)
		if err != nil {
			return item
		}
		item.FlowchartQueueAhead = ahead + processing
		item.FlowchartEstimatedWait = estimatePendingQueueWaitSeconds(s.flowchartEstimate, item.FlowchartQueueAhead)
	case FlowchartStatusProcessing:
		if !s.flowchartEstimate.enabled {
			return item
		}
		item.FlowchartQueueAhead = 0
		item.FlowchartEstimatedWait = estimateProcessingQueueWaitSeconds(s.flowchartEstimate)
	default:
		item.FlowchartQueueAhead = 0
		item.FlowchartEstimatedWait = 0
	}

	return item
}
