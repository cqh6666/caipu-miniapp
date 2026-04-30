package dietassistant

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
