package app

import (
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/admin"
	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/cqh6666/caipu-miniapp/backend/internal/invite"
	"github.com/cqh6666/caipu-miniapp/backend/internal/ratelimit"
)

func configureRequestGuards(adminHandler *admin.Handler, authHandler *auth.Handler, inviteHandler *invite.Handler) {
	adminHandler.SetLoginGuard(ratelimit.NewGuard(
		"admin_login",
		ratelimit.Policy{MaxAttempts: 10, Window: 5 * time.Minute, BanDuration: 15 * time.Minute},
		ratelimit.Policy{MaxAttempts: 5, Window: 5 * time.Minute, BanDuration: 15 * time.Minute},
	))
	authHandler.SetLoginGuard(ratelimit.NewGuard(
		"user_login",
		ratelimit.Policy{MaxAttempts: 30, Window: time.Minute, BanDuration: 5 * time.Minute},
		ratelimit.Policy{MaxAttempts: 3, Window: time.Minute, BanDuration: 5 * time.Minute},
	))
	inviteHandler.SetRequestGuards(
		ratelimit.NewGuard(
			"invite_preview",
			ratelimit.Policy{MaxAttempts: 60, Window: time.Minute, BanDuration: 5 * time.Minute},
			ratelimit.Policy{MaxAttempts: 20, Window: time.Minute, BanDuration: 5 * time.Minute},
		),
		ratelimit.NewGuard(
			"invite_accept",
			ratelimit.Policy{MaxAttempts: 20, Window: time.Minute, BanDuration: 10 * time.Minute},
			ratelimit.Policy{MaxAttempts: 5, Window: time.Minute, BanDuration: 10 * time.Minute},
		),
	)
}
