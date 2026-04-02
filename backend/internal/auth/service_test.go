package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func TestCanManageAppSettingsModes(t *testing.T) {
	adminUser := User{OpenID: "dev:alice"}
	normalUser := User{OpenID: "dev:bob"}

	tests := []struct {
		name           string
		mode           string
		adminOpenIDs   []string
		allowedOpenIDs []string
		user           User
		want           bool
		wantAdmin      bool
	}{
		{
			name:      "all mode allows everyone",
			mode:      "all",
			user:      normalUser,
			want:      true,
			wantAdmin: false,
		},
		{
			name:         "admin mode allows admin only",
			mode:         "admin",
			adminOpenIDs: []string{"dev:alice"},
			user:         adminUser,
			want:         true,
			wantAdmin:    true,
		},
		{
			name:         "admin mode blocks normal user",
			mode:         "admin",
			adminOpenIDs: []string{"dev:alice"},
			user:         normalUser,
			want:         false,
			wantAdmin:    false,
		},
		{
			name:           "whitelist mode allows listed user",
			mode:           "whitelist",
			allowedOpenIDs: []string{"dev:bob"},
			user:           normalUser,
			want:           true,
			wantAdmin:      false,
		},
		{
			name:         "whitelist mode still allows admin user",
			mode:         "whitelist",
			adminOpenIDs: []string{"dev:alice"},
			user:         adminUser,
			want:         true,
			wantAdmin:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(nil, nil, nil, nil, "", test.adminOpenIDs, test.mode, test.allowedOpenIDs)
			got := service.enrichUser(test.user)
			if got.CanManageAppSettings != test.want {
				t.Fatalf("CanManageAppSettings = %v, want %v", got.CanManageAppSettings, test.want)
			}
			if got.IsAdmin != test.wantAdmin {
				t.Fatalf("IsAdmin = %v, want %v", got.IsAdmin, test.wantAdmin)
			}
		})
	}
}

func TestUpdateProfileRejectsTemporaryAvatarURL(t *testing.T) {
	service := NewService(nil, nil, nil, nil, "", nil, "", nil)

	_, err := service.UpdateProfile(context.Background(), 1, "alice", "https://tmp/avatar.png")
	if err == nil {
		t.Fatal("expected temporary avatar url to be rejected")
	}

	var appErr *common.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != common.CodeBadRequest {
		t.Fatalf("app error code = %d, want %d", appErr.Code, common.CodeBadRequest)
	}
}

func TestSanitizeLoginAvatarURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "keeps regular avatar url",
			input: "https://wx.qlogo.cn/mmopen/vi_32/demo/132",
			want:  "https://wx.qlogo.cn/mmopen/vi_32/demo/132",
		},
		{
			name:  "drops temporary avatar url",
			input: "https://tmp/avatar.png",
			want:  "",
		},
		{
			name:  "trims whitespace",
			input: "  https://example.com/avatar.png  ",
			want:  "https://example.com/avatar.png",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sanitizeLoginAvatarURL(test.input); got != test.want {
				t.Fatalf("sanitizeLoginAvatarURL(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestIsTemporaryAvatarURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "wxfile scheme",
			input: "wxfile://tmp/avatar",
			want:  true,
		},
		{
			name:  "tmp http host",
			input: "https://tmp/avatar.png",
			want:  true,
		},
		{
			name:  "blob scheme",
			input: "blob:1234",
			want:  true,
		},
		{
			name:  "normal https avatar",
			input: "https://example.com/avatar.png",
			want:  false,
		},
		{
			name:  "empty value",
			input: "",
			want:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isTemporaryAvatarURL(test.input); got != test.want {
				t.Fatalf("isTemporaryAvatarURL(%q) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}
