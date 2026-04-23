package recipe

import (
	"encoding/json"
	"strings"
)

type ParsedStep struct {
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type ParsedContent struct {
	MainIngredients      []string     `json:"mainIngredients,omitempty"`
	SecondaryIngredients []string     `json:"secondaryIngredients,omitempty"`
	Steps                []ParsedStep `json:"steps,omitempty"`

	legacyIngredients []string
	legacySteps       []string
}

func (c ParsedContent) MarshalJSON() ([]byte, error) {
	type payload struct {
		MainIngredients      []string     `json:"mainIngredients,omitempty"`
		SecondaryIngredients []string     `json:"secondaryIngredients,omitempty"`
		Steps                []ParsedStep `json:"steps,omitempty"`
	}

	return json.Marshal(payload{
		MainIngredients:      c.MainIngredients,
		SecondaryIngredients: c.SecondaryIngredients,
		Steps:                c.Steps,
	})
}

func (c *ParsedContent) UnmarshalJSON(data []byte) error {
	type payload struct {
		MainIngredients      []string        `json:"mainIngredients"`
		SecondaryIngredients []string        `json:"secondaryIngredients"`
		Ingredients          []string        `json:"ingredients"`
		Steps                json.RawMessage `json:"steps"`
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*c = ParsedContent{}
		return nil
	}

	var raw payload
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	structuredSteps, legacySteps, err := parseParsedContentSteps(raw.Steps)
	if err != nil {
		return err
	}

	*c = ParsedContent{
		MainIngredients:      raw.MainIngredients,
		SecondaryIngredients: raw.SecondaryIngredients,
		Steps:                structuredSteps,
		legacyIngredients:    raw.Ingredients,
		legacySteps:          legacySteps,
	}
	return nil
}

func parseParsedContentSteps(data json.RawMessage) ([]ParsedStep, []string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil, nil, nil
	}

	var structured []ParsedStep
	if err := json.Unmarshal(data, &structured); err == nil {
		return structured, nil, nil
	}

	var legacy []string
	if err := json.Unmarshal(data, &legacy); err == nil {
		return nil, legacy, nil
	}

	var structuredErr error
	if err := json.Unmarshal(data, &structured); err != nil {
		structuredErr = err
	}
	return nil, nil, structuredErr
}

const (
	ParseStatusIdle       = ""
	ParseStatusPending    = "pending"
	ParseStatusProcessing = "processing"
	ParseStatusDone       = "done"
	ParseStatusFailed     = "failed"

	FlowchartStatusIdle       = ""
	FlowchartStatusPending    = "pending"
	FlowchartStatusProcessing = "processing"
	FlowchartStatusDone       = "done"
	FlowchartStatusFailed     = "failed"
)

type Recipe struct {
	ID                     string            `json:"id"`
	KitchenID              int64             `json:"kitchenId"`
	Title                  string            `json:"title"`
	Ingredient             string            `json:"ingredient"`
	Summary                string            `json:"summary"`
	Link                   string            `json:"link"`
	ImageURL               string            `json:"imageUrl"`
	ImageURLs              []string          `json:"imageUrls"`
	ImageMetas             []RecipeImageMeta `json:"-"`
	FlowchartImageURL      string            `json:"flowchartImageUrl"`
	FlowchartProvider      string            `json:"flowchartProvider"`
	FlowchartModel         string            `json:"flowchartModel"`
	FlowchartStatus        string            `json:"flowchartStatus"`
	FlowchartError         string            `json:"flowchartError"`
	FlowchartRequestedAt   string            `json:"flowchartRequestedAt"`
	FlowchartFinishedAt    string            `json:"flowchartFinishedAt"`
	FlowchartUpdatedAt     string            `json:"flowchartUpdatedAt"`
	FlowchartStale         bool              `json:"flowchartStale"`
	FlowchartQueueAhead    int               `json:"flowchartQueueAhead,omitempty"`
	FlowchartEstimatedWait int               `json:"flowchartEstimatedWaitSeconds,omitempty"`
	MealType               string            `json:"mealType"`
	Status                 string            `json:"status"`
	Note                   string            `json:"note"`
	ParsedContent          ParsedContent     `json:"parsedContent"`
	ParsedContentEdited    bool              `json:"parsedContentEdited"`
	ParseStatus            string            `json:"parseStatus"`
	ParseSource            string            `json:"parseSource"`
	ParseError             string            `json:"parseError"`
	ParseRequestedAt       string            `json:"parseRequestedAt"`
	ParseFinishedAt        string            `json:"parseFinishedAt"`
	ParseQueueAhead        int               `json:"parseQueueAhead,omitempty"`
	ParseEstimatedWait     int               `json:"parseEstimatedWaitSeconds,omitempty"`
	PinnedAt               string            `json:"pinnedAt"`
	CreatedBy              int64             `json:"createdBy"`
	UpdatedBy              int64             `json:"updatedBy"`
	CreatedAt              string            `json:"createdAt"`
	UpdatedAt              string            `json:"updatedAt"`

	// 仅在 EnsureShareToken / GetByShareToken 路径下填充；scanRecipe 主流程不扫描，
	// 普通 List/Detail 接口返回不包含此字段（omitempty）
	ShareToken string `json:"shareToken,omitempty"`

	FlowchartSourceHash string `json:"-"`
}

type ListFilter struct {
	MealType string
	Status   string
	Keyword  string
}

type createRecipeRequest struct {
	Title               string        `json:"title"`
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
	Title               string        `json:"title"`
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
	Status string `json:"status"`
}

type updatePinnedRequest struct {
	Pinned bool `json:"pinned"`
}
