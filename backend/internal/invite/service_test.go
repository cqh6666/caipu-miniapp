package invite

import (
	"bytes"
	"image/png"
	"testing"
)

func TestToInviteAndDecorateInvite(t *testing.T) {
	record := inviteRecord{
		ID:              12,
		KitchenID:       34,
		KitchenName:     "海哥的厨房",
		InviterUserID:   56,
		InviterNickname: "海哥",
		Token:           "inv_demo123",
		Code:            "ABCD1234",
		Status:          statusActive,
		MaxUses:         10,
		UsedCount:       2,
		ExpiresAt:       "2026-04-08T10:00:00+08:00",
		CreatedAt:       "2026-04-05T10:00:00+08:00",
	}
	item := toInvite(record)

	if got, want := item.SharePath, "/pages/invite/index?token=inv_demo123"; got != want {
		t.Fatalf("SharePath = %q, want %q", got, want)
	}

	if item.ShareImageURL != "" {
		t.Fatalf("ShareImageURL = %q, want empty before decorateInvite", item.ShareImageURL)
	}

	if got, want := item.RemainingUses, 8; got != want {
		t.Fatalf("RemainingUses = %d, want %d", got, want)
	}

	service := NewService(nil, nil, 72, 10, NewShareImageRenderer("", ""))
	if _, err := service.shareImageRenderer.face(false, 24); err != nil {
		t.Skipf("skip decorateInvite share image assertion: %v", err)
	}

	decorated := service.decorateInvite(item)
	if got, want := decorated.ShareImageURL, "/api/invites/inv_demo123/share-image"; got != want {
		t.Fatalf("ShareImageURL = %q, want %q", got, want)
	}
}

func TestFormatInviteCode(t *testing.T) {
	if got, want := formatInviteCode("ab cd-1234"), "ABCD 1234"; got != want {
		t.Fatalf("formatInviteCode = %q, want %q", got, want)
	}
}

func TestBuildInviteHeroLineUsesSpaceCopy(t *testing.T) {
	if got, want := buildInviteHeroLine("海哥"), "海哥 发来一份共享空间邀请"; got != want {
		t.Fatalf("buildInviteHeroLine() = %q, want %q", got, want)
	}
}

func TestReplaceKitchenLabel(t *testing.T) {
	if got, want := replaceKitchenLabel("海哥的厨房"), "海哥的空间"; got != want {
		t.Fatalf("replaceKitchenLabel() = %q, want %q", got, want)
	}
}

func TestShareImageRendererRender(t *testing.T) {
	renderer := NewShareImageRenderer("", "")
	if _, err := renderer.face(false, 24); err != nil {
		t.Skipf("skip share image render test: %v", err)
	}

	imageBytes, err := renderer.Render(ShareImageData{
		KitchenName:   "海哥的厨房",
		InviterName:   "海哥",
		InviteCode:    "ABCD1234",
		Status:        statusActive,
		MemberCount:   3,
		RemainingUses: 7,
		ExpiresAt:     "2026-04-08T10:00:00+08:00",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	decoded, err := png.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		t.Fatalf("png.Decode() error = %v", err)
	}

	if got, want := decoded.Bounds().Dx(), shareImageWidth; got != want {
		t.Fatalf("rendered width = %d, want %d", got, want)
	}
	if got, want := decoded.Bounds().Dy(), shareImageHeight; got != want {
		t.Fatalf("rendered height = %d, want %d", got, want)
	}
}
