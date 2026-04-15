package aialert

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type Service struct {
	repo           *Repository
	configProvider ConfigProvider
	sender         Sender
	logger         *slog.Logger
}

func NewService(repo *Repository, configProvider ConfigProvider, sender Sender, logger *slog.Logger) *Service {
	return &Service{
		repo:           repo,
		configProvider: configProvider,
		sender:         sender,
		logger:         logger,
	}
}

func (s *Service) RecordSuccess(ctx context.Context, event Event) {
	if s == nil || s.repo == nil || strings.TrimSpace(event.ProviderID) == "" {
		return
	}
	if err := s.repo.RecordSuccess(ctx, event); err != nil {
		s.logWarn("reset provider alert state failed", event, err)
	}
}

func (s *Service) RecordFailure(ctx context.Context, event Event) {
	if s == nil || s.repo == nil || strings.TrimSpace(event.ProviderID) == "" {
		return
	}

	state, err := s.repo.RecordFailure(ctx, event)
	if err != nil {
		s.logWarn("record provider alert failure state failed", event, err)
		return
	}

	cfg := Config{}
	if s.configProvider != nil {
		cfg = s.configProvider.AIProviderAlert(ctx)
	}
	cfg = cfg.Normalized()
	if !cfg.Enabled || state.ConsecutiveFailures < cfg.FailureThreshold {
		return
	}
	if state.LastAlertedFailureCount >= cfg.FailureThreshold {
		return
	}
	if s.sender == nil {
		s.logWarn("provider alert sender is not configured", event, nil)
		return
	}

	subject, body := BuildFailureAlertMessage(state, cfg.FailureThreshold)
	sendCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := s.sender.Send(sendCtx, SendRequest{
		Config:  cfg,
		Subject: subject,
		Body:    body,
	}); err != nil {
		s.logWarn("send provider alert email failed", event, err)
		return
	}
	if err := s.repo.MarkAlertSent(ctx, state.ProviderID, state.ConsecutiveFailures, time.Now().UTC().Format(time.RFC3339)); err != nil {
		s.logWarn("persist provider alert sent state failed", event, err)
	}
}

func BuildFailureAlertMessage(state State, threshold int) (string, string) {
	providerLabel := state.ProviderID
	if strings.TrimSpace(state.ProviderName) != "" {
		providerLabel = fmt.Sprintf("%s (%s)", state.ProviderName, state.ProviderID)
	}
	subject := fmt.Sprintf("[caipu-miniapp] AI Provider %s 连续异常 %d 次", state.ProviderID, state.ConsecutiveFailures)
	bodyLines := []string{
		"AI Provider 连续异常告警",
		"",
		fmt.Sprintf("Provider: %s", providerLabel),
		fmt.Sprintf("Scene: %s", fallbackText(state.Scene, "-")),
		fmt.Sprintf("Model: %s", fallbackText(state.Model, "-")),
		fmt.Sprintf("连续异常次数: %d", state.ConsecutiveFailures),
		fmt.Sprintf("告警阈值: %d", threshold),
		fmt.Sprintf("最近请求 ID: %s", fallbackText(state.LastRequestID, "-")),
		fmt.Sprintf("最近 HTTP 状态: %d", state.LastHTTPStatus),
		fmt.Sprintf("最近错误类型: %s", fallbackText(state.LastErrorType, "-")),
		fmt.Sprintf("最近错误信息: %s", fallbackText(state.LastErrorMessage, "-")),
		fmt.Sprintf("最近失败时间(UTC): %s", fallbackText(state.LastFailedAt, "-")),
		fmt.Sprintf("最近恢复时间(UTC): %s", fallbackText(state.LastRecoveredAt, "-")),
		"",
		"说明：同一 Provider 成功一次后，连续失败计数会自动清零。",
	}
	return subject, strings.Join(bodyLines, "\n")
}

func BuildTestMessage() (string, string) {
	now := time.Now().UTC().Format(time.RFC3339)
	return "[caipu-miniapp] AI Provider 告警测试邮件", strings.Join([]string{
		"这是一封 AI Provider 告警测试邮件。",
		"",
		fmt.Sprintf("发送时间(UTC): %s", now),
		"如果你收到了这封邮件，说明当前 SMTP 与收件人配置可用。",
	}, "\n")
}

func (s *Service) logWarn(message string, event Event, err error) {
	if s == nil || s.logger == nil {
		return
	}
	args := []any{
		"provider_id", strings.TrimSpace(event.ProviderID),
		"scene", strings.TrimSpace(event.Scene),
	}
	if err != nil {
		args = append(args, "err", err)
	}
	s.logger.Warn(message, args...)
}

func fallbackText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
