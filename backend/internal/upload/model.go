package upload

type Image struct {
	URL         string `json:"url"`
	ContentHash string `json:"contentHash,omitempty"`
}
