package kitchen

type Summary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Member struct {
	UserID        int64  `json:"userId"`
	Nickname      string `json:"nickname"`
	AvatarURL     string `json:"avatarUrl"`
	Role          string `json:"role"`
	JoinedAt      string `json:"joinedAt"`
	IsCurrentUser bool   `json:"isCurrentUser"`
}

type createKitchenRequest struct {
	Name string `json:"name"`
}

type updateKitchenRequest struct {
	Name string `json:"name"`
}
