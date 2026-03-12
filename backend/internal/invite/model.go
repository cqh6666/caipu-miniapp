package invite

import "github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"

const (
	statusActive  = "active"
	statusUsedUp  = "used_up"
	statusRevoked = "revoked"
	statusExpired = "expired"
)

type Inviter struct {
	ID       int64  `json:"id"`
	Nickname string `json:"nickname"`
}

type Invite struct {
	ID            int64   `json:"id"`
	KitchenID     int64   `json:"kitchenId"`
	KitchenName   string  `json:"kitchenName"`
	Token         string  `json:"token"`
	Status        string  `json:"status"`
	MaxUses       int     `json:"maxUses"`
	UsedCount     int     `json:"usedCount"`
	RemainingUses int     `json:"remainingUses"`
	ExpiresAt     string  `json:"expiresAt"`
	CreatedAt     string  `json:"createdAt"`
	SharePath     string  `json:"sharePath"`
	Inviter       Inviter `json:"inviter"`
}

type AcceptResult struct {
	Invite           Invite            `json:"invite"`
	Kitchen          kitchen.Summary   `json:"kitchen"`
	Kitchens         []kitchen.Summary `json:"kitchens"`
	CurrentKitchenID int64             `json:"currentKitchenId"`
	AlreadyMember    bool              `json:"alreadyMember"`
}

type createInviteRequest struct {
	MaxUses        int `json:"maxUses"`
	ExpiresInHours int `json:"expiresInHours"`
}

type inviteRecord struct {
	ID              int64
	KitchenID       int64
	KitchenName     string
	InviterUserID   int64
	InviterNickname string
	Token           string
	Status          string
	MaxUses         int
	UsedCount       int
	ExpiresAt       string
	CreatedAt       string
}

type createInviteParams struct {
	KitchenID     int64
	InviterUserID int64
	Token         string
	Status        string
	MaxUses       int
	ExpiresAt     string
	CreatedAt     string
}

type acceptInviteResult struct {
	KitchenID     int64
	KitchenName   string
	Role          string
	AlreadyMember bool
}
