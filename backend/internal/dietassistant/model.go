package dietassistant

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatStreamRequest struct {
	Messages []ChatMessage `json:"messages"`
}

type StreamEvent struct {
	Type    string `json:"type"`
	Delta   string `json:"delta,omitempty"`
	Message string `json:"message,omitempty"`
}
