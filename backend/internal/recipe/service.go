package recipe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
)

var (
	allowedMealTypes = map[string]struct{}{
		"breakfast": {},
		"main":      {},
	}
	allowedStatuses = map[string]struct{}{
		"wishlist": {},
		"done":     {},
	}
)

const (
	maxRecipeImages       = 9
	maxRecipeSummaryRunes = 24
	maxParsedSteps        = 6
)

type Service struct {
	repo              *Repository
	kitchen           *kitchen.Service
	flowchart         *FlowchartGenerator
	flowchartEnabled  bool
	autoParseEstimate queueEstimateConfig
	flowchartEstimate queueEstimateConfig
}

type ServiceOptions struct {
	Repo               *Repository
	KitchenService     *kitchen.Service
	Flowchart          *FlowchartGenerator
	FlowchartEnabled   bool
	AutoParseEnabled   bool
	AutoParseInterval  time.Duration
	AutoParseBatchSize int
	FlowchartInterval  time.Duration
	FlowchartBatchSize int
}

func NewService(opts ServiceOptions) *Service {
	return &Service{
		repo:             opts.Repo,
		kitchen:          opts.KitchenService,
		flowchart:        opts.Flowchart,
		flowchartEnabled: opts.FlowchartEnabled,
		autoParseEstimate: queueEstimateConfig{
			enabled:         opts.AutoParseEnabled,
			interval:        opts.AutoParseInterval,
			batchSize:       opts.AutoParseBatchSize,
			averageDuration: defaultAutoParseEstimatedDuration,
		},
		flowchartEstimate: queueEstimateConfig{
			enabled:         opts.FlowchartEnabled && opts.Flowchart != nil && opts.Flowchart.IsConfigured(),
			interval:        opts.FlowchartInterval,
			batchSize:       opts.FlowchartBatchSize,
			averageDuration: defaultFlowchartEstimatedDuration,
		},
	}
}

func (s *Service) ListByKitchenID(ctx context.Context, userID, kitchenID int64, filter ListFilter) ([]Recipe, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return nil, err
	}

	filter.MealType = strings.TrimSpace(filter.MealType)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Keyword = strings.TrimSpace(filter.Keyword)

	if filter.MealType != "" && !isAllowedMealType(filter.MealType) {
		return nil, common.NewAppError(common.CodeBadRequest, "invalid mealType", http.StatusBadRequest)
	}

	if filter.Status != "" && !isAllowedStatus(filter.Status) {
		return nil, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	return s.repo.ListByKitchenID(ctx, kitchenID, filter)
}

func (s *Service) Create(ctx context.Context, userID, kitchenID int64, req createRecipeRequest) (Recipe, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Recipe{}, err
	}

	item, err := normalizeRecipeInput(req.Title, req.Ingredient, req.Summary, req.Link, req.ImageURL, req.ImageURLs, req.MealType, req.Status, req.Note, req.ParsedContent)
	if err != nil {
		return Recipe{}, err
	}

	recipeID, err := common.NewPrefixedID("rec")
	if err != nil {
		return Recipe{}, fmt.Errorf("generate recipe id: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	item.ID = recipeID
	item.KitchenID = kitchenID
	item.CreatedBy = userID
	item.UpdatedBy = userID
	item.CreatedAt = now
	item.UpdatedAt = now
	applyCreateParseState(&item, req, now)
	item.ParsedContentEdited = resolveCreateParsedContentEditedState(item, req)

	created, err := s.repo.Create(ctx, item)
	if err != nil {
		return Recipe{}, err
	}
	return s.decorateRecipeRuntimeState(ctx, created), nil
}

func (s *Service) GetByID(ctx context.Context, userID int64, recipeID string) (Recipe, error) {
	item, err := s.repo.FindByID(ctx, recipeID)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	}
	if err != nil {
		return Recipe{}, err
	}

	if err := s.kitchen.EnsureMember(ctx, userID, item.KitchenID); err != nil {
		return Recipe{}, err
	}

	return s.decorateRecipeRuntimeState(ctx, item), nil
}

func (s *Service) Update(ctx context.Context, userID int64, recipeID string, req updateRecipeRequest) (Recipe, error) {
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return Recipe{}, err
	}

	summary := req.Summary
	if strings.TrimSpace(summary) == "" {
		summary = current.Summary
	}

	next, err := normalizeRecipeInput(req.Title, req.Ingredient, summary, req.Link, req.ImageURL, req.ImageURLs, req.MealType, req.Status, req.Note, req.ParsedContent)
	if err != nil {
		return Recipe{}, err
	}

	next.ID = current.ID
	next.KitchenID = current.KitchenID
	next.PinnedAt = current.PinnedAt
	next.FlowchartImageURL = current.FlowchartImageURL
	next.FlowchartStatus = current.FlowchartStatus
	next.FlowchartError = current.FlowchartError
	next.FlowchartRequestedAt = current.FlowchartRequestedAt
	next.FlowchartFinishedAt = current.FlowchartFinishedAt
	next.FlowchartUpdatedAt = current.FlowchartUpdatedAt
	next.FlowchartSourceHash = current.FlowchartSourceHash
	next.CreatedBy = current.CreatedBy
	next.CreatedAt = current.CreatedAt
	next.UpdatedBy = userID
	next.UpdatedAt = time.Now().Format(time.RFC3339)
	applyUpdateParseState(&next, current, req, next.UpdatedAt)
	next.ParsedContentEdited = resolveUpdateParsedContentEditedState(current, next, req)

	updated, err := s.repo.Update(ctx, next)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	}
	if err != nil {
		return Recipe{}, err
	}

	return s.decorateRecipeRuntimeState(ctx, updated), nil
}

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
		return
	}

	item.ParseStatus = ParseStatusIdle
	item.ParseSource = ""
	item.ParseError = ""
	item.ParseRequestedAt = ""
	item.ParseFinishedAt = ""
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
	case platform != "":
		item.ParseStatus = current.ParseStatus
		item.ParseSource = current.ParseSource
		item.ParseError = current.ParseError
		item.ParseRequestedAt = current.ParseRequestedAt
		item.ParseFinishedAt = current.ParseFinishedAt
	default:
		item.ParseStatus = ParseStatusIdle
		item.ParseSource = ""
		item.ParseError = ""
		item.ParseRequestedAt = ""
		item.ParseFinishedAt = ""
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

func (s *Service) UpdateStatus(ctx context.Context, userID int64, recipeID string, status string) (Recipe, error) {
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return Recipe{}, err
	}

	status = strings.TrimSpace(status)
	if !isAllowedStatus(status) {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	now := time.Now().Format(time.RFC3339)
	if err := s.repo.UpdateStatus(ctx, recipeID, current.KitchenID, status, userID, now); errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	} else if err != nil {
		return Recipe{}, err
	}

	current.Status = status
	current.UpdatedBy = userID
	current.UpdatedAt = now
	return current, nil
}

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
	if err := s.repo.QueueFlowchart(ctx, current.ID, current.KitchenID, userID, now); errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	} else if err != nil {
		return Recipe{}, err
	}

	current.FlowchartStatus = FlowchartStatusPending
	current.FlowchartError = ""
	current.FlowchartRequestedAt = now
	current.FlowchartFinishedAt = ""
	current.UpdatedBy = userID
	current.UpdatedAt = now
	return s.decorateRecipeRuntimeState(ctx, current), nil
}

func (s *Service) UpdatePinned(ctx context.Context, userID int64, recipeID string, pinned bool) (Recipe, error) {
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return Recipe{}, err
	}

	currentPinned := strings.TrimSpace(current.PinnedAt) != ""
	if currentPinned == pinned {
		return current, nil
	}

	now := time.Now().Format(time.RFC3339)
	if err := s.repo.UpdatePinned(ctx, recipeID, current.KitchenID, pinned, userID, now); errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, common.ErrNotFound
	} else if err != nil {
		return Recipe{}, err
	}

	if pinned {
		current.PinnedAt = now
	} else {
		current.PinnedAt = ""
	}
	current.UpdatedBy = userID
	return current, nil
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

func (s *Service) Delete(ctx context.Context, userID int64, recipeID string) error {
	current, err := s.GetByID(ctx, userID, recipeID)
	if err != nil {
		return err
	}

	if err := s.repo.SoftDelete(ctx, recipeID, current.KitchenID, userID, time.Now().Format(time.RFC3339)); errors.Is(err, sql.ErrNoRows) {
		return common.ErrNotFound
	} else if err != nil {
		return err
	}

	return nil
}

func normalizeRecipeInput(
	title string,
	ingredient string,
	summary string,
	link string,
	imageURL string,
	imageURLs []string,
	mealType string,
	status string,
	note string,
	parsedContent ParsedContent,
) (Recipe, error) {
	title = strings.TrimSpace(title)
	ingredient = strings.TrimSpace(ingredient)
	summary = strings.TrimSpace(summary)
	link = strings.TrimSpace(link)
	imageURL = strings.TrimSpace(imageURL)
	mealType = strings.TrimSpace(mealType)
	status = strings.TrimSpace(status)
	note = strings.TrimSpace(note)

	if title == "" {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "title is required", http.StatusBadRequest)
	}
	if len([]rune(title)) > 40 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "title must be 40 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(ingredient)) > 60 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "ingredient must be 60 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(summary)) > maxRecipeSummaryRunes {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "summary must be 24 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(link)) > 300 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "link must be 300 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(imageURL)) > 500 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "imageUrl must be 500 characters or fewer", http.StatusBadRequest)
	}
	if len([]rune(note)) > 300 {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "note must be 300 characters or fewer", http.StatusBadRequest)
	}

	if mealType == "" {
		mealType = "breakfast"
	}
	if status == "" {
		status = "wishlist"
	}

	if !isAllowedMealType(mealType) {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "invalid mealType", http.StatusBadRequest)
	}
	if !isAllowedStatus(status) {
		return Recipe{}, common.NewAppError(common.CodeBadRequest, "invalid status", http.StatusBadRequest)
	}

	normalizedImages, err := normalizeImageURLs(imageURL, imageURLs)
	if err != nil {
		return Recipe{}, err
	}
	normalizedContent := normalizeParsedContent(parsedContent, mealType, title, ingredient)

	return Recipe{
		Title:         title,
		Ingredient:    ingredient,
		Summary:       summary,
		Link:          link,
		ImageURL:      firstImageURL(normalizedImages),
		ImageURLs:     normalizedImages,
		MealType:      mealType,
		Status:        status,
		Note:          note,
		ParsedContent: normalizedContent,
	}, nil
}

func normalizeImageURLs(imageURL string, imageURLs []string) ([]string, error) {
	candidates := make([]string, 0, len(imageURLs)+1)
	candidates = append(candidates, imageURLs...)
	if strings.TrimSpace(imageURL) != "" {
		candidates = append(candidates, imageURL)
	}

	normalized := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, raw := range candidates {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if len([]rune(value)) > 500 {
			return nil, common.NewAppError(common.CodeBadRequest, "each imageUrl must be 500 characters or fewer", http.StatusBadRequest)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	if len(normalized) > maxRecipeImages {
		return nil, common.NewAppError(common.CodeBadRequest, "at most 9 images are allowed", http.StatusBadRequest)
	}

	return normalized, nil
}

func firstImageURL(imageURLs []string) string {
	if len(imageURLs) == 0 {
		return ""
	}
	return strings.TrimSpace(imageURLs[0])
}

var (
	secondaryIngredientPattern          = regexp.MustCompile(`(?i)(常用配菜|基础调味|调味|葱|姜|蒜|香叶|桂皮|八角|花椒|胡椒|盐|糖|冰糖|白糖|红糖|生抽|老抽|蚝油|料酒|鸡精|味精|醋|陈醋|米醋|香醋|豆瓣酱|辣椒|小米椒|淀粉|清水|热水|食用油|香油|芝麻油|花椒粉|辣椒粉|五香粉|十三香|孜然|芝麻|香菜|葱花)`)
	secondaryIngredientExceptionPattern = regexp.MustCompile(`(?i)^(洋葱|红葱头|葱头)`)
	ingredientSuffixPattern             = regexp.MustCompile(`\s*(?:\d+(?:\.\d+)?\s*(?:g|kg|克|千克|ml|毫升|l|升|勺|汤匙|茶匙|匙|杯|个|颗|根|把|片|块|斤|两|袋|盒|碗)|半个|半颗|半根|半头|适量|少许)$`)
)

func normalizeParsedContent(content ParsedContent, mealType, title, ingredient string) ParsedContent {
	mainIngredients := cleanLines(content.MainIngredients)
	secondaryIngredients := cleanLines(content.SecondaryIngredients)
	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 {
		mainIngredients, secondaryIngredients = splitIngredientLines(cleanLines(content.legacyIngredients))
	}

	steps := cleanParsedSteps(content.Steps)
	if len(steps) == 0 {
		steps = buildParsedSteps(cleanLines(content.legacySteps))
	}

	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 && len(steps) == 0 {
		return defaultParsedContent(mealType, title, ingredient)
	}

	fallback := defaultParsedContent(mealType, title, ingredient)
	if len(mainIngredients) == 0 && len(secondaryIngredients) == 0 {
		mainIngredients = append([]string{}, fallback.MainIngredients...)
		secondaryIngredients = append([]string{}, fallback.SecondaryIngredients...)
	}
	if len(steps) == 0 {
		steps = append([]ParsedStep{}, fallback.Steps...)
	}

	return ParsedContent{
		MainIngredients:      mainIngredients,
		SecondaryIngredients: secondaryIngredients,
		Steps:                steps,
	}
}

func defaultParsedContent(mealType, title, ingredient string) ParsedContent {
	mainIngredient := ingredient
	if mainIngredient == "" {
		mainIngredient = title
	}
	if mainIngredient == "" {
		mainIngredient = "主食材"
	}

	mealLabel := "早餐"
	if mealType == "main" {
		mealLabel = "正餐"
	}

	return ParsedContent{
		MainIngredients: []string{
			mainIngredient + " 1份",
		},
		SecondaryIngredients: []string{
			mealLabel + "常用配菜 适量",
			"基础调味 适量",
		},
		Steps: []ParsedStep{
			{Title: "整理做法", Detail: "先整理这道菜的核心做法。"},
			{Title: "调整口味", Detail: "按自己的口味调整成容易复刻的版本。"},
			{Title: "补充记录", Detail: "做完以后补充口感和火候记录。"},
		},
	}
}

func legacyFrontendFallbackParsedContent(mealType, title, ingredient string) ParsedContent {
	return normalizeParsedContent(ParsedContent{
		legacyIngredients: legacyFrontendFallbackIngredientLines(mealType, title, ingredient),
		legacySteps:       legacyFrontendFallbackStepLines(title),
	}, mealType, title, ingredient)
}

func cleanLines(lines []string) []string {
	items := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, exists := seen[line]; exists {
			continue
		}
		seen[line] = struct{}{}
		items = append(items, line)
	}
	return items
}

func shouldQueueAutoParse(link string, content ParsedContent, mealType, title, ingredient string) bool {
	if !linkparse.SupportsAutoParseURL(link) {
		return false
	}

	return !hasUserProvidedParsedContent(content, mealType, title, ingredient)
}

func canGenerateFlowchartForRecipe(item Recipe) bool {
	if strings.TrimSpace(item.Title) == "" {
		return false
	}
	if !hasUserProvidedParsedContent(item.ParsedContent, item.MealType, item.Title, item.Ingredient) {
		return false
	}
	return len(cleanParsedSteps(item.ParsedContent.Steps)) >= 3
}

func hasMeaningfulParsedContent(content ParsedContent) bool {
	return len(parsedContentIngredientLines(content)) > 0 || len(parsedContentStepLines(content)) > 0
}

func hasUserProvidedParsedContent(content ParsedContent, mealType, title, ingredient string) bool {
	if !hasMeaningfulParsedContent(content) {
		return false
	}

	requestedIngredients := parsedContentIngredientLines(content)
	requestedSteps := parsedContentStepLines(content)

	for _, fallback := range []struct {
		ingredients []string
		steps       []string
	}{
		{
			ingredients: parsedContentIngredientLines(defaultParsedContent(mealType, title, ingredient)),
			steps:       parsedContentStepLines(defaultParsedContent(mealType, title, ingredient)),
		},
		{
			ingredients: legacyFrontendFallbackIngredientLines(mealType, title, ingredient),
			steps:       legacyFrontendFallbackStepLines(title),
		},
	} {
		if stringSlicesEqual(requestedIngredients, fallback.ingredients) && stringSlicesEqual(requestedSteps, fallback.steps) {
			return false
		}
	}

	return true
}

func legacyFrontendFallbackIngredientLines(mealType, title, ingredient string) []string {
	return parsedContentIngredientLines(defaultParsedContent(mealType, title, ingredient))
}

func legacyFrontendFallbackStepLines(title string) []string {
	titleLabel := strings.TrimSpace(title)
	if titleLabel == "" {
		titleLabel = "这道菜"
	}

	return []string{
		"先从链接里抓出 " + titleLabel + " 的核心做法。",
		"按自己的口味整理成容易复刻的家常版本。",
		"做完以后回来补充口感、火候和踩坑点。",
	}
}

func parsedContentIngredientLines(content ParsedContent) []string {
	mainIngredients := cleanLines(content.MainIngredients)
	secondaryIngredients := cleanLines(content.SecondaryIngredients)
	if len(mainIngredients) > 0 || len(secondaryIngredients) > 0 {
		return append(append([]string{}, mainIngredients...), secondaryIngredients...)
	}
	return cleanLines(content.legacyIngredients)
}

func parsedContentStepLines(content ParsedContent) []string {
	steps := cleanParsedSteps(content.Steps)
	if len(steps) > 0 {
		items := make([]string, 0, len(steps))
		for _, step := range steps {
			items = append(items, step.Detail)
		}
		return items
	}
	return cleanLines(content.legacySteps)
}

func splitIngredientLines(lines []string) ([]string, []string) {
	cleaned := cleanLines(lines)
	if len(cleaned) == 0 {
		return nil, nil
	}

	mainIngredients := make([]string, 0, 4)
	secondaryIngredients := make([]string, 0, len(cleaned))

	for _, line := range cleaned {
		if isSecondaryIngredientLine(line) {
			secondaryIngredients = append(secondaryIngredients, line)
			continue
		}
		mainIngredients = append(mainIngredients, line)
	}

	if len(mainIngredients) == 0 {
		limit := 3
		if len(cleaned) < limit {
			limit = len(cleaned)
		}
		mainIngredients = append(mainIngredients, cleaned[:limit]...)
		secondaryIngredients = append([]string{}, cleaned[limit:]...)
	}

	return mainIngredients, secondaryIngredients
}

func isSecondaryIngredientLine(line string) bool {
	label := ingredientLabelFromLine(line)
	return secondaryIngredientPattern.MatchString(label) && !secondaryIngredientExceptionPattern.MatchString(label)
}

func ingredientLabelFromLine(line string) string {
	label := strings.TrimSpace(line)
	label = ingredientSuffixPattern.ReplaceAllString(label, "")
	return strings.TrimSpace(label)
}

func cleanParsedSteps(steps []ParsedStep) []ParsedStep {
	items := make([]ParsedStep, 0, len(steps))
	seen := make(map[string]struct{}, len(steps))
	for index, step := range steps {
		title := strings.TrimSpace(step.Title)
		detail := strings.TrimSpace(step.Detail)
		if detail == "" {
			detail = title
		}
		if detail == "" {
			continue
		}
		if title == "" {
			title = inferParsedStepTitle(detail, index)
		}
		key := title + "\x00" + detail
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, ParsedStep{
			Title:  title,
			Detail: detail,
		})
	}
	return compactParsedSteps(items)
}

func buildParsedSteps(lines []string) []ParsedStep {
	items := make([]ParsedStep, 0, len(lines))
	for index, line := range cleanLines(lines) {
		items = append(items, ParsedStep{
			Title:  inferParsedStepTitle(line, index),
			Detail: line,
		})
	}
	return compactParsedSteps(items)
}

func compactParsedSteps(steps []ParsedStep) []ParsedStep {
	if len(steps) <= maxParsedSteps {
		return append([]ParsedStep{}, steps...)
	}

	items := make([]ParsedStep, 0, maxParsedSteps)
	for index := 0; index < maxParsedSteps; index++ {
		start := index * len(steps) / maxParsedSteps
		end := (index + 1) * len(steps) / maxParsedSteps
		if start >= len(steps) {
			break
		}
		if end <= start {
			end = start + 1
		}
		if end > len(steps) {
			end = len(steps)
		}

		group := steps[start:end]
		title := strings.TrimSpace(group[0].Title)
		if title == "" {
			title = inferParsedStepTitle(group[0].Detail, index)
		}

		details := make([]string, 0, len(group))
		for _, step := range group {
			detail := strings.TrimSpace(step.Detail)
			if detail == "" {
				continue
			}
			details = append(details, detail)
		}
		if len(details) == 0 && title != "" {
			details = append(details, title)
		}

		items = append(items, ParsedStep{
			Title:  title,
			Detail: strings.Join(details, "；"),
		})
	}

	return items
}

func inferParsedStepTitle(detail string, index int) string {
	switch {
	case strings.Contains(detail, "焯水") || strings.Contains(detail, "汆水"):
		if strings.Contains(detail, "腥") || strings.Contains(detail, "浮沫") {
			return "焯水去腥"
		}
		return "焯水备用"
	case strings.Contains(detail, "腌"):
		return "腌制入味"
	case strings.Contains(detail, "糖色") || strings.Contains(detail, "冰糖"):
		return "炒糖上色"
	case strings.Contains(detail, "爆香") || strings.Contains(detail, "炒香"):
		return "炒香底料"
	case strings.Contains(detail, "切") || strings.Contains(detail, "改刀"):
		return "切配备料"
	case strings.Contains(detail, "收汁"):
		return "收汁出锅"
	case strings.Contains(detail, "炖") || strings.Contains(detail, "焖"):
		return "小火慢炖"
	case strings.Contains(detail, "蒸"):
		return "上锅蒸熟"
	case strings.Contains(detail, "炸"):
		return "炸至金黄"
	case strings.Contains(detail, "煎"):
		return "煎香上色"
	case strings.Contains(detail, "烤"):
		return "烤至上色"
	case strings.Contains(detail, "煮"):
		return "煮至入味"
	case strings.Contains(detail, "拌"):
		return "拌匀调味"
	case strings.Contains(detail, "炒") || strings.Contains(detail, "翻炒"):
		return "翻炒入味"
	case strings.Contains(detail, "出锅"):
		return "调味出锅"
	case index == 0:
		return "处理食材"
	default:
		return "继续烹饪"
	}
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}

func parsedStepSlicesEqual(left, right []ParsedStep) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index].Title != right[index].Title || left[index].Detail != right[index].Detail {
			return false
		}
	}

	return true
}

func parsedContentSlicesEqual(left, right ParsedContent) bool {
	return stringSlicesEqual(cleanLines(left.MainIngredients), cleanLines(right.MainIngredients)) &&
		stringSlicesEqual(cleanLines(left.SecondaryIngredients), cleanLines(right.SecondaryIngredients)) &&
		parsedStepSlicesEqual(cleanParsedSteps(left.Steps), cleanParsedSteps(right.Steps))
}

func isAllowedMealType(value string) bool {
	_, ok := allowedMealTypes[value]
	return ok
}

func isAllowedStatus(value string) bool {
	_, ok := allowedStatuses[value]
	return ok
}
