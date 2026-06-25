package addpreview

import "github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"

const (
	StatusPlaceCandidates = "place_candidates"
	StatusRecipeResult    = "recipe_result"
	StatusPartial         = "partial"
	StatusFailed          = "failed"

	ContentTypePlace  = "place"
	ContentTypeRecipe = "recipe"

	SourceMeituan  = "meituan"
	SourceDianping = "dianping"
	SourceOther    = "other"
)

type PreviewRequest struct {
	Text      string  `json:"text"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Limit     int     `json:"limit"`
}

type PreviewResponse struct {
	PreviewID   string           `json:"previewId"`
	Status      string           `json:"status"`
	ContentType string           `json:"contentType,omitempty"`
	Source      string           `json:"source,omitempty"`
	Extracted   ExtractedPlace   `json:"extracted,omitempty"`
	Draft       PlaceDraft       `json:"draft,omitempty"`
	Candidates  []PlaceCandidate `json:"candidates,omitempty"`
	RecipeDraft RecipeDraft      `json:"recipeDraft,omitempty"`
	Warnings    []Warning        `json:"warnings,omitempty"`
	Message     string           `json:"message,omitempty"`
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ExtractedPlace struct {
	Name         string `json:"name,omitempty"`
	Address      string `json:"address,omitempty"`
	Phone        string `json:"phone,omitempty"`
	SourceURL    string `json:"sourceUrl,omitempty"`
	POIID        string `json:"poiId,omitempty"`
	POIIDEncrypt string `json:"poiIdEncrypt,omitempty"`
}

type PlaceDraft struct {
	Name             string   `json:"name,omitempty"`
	Type             string   `json:"type,omitempty"`
	Address          string   `json:"address,omitempty"`
	Latitude         float64  `json:"latitude,omitempty"`
	Longitude        float64  `json:"longitude,omitempty"`
	Phone            string   `json:"phone,omitempty"`
	Price            string   `json:"price,omitempty"`
	Source           string   `json:"source,omitempty"`
	SourceURL        string   `json:"sourceUrl,omitempty"`
	Images           []string `json:"images,omitempty"`
	ImageURLs        []string `json:"imageUrls,omitempty"`
	Status           string   `json:"status,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Note             string   `json:"note,omitempty"`
	ExternalProvider string   `json:"externalProvider,omitempty"`
	ExternalPOIID    string   `json:"externalPoiId,omitempty"`
	Rating           string   `json:"rating,omitempty"`
}

type PlaceCandidate struct {
	CandidateID   string     `json:"candidateId"`
	Provider      string     `json:"provider"`
	ProviderPOIID string     `json:"providerPoiId,omitempty"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	Address       string     `json:"address,omitempty"`
	Latitude      float64    `json:"latitude,omitempty"`
	Longitude     float64    `json:"longitude,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	Price         string     `json:"price,omitempty"`
	Rating        string     `json:"rating,omitempty"`
	ImageURLs     []string   `json:"imageUrls,omitempty"`
	MatchScore    int        `json:"matchScore"`
	MatchReasons  []string   `json:"matchReasons,omitempty"`
	PlaceDraft    PlaceDraft `json:"placeDraft"`
}

type RecipeDraft struct {
	Title         string                  `json:"title,omitempty"`
	Ingredient    string                  `json:"ingredient,omitempty"`
	Summary       string                  `json:"summary,omitempty"`
	Link          string                  `json:"link,omitempty"`
	ImageURL      string                  `json:"imageUrl,omitempty"`
	Images        []string                `json:"images,omitempty"`
	ImageURLs     []string                `json:"imageUrls,omitempty"`
	Note          string                  `json:"note,omitempty"`
	ParsedContent linkparse.ParsedContent `json:"parsedContent,omitempty"`
}

type poiSearchInput struct {
	Keyword string
	City    string
	Limit   int
}

type poiItem struct {
	ID           string
	Name         string
	Type         string
	TypeCode     string
	Address      string
	Location     string
	Tel          string
	Rating       string
	Cost         string
	BusinessArea string
	AdName       string
	PName        string
	CityName     string
	Photos       []string
}
