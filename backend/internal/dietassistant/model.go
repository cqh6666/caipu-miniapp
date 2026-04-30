package dietassistant

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StoredMessage struct {
	ID        int64  `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type ChatStreamRequest struct {
	Messages  []ChatMessage `json:"messages"`
	KitchenID int64         `json:"kitchenId,omitempty"`
}

type StreamEvent struct {
	Type    string `json:"type"`
	Delta   string `json:"delta,omitempty"`
	Message string `json:"message,omitempty"`
}

type ChatContext struct {
	UserID    int64
	KitchenID int64
}

type RecipeCountInput struct {
	UserID    int64
	KitchenID int64
	MealType  string
	Status    string
}

type RecipeFromURLInput struct {
	UserID    int64
	KitchenID int64
	URL       string
	MealType  string
	Status    string
}

type RecipeSearchInput struct {
	UserID       int64
	KitchenID    int64
	TitleKeyword string
	MealType     string
	Status       string
	Limit        int
}

type RecipeFromURLResult struct {
	Recipe               RecipeToolItem `json:"recipe"`
	Source               string         `json:"source,omitempty"`
	SourceDetail         string         `json:"sourceDetail,omitempty"`
	SummaryMode          string         `json:"summaryMode,omitempty"`
	MainIngredients      []string       `json:"mainIngredients,omitempty"`
	SecondaryIngredients []string       `json:"secondaryIngredients,omitempty"`
	StepsCount           int            `json:"stepsCount,omitempty"`
	Warnings             []string       `json:"warnings,omitempty"`
}

type RecipeToolItem struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	MealType   string `json:"mealType"`
	Status     string `json:"status"`
	Ingredient string `json:"ingredient,omitempty"`
	Summary    string `json:"summary,omitempty"`
	Note       string `json:"note,omitempty"`
	Link       string `json:"link,omitempty"`
}
