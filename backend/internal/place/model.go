package place

type Place struct {
	ID               string   `json:"id"`
	KitchenID        int64    `json:"kitchenId"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	Address          string   `json:"address"`
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	Price            string   `json:"price"`
	Source           string   `json:"source"`
	SourceURL        string   `json:"sourceUrl"`
	ImageURLs        []string `json:"imageUrls"`
	Status           string   `json:"status"`
	Tags             []string `json:"tags"`
	Note             string   `json:"note"`
	VisitedAt        string   `json:"visitedAt"`
	RevisitRating    int      `json:"revisitRating"`
	RecommendedItems []string `json:"recommendedItems"`
	Phone            string   `json:"phone"`
	ExternalProvider string   `json:"externalProvider"`
	ExternalPOIID    string   `json:"externalPoiId"`
	Rating           string   `json:"rating"`
	DiningTips       string   `json:"diningTips"`
	Scenes           []string `json:"scenes"`
	BestTime         string   `json:"bestTime"`
	Duration         string   `json:"duration"`
	CompanionTags    []string `json:"companionTags"`
	ParkingNote      string   `json:"parkingNote"`
	CreatedBy        int64    `json:"createdBy"`
	UpdatedBy        int64    `json:"updatedBy"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
}

type ListFilter struct {
	Status  string
	Keyword string
}

type placeRequest struct {
	Name             *string  `json:"name"`
	Type             *string  `json:"type"`
	Address          *string  `json:"address"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	Price            *string  `json:"price"`
	Source           *string  `json:"source"`
	SourceURL        *string  `json:"sourceUrl"`
	ImageURLs        []string `json:"imageUrls"`
	Status           *string  `json:"status"`
	Tags             []string `json:"tags"`
	Note             *string  `json:"note"`
	VisitedAt        *string  `json:"visitedAt"`
	RevisitRating    *int     `json:"revisitRating"`
	RecommendedItems []string `json:"recommendedItems"`
	Phone            *string  `json:"phone"`
	ExternalProvider *string  `json:"externalProvider"`
	ExternalPOIID    *string  `json:"externalPoiId"`
	Rating           *string  `json:"rating"`
	DiningTips       *string  `json:"diningTips"`
	Scenes           []string `json:"scenes"`
	BestTime         *string  `json:"bestTime"`
	Duration         *string  `json:"duration"`
	CompanionTags    []string `json:"companionTags"`
	ParkingNote      *string  `json:"parkingNote"`
}

type updateStatusRequest struct {
	Status           string   `json:"status"`
	VisitedAt        *string  `json:"visitedAt"`
	RevisitRating    *int     `json:"revisitRating"`
	RecommendedItems []string `json:"recommendedItems"`
}
