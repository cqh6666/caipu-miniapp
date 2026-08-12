package recipecontent

import (
	"encoding/json"
	"strings"
)

type Step struct {
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type Content struct {
	MainIngredients      []string `json:"mainIngredients,omitempty"`
	SecondaryIngredients []string `json:"secondaryIngredients,omitempty"`
	Steps                []Step   `json:"steps,omitempty"`

	legacyIngredients []string
	legacySteps       []string
}

func (c Content) MarshalJSON() ([]byte, error) {
	type payload struct {
		MainIngredients      []string `json:"mainIngredients,omitempty"`
		SecondaryIngredients []string `json:"secondaryIngredients,omitempty"`
		Steps                []Step   `json:"steps,omitempty"`
	}

	return json.Marshal(payload{
		MainIngredients:      c.MainIngredients,
		SecondaryIngredients: c.SecondaryIngredients,
		Steps:                c.Steps,
	})
}

func (c *Content) UnmarshalJSON(data []byte) error {
	type payload struct {
		MainIngredients      []string        `json:"mainIngredients"`
		SecondaryIngredients []string        `json:"secondaryIngredients"`
		Ingredients          []string        `json:"ingredients"`
		Steps                json.RawMessage `json:"steps"`
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*c = Content{}
		return nil
	}

	var raw payload
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	content, err := DecodeFields(raw.MainIngredients, raw.SecondaryIngredients, raw.Ingredients, raw.Steps)
	if err != nil {
		return err
	}
	*c = content
	return nil
}

func DecodeFields(mainIngredients, secondaryIngredients, legacyIngredients []string, stepsJSON json.RawMessage) (Content, error) {
	steps, legacySteps, err := decodeSteps(stepsJSON)
	if err != nil {
		return Content{}, err
	}

	return Content{
		MainIngredients:      mainIngredients,
		SecondaryIngredients: secondaryIngredients,
		Steps:                steps,
		legacyIngredients:    legacyIngredients,
		legacySteps:          legacySteps,
	}, nil
}

func FromLegacy(ingredients, steps []string) Content {
	return Content{
		legacyIngredients: append([]string{}, ingredients...),
		legacySteps:       append([]string{}, steps...),
	}
}

func decodeSteps(data json.RawMessage) ([]Step, []string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil, nil, nil
	}

	var structured []Step
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
