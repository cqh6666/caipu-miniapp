package recipecontent_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipe"
	"github.com/cqh6666/caipu-miniapp/backend/internal/recipecontent"
)

func TestLinkPreviewAndStoredRecipeShareContentContract(t *testing.T) {
	t.Parallel()

	input := []byte(`{
		"ingredients":["牛腩 500克","番茄 3个","常用调味料 适量"],
		"steps":["先把牛腩焯水去腥。","然后加入番茄炖煮。"]
	}`)
	var preview linkparse.ParsedContent
	var stored recipe.ParsedContent
	if err := json.Unmarshal(input, &preview); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(input, &stored); err != nil {
		t.Fatal(err)
	}

	previewNormalized := recipecontent.Normalize(preview)
	storedNormalized := recipecontent.Normalize(stored)
	if !reflect.DeepEqual(previewNormalized, storedNormalized) {
		t.Fatalf("preview=%#v stored=%#v", previewNormalized, storedNormalized)
	}

	stored = preview
	if !reflect.DeepEqual(stored, preview) {
		t.Fatal("business packages no longer share the same content type")
	}
}
