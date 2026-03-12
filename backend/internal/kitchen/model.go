package kitchen

type Summary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type createKitchenRequest struct {
	Name string `json:"name"`
}
