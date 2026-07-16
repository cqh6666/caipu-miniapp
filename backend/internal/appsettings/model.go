package appsettings

const (
	BilibiliSessionStatusUnconfigured = "unconfigured"
	BilibiliSessionStatusValid        = "valid"
)

type BilibiliSessionSetting struct {
	Version        int    `json:"version"`
	Configured     bool   `json:"configured"`
	Status         string `json:"status"`
	MaskedSessdata string `json:"maskedSessdata"`
	LastCheckedAt  string `json:"lastCheckedAt"`
	LastSuccessAt  string `json:"lastSuccessAt"`
	LastError      string `json:"lastError"`
	UpdatedAt      string `json:"updatedAt"`
}

type bilibiliSessionRecord struct {
	Version            int
	SessdataCiphertext string
	MaskedSessdata     string
	Status             string
	LastCheckedAt      string
	LastSuccessAt      string
	LastError          string
	UpdatedBy          *int64
	UpdatedBySubject   string
	UpdatedAt          string
}

type updateBilibiliSessionRequest struct {
	Sessdata string `json:"sessdata"`
}
