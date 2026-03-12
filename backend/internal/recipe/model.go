package recipe

type ParsedContent struct {
	Ingredients []string `json:"ingredients"`
	Steps       []string `json:"steps"`
}

type Recipe struct {
	ID            string        `json:"id"`
	KitchenID     int64         `json:"kitchenId"`
	Title         string        `json:"title"`
	Ingredient    string        `json:"ingredient"`
	Link          string        `json:"link"`
	ImageURL      string        `json:"imageUrl"`
	MealType      string        `json:"mealType"`
	Status        string        `json:"status"`
	Note          string        `json:"note"`
	ParsedContent ParsedContent `json:"parsedContent"`
	CreatedBy     int64         `json:"createdBy"`
	UpdatedBy     int64         `json:"updatedBy"`
	CreatedAt     string        `json:"createdAt"`
	UpdatedAt     string        `json:"updatedAt"`
}

type ListFilter struct {
	MealType string
	Status   string
	Keyword  string
}

type createRecipeRequest struct {
	Title         string        `json:"title"`
	Ingredient    string        `json:"ingredient"`
	Link          string        `json:"link"`
	ImageURL      string        `json:"imageUrl"`
	MealType      string        `json:"mealType"`
	Status        string        `json:"status"`
	Note          string        `json:"note"`
	ParsedContent ParsedContent `json:"parsedContent"`
}

type updateRecipeRequest struct {
	Title         string        `json:"title"`
	Ingredient    string        `json:"ingredient"`
	Link          string        `json:"link"`
	ImageURL      string        `json:"imageUrl"`
	MealType      string        `json:"mealType"`
	Status        string        `json:"status"`
	Note          string        `json:"note"`
	ParsedContent ParsedContent `json:"parsedContent"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}
