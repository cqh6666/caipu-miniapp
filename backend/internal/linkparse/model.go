package linkparse

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

type RecipeDraft struct {
	Title         string        `json:"title"`
	Ingredient    string        `json:"ingredient"`
	Summary       string        `json:"summary"`
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
	Transcript        string      `json:"transcript"`
	TranscriptStatus  string      `json:"transcriptStatus"`
	TranscriptError   string      `json:"transcriptError"`
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
