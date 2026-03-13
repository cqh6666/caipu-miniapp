package appsettings

const (
	BilibiliSessionStatusUnconfigured = "unconfigured"
	BilibiliSessionStatusValid        = "valid"
)

type BilibiliSessionSetting struct {
	Configured     bool   `json:"configured"`
	Status         string `json:"status"`
	MaskedSessdata string `json:"maskedSessdata"`
	LastCheckedAt  string `json:"lastCheckedAt"`
	LastSuccessAt  string `json:"lastSuccessAt"`
	LastError      string `json:"lastError"`
	UpdatedAt      string `json:"updatedAt"`
}

type bilibiliSessionRecord struct {
	SessdataCiphertext string
	MaskedSessdata     string
	Status             string
	LastCheckedAt      string
	LastSuccessAt      string
	LastError          string
	UpdatedBy          int64
	UpdatedAt          string
}

type updateBilibiliSessionRequest struct {
	Sessdata string `json:"sessdata"`
}
