package auth

import "github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"

type User struct {
	ID        int64  `json:"id"`
	OpenID    string `json:"openid"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type SessionResponse struct {
	Token            string            `json:"token,omitempty"`
	User             User              `json:"user"`
	CurrentKitchenID int64             `json:"currentKitchenId"`
	Kitchens         []kitchen.Summary `json:"kitchens"`
}

type wechatLoginRequest struct {
	Code  string `json:"code"`
	AppID string `json:"appId"`
}

type devLoginRequest struct {
	Identity string `json:"identity"`
}
