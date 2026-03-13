package linkparse

type ParsedContent struct {
	Ingredients []string `json:"ingredients"`
	Steps       []string `json:"steps"`
}

type RecipeDraft struct {
	Title         string        `json:"title"`
	Ingredient    string        `json:"ingredient"`
	Link          string        `json:"link"`
	Note          string        `json:"note"`
	ParsedContent ParsedContent `json:"parsedContent"`
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

type parseBilibiliRequest struct {
	URL string `json:"url"`
}
