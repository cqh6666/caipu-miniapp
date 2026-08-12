package recipe

import "github.com/cqh6666/caipu-miniapp/backend/internal/recipecontent"

type ParsedStep = recipecontent.Step
type ParsedContent = recipecontent.Content

const (
	ParseStatusIdle       = ""
	ParseStatusPending    = "pending"
	ParseStatusProcessing = "processing"
	ParseStatusDone       = "done"
	ParseStatusFailed     = "failed"

	TitleSourceManual      = "manual"
	TitleSourcePlaceholder = "placeholder"
	TitleSourceParsed      = "parsed"

	FlowchartStatusIdle       = ""
	FlowchartStatusPending    = "pending"
	FlowchartStatusProcessing = "processing"
	FlowchartStatusDone       = "done"
	FlowchartStatusFailed     = "failed"
)

type Recipe struct {
	ID                       string            `json:"id"`
	KitchenID                int64             `json:"kitchenId"`
	Title                    string            `json:"title"`
	TitleSource              string            `json:"titleSource,omitempty"`
	Ingredient               string            `json:"ingredient"`
	Summary                  string            `json:"summary"`
	Link                     string            `json:"link"`
	ImageURL                 string            `json:"imageUrl"`
	ImageURLs                []string          `json:"imageUrls"`
	ImageMetas               []RecipeImageMeta `json:"-"`
	FlowchartImageURL        string            `json:"flowchartImageUrl"`
	FlowchartProvider        string            `json:"flowchartProvider"`
	FlowchartModel           string            `json:"flowchartModel"`
	FlowchartStatus          string            `json:"flowchartStatus"`
	FlowchartError           string            `json:"flowchartError"`
	FlowchartRequestedAt     string            `json:"flowchartRequestedAt"`
	FlowchartFinishedAt      string            `json:"flowchartFinishedAt"`
	FlowchartUpdatedAt       string            `json:"flowchartUpdatedAt"`
	FlowchartStale           bool              `json:"flowchartStale"`
	FlowchartQueueAhead      int               `json:"flowchartQueueAhead,omitempty"`
	FlowchartEstimatedWait   int               `json:"flowchartEstimatedWaitSeconds,omitempty"`
	MealType                 string            `json:"mealType"`
	Status                   string            `json:"status"`
	DoneAt                   string            `json:"-"`
	Note                     string            `json:"note"`
	ParsedContent            ParsedContent     `json:"parsedContent"`
	ParsedContentEdited      bool              `json:"parsedContentEdited"`
	ParseStatus              string            `json:"parseStatus"`
	ParseSource              string            `json:"parseSource"`
	ParseError               string            `json:"parseError"`
	ParseRequestedAt         string            `json:"parseRequestedAt"`
	ParseFinishedAt          string            `json:"parseFinishedAt"`
	ParseAttempts            int               `json:"parseAttempts,omitempty"`
	ParseNextAttemptAt       string            `json:"parseNextAttemptAt,omitempty"`
	ParseLastErrorType       string            `json:"parseLastErrorType,omitempty"`
	ParseProcessingStartedAt string            `json:"-"`
	ParseQueueAhead          int               `json:"parseQueueAhead,omitempty"`
	ParseEstimatedWait       int               `json:"parseEstimatedWaitSeconds,omitempty"`
	PinnedAt                 string            `json:"pinnedAt"`
	CreatedBy                int64             `json:"createdBy"`
	UpdatedBy                int64             `json:"updatedBy"`
	CreatedAt                string            `json:"createdAt"`
	UpdatedAt                string            `json:"updatedAt"`
	Version                  int64             `json:"version"`

	// 仅在 EnsureShareToken / GetByShareToken 路径下填充；scanRecipe 主流程不扫描，
	// 普通 List/Detail 接口返回不包含此字段（omitempty）
	ShareToken string `json:"shareToken,omitempty"`

	FlowchartSourceHash string `json:"-"`
}

type ListFilter struct {
	MealType                 string
	Status                   string
	Keyword                  string
	TitleKeyword             string
	IngredientKeyword        string
	TitleOrIngredientKeyword string
}

type CreateInput struct {
	Title               string
	TitleSource         string
	Ingredient          string
	Summary             string
	Link                string
	ImageURL            string
	ImageURLs           []string
	MealType            string
	Status              string
	Note                string
	ParsedContent       ParsedContent
	ParsedContentEdited *bool
}

type createRecipeRequest struct {
	Title               string        `json:"title"`
	TitleSource         string        `json:"titleSource"`
	Ingredient          string        `json:"ingredient"`
	Summary             string        `json:"summary"`
	Link                string        `json:"link"`
	ImageURL            string        `json:"imageUrl"`
	ImageURLs           []string      `json:"imageUrls"`
	MealType            string        `json:"mealType"`
	Status              string        `json:"status"`
	Note                string        `json:"note"`
	ParsedContent       ParsedContent `json:"parsedContent"`
	ParsedContentEdited *bool         `json:"parsedContentEdited"`
}

type updateRecipeRequest struct {
	Version             *int64        `json:"version"`
	Title               string        `json:"title"`
	TitleSource         string        `json:"titleSource"`
	Ingredient          string        `json:"ingredient"`
	Summary             string        `json:"summary"`
	Link                string        `json:"link"`
	ImageURL            string        `json:"imageUrl"`
	ImageURLs           []string      `json:"imageUrls"`
	MealType            string        `json:"mealType"`
	Status              string        `json:"status"`
	Note                string        `json:"note"`
	ParsedContent       ParsedContent `json:"parsedContent"`
	ParsedContentEdited *bool         `json:"parsedContentEdited"`
}

type updateStatusRequest struct {
	Status  string `json:"status"`
	Version *int64 `json:"version"`
}

type updatePinnedRequest struct {
	Pinned  bool   `json:"pinned"`
	Version *int64 `json:"version"`
}
