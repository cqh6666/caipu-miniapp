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

func TestBuildAutoParseResultMessageUsesFirstWarning(t *testing.T) {
	t.Parallel()

	got := buildAutoParseResultMessage(linkparse.RecipeParseOutcome{
		SummaryMode: "heuristic",
		Warnings: []string{
			"AI 总结失败：未提供令牌；已回退到规则整理。",
			"其他提示",
		},
	})

	if want := "AI 总结失败：未提供令牌；已回退到规则整理。"; got != want {
		t.Fatalf("buildAutoParseResultMessage() = %q, want %q", got, want)
	}
}
