package recipecontent

import (
	"encoding/json"
	"fmt"
	"strings"
)

func EncodeStored(content Content) (string, string, error) {
	normalized := Normalize(content)
	ingredients, err := json.Marshal(struct {
		MainIngredients      []string `json:"mainIngredients,omitempty"`
		SecondaryIngredients []string `json:"secondaryIngredients,omitempty"`
	}{
		MainIngredients:      normalized.MainIngredients,
		SecondaryIngredients: normalized.SecondaryIngredients,
	})
	if err != nil {
		return "", "", fmt.Errorf("marshal ingredients: %w", err)
	}

	steps, err := json.Marshal(normalized.Steps)
	if err != nil {
		return "", "", fmt.Errorf("marshal steps: %w", err)
	}

	return string(ingredients), string(steps), nil
}

func DecodeStored(ingredientsJSON, stepsJSON string) (Content, error) {
	content := Content{}
	if strings.TrimSpace(ingredientsJSON) != "" {
		var grouped struct {
			MainIngredients      []string `json:"mainIngredients"`
			SecondaryIngredients []string `json:"secondaryIngredients"`
		}
		if err := json.Unmarshal([]byte(ingredientsJSON), &grouped); err == nil {
			content.MainIngredients = grouped.MainIngredients
			content.SecondaryIngredients = grouped.SecondaryIngredients
		} else if err := json.Unmarshal([]byte(ingredientsJSON), &content.legacyIngredients); err != nil {
			return Content{}, fmt.Errorf("unmarshal ingredients: %w", err)
		}
	}

	if strings.TrimSpace(stepsJSON) != "" {
		steps, legacySteps, err := decodeSteps(json.RawMessage(stepsJSON))
		if err != nil {
			return Content{}, fmt.Errorf("unmarshal steps: %w", err)
		}
		content.Steps = steps
		content.legacySteps = legacySteps
	}

	return content, nil
}
