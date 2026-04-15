package aialert

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
)

type Config struct {
	Enabled          bool
	FailureThreshold int
	SMTPHost         string
	SMTPPort         int
	SMTPUsername     string
	SMTPPassword     string
	FromEmail        string
	ToEmails         string
}

type ConfigProvider interface {
	AIProviderAlert(context.Context) Config
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
	UpdatedAt               string
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
