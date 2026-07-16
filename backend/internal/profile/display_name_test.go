package profile

import "testing"

func TestDisplayNamePreservesExplicitNickname(t *testing.T) {
	if got := DisplayName("  小陈  ", 42, "openid"); got != "小陈" {
		t.Fatalf("DisplayName()=%q", got)
	}
}

func TestDisplayNameReplacesPlatformAndGeneratedPlaceholders(t *testing.T) {
	fallback := FallbackNickname(42, "")
	for _, value := range []string{"", "微信用户", "Wechat User", fallback} {
		if got := DisplayName(value, 42, "ignored"); got != fallback {
			t.Fatalf("DisplayName(%q)=%q, want=%q", value, got, fallback)
		}
	}
}

func TestFallbackNicknameIsStableForUserOrOpenID(t *testing.T) {
	byUser := FallbackNickname(7, "first-openid")
	if byUser != FallbackNickname(7, "different-openid") || !IsFallbackNickname(byUser) {
		t.Fatalf("user fallback is not stable: %q", byUser)
	}
	byOpenID := FallbackNickname(0, "stable-openid")
	if byOpenID != FallbackNickname(0, " stable-openid ") || !IsPlaceholderNickname(byOpenID) {
		t.Fatalf("openid fallback is not stable: %q", byOpenID)
	}
}
