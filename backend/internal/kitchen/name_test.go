package kitchen

import (
	"strings"
	"testing"
)

func TestFallbackKitchenNameUsesSpaceCopy(t *testing.T) {
	if got, want := fallbackKitchenName, "我的空间"; got != want {
		t.Fatalf("fallbackKitchenName = %q, want %q", got, want)
	}
}

func TestBuildAutoKitchenNameUsesSpaceCopy(t *testing.T) {
	if got, want := buildAutoKitchenName("海哥", 7, ""), "海哥的空间"; got != want {
		t.Fatalf("buildAutoKitchenName() = %q, want %q", got, want)
	}
}

func TestBuildAutoKitchenNameFallbackNicknameUsesSpaceSuffix(t *testing.T) {
	got := buildAutoKitchenName("", 7, "")
	if !strings.HasSuffix(got, "的空间") {
		t.Fatalf("buildAutoKitchenName() = %q, want suffix %q", got, "的空间")
	}
	if strings.Contains(got, "厨房") {
		t.Fatalf("buildAutoKitchenName() = %q, should not contain %q", got, "厨房")
	}
}
