package recipe

import (
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
)

func TestBuildAutoParseSourceKeepsLegacyXiaohongshuShapeWithoutDetail(t *testing.T) {
	t.Parallel()

	got := buildAutoParseSource(linkparse.RecipeParseOutcome{
		Source:      "xiaohongshu",
		SummaryMode: "ai",
	})

	if want := "xiaohongshu:ai"; got != want {
		t.Fatalf("buildAutoParseSource() = %q, want %q", got, want)
	}
}

func TestBuildAutoParseSourceIncludesXiaohongshuVideoDetail(t *testing.T) {
	t.Parallel()

	got := buildAutoParseSource(linkparse.RecipeParseOutcome{
		Source:       "xiaohongshu",
		SourceDetail: "video",
		SummaryMode:  "ai",
	})

	if want := "xiaohongshu:video:ai"; got != want {
		t.Fatalf("buildAutoParseSource() = %q, want %q", got, want)
	}
}

func TestBuildAutoParseSourceIgnoresUnknownDetail(t *testing.T) {
	t.Parallel()

	got := buildAutoParseSource(linkparse.RecipeParseOutcome{
		Source:       "xiaohongshu",
		SourceDetail: "unknown",
		SummaryMode:  "heuristic",
	})

	if want := "xiaohongshu:heuristic"; got != want {
		t.Fatalf("buildAutoParseSource() = %q, want %q", got, want)
	}
}
