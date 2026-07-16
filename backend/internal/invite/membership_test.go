package invite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func TestRepositoryCreateInviteRechecksMembership(t *testing.T) {
	database := openInviteMigrationTestDB(t)
	defer database.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.Exec(`
INSERT INTO users (id, openid, created_at, updated_at)
VALUES (1, 'invite-owner', ?, ?);
INSERT INTO kitchens (id, name, owner_user_id, created_at, updated_at, name_source)
VALUES (1, '邀请空间', 1, ?, ?, 'custom');`, now, now, now, now); err != nil {
		t.Fatal(err)
	}

	_, err := NewRepository(database).Create(context.Background(), createInviteParams{
		KitchenID:     1,
		InviterUserID: 1,
		Token:         "inv_after_leave",
		Code:          "LEAVE001",
		Status:        statusActive,
		MaxUses:       1,
		ExpiresAt:     time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		CreatedAt:     now,
	})
	if !errors.Is(err, common.ErrForbidden) {
		t.Fatalf("Create() error = %v, want forbidden", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(1) FROM kitchen_invites WHERE token = 'inv_after_leave'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("invite count = %d, want 0", count)
	}
}
