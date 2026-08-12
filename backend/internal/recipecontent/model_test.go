package recipecontent

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestContentLegacyJSONNormalizesToStructuredContract(t *testing.T) {
	t.Parallel()

	var content Content
	if err := json.Unmarshal([]byte(`{
		"ingredients":["牛腩 500克","番茄 3个","常用调味料 适量"],
		"steps":["先把牛腩焯水去腥。","然后加入番茄炖煮。"]
	}`), &content); err != nil {
		t.Fatal(err)
	}

	normalized := Normalize(content)
	if want := []string{"牛腩 500克", "番茄 3个"}; !reflect.DeepEqual(normalized.MainIngredients, want) {
		t.Fatalf("MainIngredients = %#v, want %#v", normalized.MainIngredients, want)
	}
	if want := []string{"常用调味料 适量"}; !reflect.DeepEqual(normalized.SecondaryIngredients, want) {
		t.Fatalf("SecondaryIngredients = %#v, want %#v", normalized.SecondaryIngredients, want)
	}
	if got, want := normalized.Steps[0], (Step{Title: "焯水去腥", Detail: "先把牛腩焯水去腥"}); got != want {
		t.Fatalf("Steps[0] = %#v, want %#v", got, want)
	}

	encoded, err := json.Marshal(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"ingredients"`) {
		t.Fatalf("modern JSON leaked legacy field: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"mainIngredients"`) || !strings.Contains(string(encoded), `"secondaryIngredients"`) {
		t.Fatalf("modern JSON missing structured fields: %s", encoded)
	}
}

func TestStoredCodecReadsLegacyArraysAndWritesStructuredFormat(t *testing.T) {
	t.Parallel()

	content, err := DecodeStored(
		`["牛腩 500克","番茄 3个","基础调味 适量"]`,
		`["牛腩焯水去腥。","加入番茄小火炖煮。"]`,
	)
	if err != nil {
		t.Fatal(err)
	}
	ingredientsJSON, stepsJSON, err := EncodeStored(content)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ingredientsJSON, `{"mainIngredients":["牛腩 500克","番茄 3个"],"secondaryIngredients":["基础调味 适量"]}`; got != want {
		t.Fatalf("ingredientsJSON = %s, want %s", got, want)
	}
	if !strings.Contains(stepsJSON, `"title":"焯水去腥"`) || !strings.Contains(stepsJSON, `"detail":"牛腩焯水去腥"`) {
		t.Fatalf("stepsJSON missing structured step: %s", stepsJSON)
	}
}

func TestNormalizeUsesOneRuleSetForStructuredAndLegacyInputs(t *testing.T) {
	t.Parallel()

	structured := Content{
		MainIngredients:      []string{" 牛腩 500克 ", "番茄 3个"},
		SecondaryIngredients: []string{"常用配菜 适量"},
		Steps: []Step{
			{Detail: " 然后把牛腩焯水去腥。 "},
			{Detail: "加入番茄炖煮！"},
		},
	}
	legacy := FromLegacy(
		[]string{" 牛腩 500克 ", "番茄 3个", "常用配菜 适量"},
		[]string{" 然后把牛腩焯水去腥。 ", "加入番茄炖煮！"},
	)

	if !Equal(structured, legacy) {
		t.Fatalf("structured=%#v legacy=%#v", Normalize(structured), Normalize(legacy))
	}
}

func TestNormalizePreservesLinkparseCleaningAndBounds(t *testing.T) {
	t.Parallel()

	mainIngredients := []string{" Tomato 1个。", "tomato 1个", "盐 适量即可"}
	for index := 0; index < MaxIngredientItems+4; index++ {
		mainIngredients = append(mainIngredients, fmt.Sprintf("主料%d 1份", index))
	}
	legacyIngredients := make([]string, 0, MaxIngredientItems+4)
	legacySteps := make([]string, 0, MaxRawSteps+4)
	for index := 0; index < MaxIngredientItems+4; index++ {
		legacyIngredients = append(legacyIngredients, fmt.Sprintf("旧食材%d 1份", index))
	}
	for index := 0; index < MaxRawSteps+4; index++ {
		legacySteps = append(legacySteps, fmt.Sprintf("。第%d步翻炒即可。", index))
	}

	structured := Normalize(Content{
		MainIngredients: mainIngredients,
		Steps: []Step{
			{Title: "调味", Detail: "。盐适量即可。"},
		},
	})
	if got, want := len(structured.MainIngredients), MaxIngredientItems; got != want {
		t.Fatalf("structured ingredient count = %d, want %d", got, want)
	}
	if got, want := structured.MainIngredients[:2], []string{"Tomato 1个", "盐 适量"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("structured ingredients = %#v, want prefix %#v", structured.MainIngredients, want)
	}
	if got, want := structured.Steps[0].Detail, "盐适量"; got != want {
		t.Fatalf("structured step detail = %q, want %q", got, want)
	}

	legacy := Normalize(FromLegacy(legacyIngredients, legacySteps))
	if got, want := len(IngredientLines(legacy)), MaxIngredientItems; got != want {
		t.Fatalf("legacy ingredient count = %d, want %d", got, want)
	}
	if got, want := len(legacy.Steps), MaxSteps; got != want {
		t.Fatalf("legacy step count = %d, want %d", got, want)
	}
	if strings.HasPrefix(legacy.Steps[0].Detail, "。") || strings.HasSuffix(legacy.Steps[0].Detail, "。") {
		t.Fatalf("legacy step detail was not cleaned: %q", legacy.Steps[0].Detail)
	}
}
