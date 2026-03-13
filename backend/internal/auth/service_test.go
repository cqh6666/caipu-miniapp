package auth

import "testing"

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
