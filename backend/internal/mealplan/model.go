package mealplan

const (
	StatusDraft     = "draft"
	StatusSubmitted = "submitted"
)

type Item struct {
	RecipeID         string `json:"recipeId"`
	Quantity         int    `json:"quantity"`
	MealTypeSnapshot string `json:"mealTypeSnapshot"`
	TitleSnapshot    string `json:"titleSnapshot"`
	ImageSnapshot    string `json:"imageSnapshot"`
	Sort             int    `json:"sort,omitempty"`
}

type Plan struct {
	ID          int64  `json:"id"`
	KitchenID   int64  `json:"kitchenId,omitempty"`
	PlanDate    string `json:"planDate"`
	Status      string `json:"status"`
	Note        string `json:"note"`
	Items       []Item `json:"items"`
	CreatedBy   int64  `json:"createdBy,omitempty"`
	UpdatedBy   int64  `json:"updatedBy,omitempty"`
	SubmittedBy int64  `json:"submittedBy,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	SubmittedAt string `json:"submittedAt,omitempty"`
}

type Store struct {
	Drafts    map[string]Plan `json:"drafts"`
	Submitted []Plan          `json:"submitted"`
}

type savePlanRequest struct {
	Items []Item `json:"items"`
	Note  string `json:"note"`
}
