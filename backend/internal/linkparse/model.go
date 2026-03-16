package linkparse

type ParsedContent struct {
	Ingredients []string `json:"ingredients"`
	Steps       []string `json:"steps"`
}

type RecipeDraft struct {
	Title         string        `json:"title"`
	Ingredient    string        `json:"ingredient"`
	Link          string        `json:"link"`
	ImageURL      string        `json:"imageUrl"`
	ImageURLs     []string      `json:"imageUrls"`
	Note          string        `json:"note"`
	ParsedContent ParsedContent `json:"parsedContent"`
}

type RecipeParseOutcome struct {
	Source      string      `json:"source"`
	SummaryMode string      `json:"summaryMode"`
	RecipeDraft RecipeDraft `json:"recipeDraft"`
}

type LinkPreviewResult struct {
	Platform     string   `json:"platform"`
	Link         string   `json:"link"`
	CanonicalURL string   `json:"canonicalUrl"`
	Title        string   `json:"title"`
	CoverURL     string   `json:"coverUrl"`
	ImageURLs    []string `json:"imageUrls"`
	ProviderUsed string   `json:"providerUsed,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

type BilibiliParseResult struct {
	Source            string      `json:"source"`
	Link              string      `json:"link"`
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	Part              string      `json:"part"`
	Author            string      `json:"author"`
	CoverURL          string      `json:"coverUrl"`
	BVID              string      `json:"bvid"`
	AID               int64       `json:"aid"`
	CID               int64       `json:"cid"`
	Page              int         `json:"page"`
	SubtitleAvailable bool        `json:"subtitleAvailable"`
	SubtitleLanguage  string      `json:"subtitleLanguage"`
	SubtitleSegments  int         `json:"subtitleSegments"`
	SubtitleText      string      `json:"subtitleText"`
	SummaryMode       string      `json:"summaryMode"`
	RecipeDraft       RecipeDraft `json:"recipeDraft"`
	Warnings          []string    `json:"warnings"`
}

type XiaohongshuParseResult struct {
	Source            string      `json:"source"`
	Link              string      `json:"link"`
	CanonicalURL      string      `json:"canonicalUrl"`
	ProviderRequested string      `json:"providerRequested"`
	ProviderUsed      string      `json:"providerUsed"`
	Title             string      `json:"title"`
	Content           string      `json:"content"`
	CoverURL          string      `json:"coverUrl"`
	Images            []string    `json:"images"`
	Videos            []string    `json:"videos"`
	Tags              []string    `json:"tags"`
	Author            string      `json:"author"`
	NoteType          string      `json:"noteType"`
	NoteID            string      `json:"noteId"`
	XSECToken         string      `json:"xsecToken"`
	SummaryMode       string      `json:"summaryMode"`
	RecipeDraft       RecipeDraft `json:"recipeDraft"`
	Warnings          []string    `json:"warnings"`
}

type parseLinkRequest struct {
	URL string `json:"url"`
}
