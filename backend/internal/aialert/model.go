package aialert

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
)

type Config struct {
	Enabled           bool
	FailureThreshold  int
	ActiveWindowHours int
	SMTPHost          string
	SMTPPort          int
	SMTPUsername      string
	SMTPPassword      string
	FromEmail         string
	ToEmails          string
}

type ConfigProvider interface {
	AIProviderAlert(context.Context) Config
}

// AlertStatus 表示单个 Provider 告警状态的语义分层，前端据此决定红/黄/灰/绿。
type AlertStatus string

const (
	StatusNormal        AlertStatus = "normal"
	StatusActive        AlertStatus = "active"
	StatusStale         AlertStatus = "stale"
	StatusPendingVerify AlertStatus = "pending_verify"
	StatusMuted         AlertStatus = "muted"
	StatusArchived      AlertStatus = "archived"
	StatusRecovered     AlertStatus = "recovered"
)

// ProviderRuntimeStatus 由 airouter 反向注入，描述 Provider 在当前生效路由中的存在性。
type ProviderRuntimeStatus struct {
	Enabled          bool
	InEffectiveRoute bool
	Scene            string
	ProviderName     string
	Model            string
}

// ProviderStatusResolver 在 aialert 侧定义、airouter 侧实现，避免包级循环依赖。
type ProviderStatusResolver interface {
	ResolveProviderStatuses(context.Context) (map[string]ProviderRuntimeStatus, error)
}

// ProviderRetestOutcome 是单节点复测结果。
type ProviderRetestOutcome struct {
	OK           bool
	Message      string
	Model        string
	HTTPStatus   int
	ErrorType    string
	ErrorMessage string
	RequestID    string
}

// ProviderRetester 由 airouter 实现，对已保存线上配置执行单节点真实复测。
type ProviderRetester interface {
	RetestProvider(ctx context.Context, providerID string) (ProviderRetestOutcome, bool, error)
}

// MutationResult 是复测/归档/静默/解除静默动作的统一返回，携带重算后的概览。
type MutationResult struct {
	OK       bool     `json:"ok"`
	Message  string   `json:"message"`
	Overview Overview `json:"overview"`
}

type Event struct {
	Scene         string
	ProviderID    string
	ProviderName  string
	Model         string
	HTTPStatus    int
	ErrorType     string
	ErrorMessage  string
	RequestID     string
	TriggerSource string
	TargetType    string
	TargetID      string
	OccurredAt    string
}

type State struct {
	ProviderID              string
	Scene                   string
	ProviderName            string
	Model                   string
	ConsecutiveFailures     int
	LastStatus              string
	LastErrorType           string
	LastErrorMessage        string
	LastHTTPStatus          int
	LastRequestID           string
	LastFailedAt            string
	LastRecoveredAt         string
	LastAlertedAt           string
	LastAlertedFailureCount int
	ArchivedAt              string
	ArchivedBy              string
	ArchiveReason           string
	MutedUntil              string
	MutedBy                 string
	MuteReason              string
	LastConfigChangedAt     string
	UpdatedAt               string
}

type Overview struct {
	GeneratedAt        string         `json:"generatedAt"`
	Enabled            bool           `json:"enabled"`
	FailureThreshold   int            `json:"failureThreshold"`
	ActiveWindowHours  int            `json:"activeWindowHours"`
	HasDeliveryConfig  bool           `json:"hasDeliveryConfig"`
	ActiveAlertCount   int            `json:"activeAlertCount"`
	StaleAlertCount    int            `json:"staleAlertCount"`
	PendingVerifyCount int            `json:"pendingVerifyCount"`
	MutedAlertCount    int            `json:"mutedAlertCount"`
	ArchivedAlertCount int            `json:"archivedAlertCount"`
	RecoveredCount     int            `json:"recoveredCount"`
	ReviewAlertCount   int            `json:"reviewAlertCount"`
	LatestAlertedAt    string         `json:"latestAlertedAt"`
	Items              []OverviewItem `json:"items"`
}

type OverviewItem struct {
	ProviderID          string      `json:"providerId"`
	ProviderName        string      `json:"providerName"`
	Scene               string      `json:"scene"`
	Model               string      `json:"model"`
	ConsecutiveFailures int         `json:"consecutiveFailures"`
	LastStatus          string      `json:"lastStatus"`
	LastErrorType       string      `json:"lastErrorType"`
	LastErrorMessage    string      `json:"lastErrorMessage"`
	LastRequestID       string      `json:"lastRequestId"`
	LastFailedAt        string      `json:"lastFailedAt"`
	LastRecoveredAt     string      `json:"lastRecoveredAt"`
	LastAlertedAt       string      `json:"lastAlertedAt"`
	UpdatedAt           string      `json:"updatedAt"`
	AlertStatus         AlertStatus `json:"alertStatus"`
	StatusReason        string      `json:"statusReason"`
	ActiveUntil         string      `json:"activeUntil,omitempty"`
	MutedUntil          string      `json:"mutedUntil,omitempty"`
	MuteReason          string      `json:"muteReason,omitempty"`
	ArchivedAt          string      `json:"archivedAt,omitempty"`
	ArchivedBy          string      `json:"archivedBy,omitempty"`
	ArchiveReason       string      `json:"archiveReason,omitempty"`
	LastConfigChangedAt string      `json:"lastConfigChangedAt,omitempty"`
	IsProviderEnabled   bool        `json:"isProviderEnabled"`
	IsInEffectiveRoute  bool        `json:"isInEffectiveRoute"`
	CanRetest           bool        `json:"canRetest"`
	CanArchive          bool        `json:"canArchive"`
	CanMute             bool        `json:"canMute"`
	CanUnmute           bool        `json:"canUnmute"`
	ThresholdReached    bool        `json:"thresholdReached"`
}

type FailureSummary struct {
	Scene        string
	Model        string
	HTTPStatus   int
	ErrorType    string
	ErrorMessage string
	RequestID    string
	CreatedAt    string
}

type SendRequest struct {
	Config  Config
	Subject string
	Body    string
}

type Sender interface {
	Send(context.Context, SendRequest) error
}

type Tracker interface {
	RecordSuccess(context.Context, Event)
	RecordFailure(context.Context, Event)
}

func (c Config) Normalized() Config {
	normalized := c
	normalized.SMTPHost = strings.TrimSpace(normalized.SMTPHost)
	normalized.SMTPUsername = strings.TrimSpace(normalized.SMTPUsername)
	normalized.SMTPPassword = strings.TrimSpace(normalized.SMTPPassword)
	normalized.FromEmail = strings.TrimSpace(normalized.FromEmail)
	normalized.ToEmails = normalizeRecipients(normalized.ToEmails)
	if normalized.FailureThreshold <= 0 {
		normalized.FailureThreshold = 3
	}
	if normalized.ActiveWindowHours <= 0 {
		normalized.ActiveWindowHours = 72
	}
	if normalized.SMTPHost == "" {
		normalized.SMTPHost = "smtp.qq.com"
	}
	if normalized.SMTPPort <= 0 {
		normalized.SMTPPort = 587
	}
	if normalized.FromEmail == "" {
		normalized.FromEmail = normalized.SMTPUsername
	}
	return normalized
}

func (c Config) FromAddress() string {
	return c.Normalized().FromEmail
}

func (c Config) Recipients() []string {
	normalized := c.Normalized()
	if normalized.ToEmails == "" {
		return nil
	}
	return strings.Split(normalized.ToEmails, ",")
}

func (c Config) ValidateForSend() error {
	normalized := c.Normalized()
	if normalized.SMTPHost == "" {
		return fmt.Errorf("SMTP Host 未配置")
	}
	if normalized.SMTPPort <= 0 {
		return fmt.Errorf("SMTP Port 必须大于 0")
	}
	if normalized.SMTPUsername == "" {
		return fmt.Errorf("SMTP 用户名未配置")
	}
	if normalized.SMTPPassword == "" {
		return fmt.Errorf("SMTP 授权码未配置")
	}
	if _, err := mail.ParseAddress(normalized.FromEmail); err != nil {
		return fmt.Errorf("发件邮箱格式不合法: %w", err)
	}
	recipients := normalized.Recipients()
	if len(recipients) == 0 {
		return fmt.Errorf("收件邮箱未配置")
	}
	for _, recipient := range recipients {
		if _, err := mail.ParseAddress(recipient); err != nil {
			return fmt.Errorf("收件邮箱格式不合法: %w", err)
		}
	}
	return nil
}

func normalizeRecipients(raw string) string {
	replacer := strings.NewReplacer("\n", ",", ";", ",", "，", ",")
	parts := strings.Split(replacer.Replace(raw), ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		items = append(items, value)
	}
	return strings.Join(items, ",")
}
