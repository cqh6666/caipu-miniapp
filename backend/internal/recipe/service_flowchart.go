package recipe

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func (s *Service) GenerateFlowchart(ctx context.Context, userID int64, recipeID string) (Recipe, error) {
	if !s.flowchartEnabled || s.flowchart == nil || !s.flowchart.IsConfigured() {
		return Recipe{}, common.NewAppError(common.CodeInternalServer, "flowchart generation is not configured", http.StatusServiceUnavailable)
	}

	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return Recipe{}, err
	}

	if !canGenerateFlowchartForRecipe(current) {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "please complete key recipe steps before generating a flowchart", http.StatusBadRequest)
	}

	switch current.FlowchartStatus {
	case FlowchartStatusPending, FlowchartStatusProcessing:
		return s.decorateRecipeRuntimeState(ctx, current), nil
	}

	now := time.Now().Format(time.RFC3339)
	if err := s.repo.QueueFlowchart(ctx, current.ID, now); errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	} else if err != nil {
		return Recipe{}, err
	}

	current.FlowchartStatus = FlowchartStatusPending
	current.FlowchartError = ""
	current.FlowchartRequestedAt = now
	current.FlowchartFinishedAt = ""
	return s.decorateRecipeRuntimeState(ctx, current), nil
}
