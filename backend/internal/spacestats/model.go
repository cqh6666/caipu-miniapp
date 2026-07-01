package spacestats

type Stats struct {
	UpdatedAt   string        `json:"updatedAt"`
	Source      string        `json:"source"`
	Window      string        `json:"window"`
	WindowStart string        `json:"windowStart,omitempty"`
	Overview    OverviewStats `json:"overview"`
	Recipes     RecipeStats   `json:"recipes"`
	Places      PlaceStats    `json:"places"`
	MealPlans   MealPlanStats `json:"mealPlans"`
	Members     MemberStats   `json:"members"`
	Trends      TrendStats    `json:"trends"`
	Actions     []Action      `json:"actions"`
}

type OverviewStats struct {
	RecipeTotal              int               `json:"recipeTotal"`
	PlaceTotal               int               `json:"placeTotal"`
	SubmittedMealPlanDays    int               `json:"submittedMealPlanDays"`
	MemberTotal              int               `json:"memberTotal"`
	WishlistRecipeTotal      int               `json:"wishlistRecipeTotal"`
	WantPlaceTotal           int               `json:"wantPlaceTotal"`
	TopRevisitPlaces         []TopRevisitPlace `json:"topRevisitPlaces"`
	RecentCreatedRecipes     int               `json:"recentCreatedRecipes"`
	RecentCreatedPlaces      int               `json:"recentCreatedPlaces"`
	RecentVisitedPlaces      int               `json:"recentVisitedPlaces"`
	RecentSubmittedMealPlans int               `json:"recentSubmittedMealPlans"`
}

type RecipeStats struct {
	Total               int            `json:"total"`
	ByMealType          map[string]int `json:"byMealType"`
	ByStatus            map[string]int `json:"byStatus"`
	ImageCoveredTotal   int            `json:"imageCoveredTotal"`
	ImageCoverage       float64        `json:"imageCoverage"`
	ParsedTotal         int            `json:"parsedTotal"`
	FlowchartDoneTotal  int            `json:"flowchartDoneTotal"`
	FlowchartQueueTotal int            `json:"flowchartQueueTotal"`
	FlowchartTodoTotal  int            `json:"flowchartTodoTotal"`
	RecentCreatedTotal  int            `json:"recentCreatedTotal"`
	DoneTrendTotal      int            `json:"doneTrendTotal"`
}

type PlaceStats struct {
	Total                     int            `json:"total"`
	ByStatus                  map[string]int `json:"byStatus"`
	LocatedTotal              int            `json:"locatedTotal"`
	LocationCoverage          float64        `json:"locationCoverage"`
	ExperienceCompletedTotal  int            `json:"experienceCompletedTotal"`
	HighlyRecommendedTotal    int            `json:"highlyRecommendedTotal"`
	LowRatingTotal            int            `json:"lowRatingTotal"`
	AverageRevisitRating      float64        `json:"averageRevisitRating"`
	POIMatchedTotal           int            `json:"poiMatchedTotal"`
	RecentCreatedTotal        int            `json:"recentCreatedTotal"`
	RecentVisitedTotal        int            `json:"recentVisitedTotal"`
	PricedPlaceTotal          int            `json:"pricedPlaceTotal"`
	TotalPriceAmountCents     int64          `json:"totalPriceAmountCents"`
	AveragePriceAmountCents   int64          `json:"averagePriceAmountCents"`
	PriceCurrency             string         `json:"priceCurrency"`
	TopRecommendedItems       []CountedLabel `json:"topRecommendedItems"`
	TopScenes                 []CountedLabel `json:"topScenes"`
	RevisitRatingDistribution map[string]int `json:"revisitRatingDistribution"`
}

type MealPlanStats struct {
	DraftDays           int            `json:"draftDays"`
	SubmittedDays       int            `json:"submittedDays"`
	RecentSubmittedDays int            `json:"recentSubmittedDays"`
	AverageDishCount    float64        `json:"averageDishCount"`
	NextPlan            *MealPlanBrief `json:"nextPlan"`
	LatestPlan          *MealPlanBrief `json:"latestPlan"`
	ItemsByMealType     map[string]int `json:"itemsByMealType"`
}

type MealPlanBrief struct {
	ID        int64  `json:"id"`
	PlanDate  string `json:"planDate"`
	Status    string `json:"status"`
	ItemCount int    `json:"itemCount"`
}

type MemberStats struct {
	Total        int                  `json:"total"`
	Contributors []MemberContribution `json:"contributors"`
}

type MemberContribution struct {
	UserID                 int64  `json:"userId"`
	Nickname               string `json:"nickname"`
	AvatarURL              string `json:"avatarUrl"`
	Role                   string `json:"role"`
	JoinedAt               string `json:"joinedAt"`
	RecipeCreatedTotal     int    `json:"recipeCreatedTotal"`
	PlaceCreatedTotal      int    `json:"placeCreatedTotal"`
	MealPlanSubmittedTotal int    `json:"mealPlanSubmittedTotal"`
	Total                  int    `json:"total"`
}

type TrendStats struct {
	RecipeCreated     []DailyPoint `json:"recipeCreated"`
	RecipeDone        []DailyPoint `json:"recipeDone"`
	PlaceCreated      []DailyPoint `json:"placeCreated"`
	PlaceVisited      []DailyPoint `json:"placeVisited"`
	MealPlanSubmitted []DailyPoint `json:"mealPlanSubmitted"`
}

type DailyPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type TopRevisitPlace struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	RevisitRating    int      `json:"revisitRating"`
	RecommendedItems []string `json:"recommendedItems"`
	ImageURL         string   `json:"imageUrl"`
	VisitedAt        string   `json:"visitedAt"`
}

type CountedLabel struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type Action struct {
	Type   string `json:"type"`
	Label  string `json:"label"`
	Count  int    `json:"count,omitempty"`
	Target string `json:"target,omitempty"`
}
