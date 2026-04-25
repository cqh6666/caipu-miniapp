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

func (s *Service) Overview(ctx context.Context) (Overview, error) {
	overview := Overview{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Items:       []OverviewItem{},
	}

	cfg := Config{}
	if s != nil && s.configProvider != nil {
		cfg = s.configProvider.AIProviderAlert(ctx)
	}
	cfg = cfg.Normalized()
	overview.Enabled = cfg.Enabled
	overview.FailureThreshold = cfg.FailureThreshold
	overview.HasDeliveryConfig = cfg.ValidateForSend() == nil

	if s == nil || s.repo == nil {
		return overview, nil
	}

	states, err := s.repo.ListStates(ctx, cfg.FailureThreshold)
	if err != nil {
		return Overview{}, err
	}
	items := make([]OverviewItem, 0, len(states))
	for _, state := range states {
		item := OverviewItem{
			ProviderID:          state.ProviderID,
			ProviderName:        state.ProviderName,
			Scene:               state.Scene,
			Model:               state.Model,
			ConsecutiveFailures: state.ConsecutiveFailures,
			LastStatus:          state.LastStatus,
			LastErrorType:       state.LastErrorType,
			LastErrorMessage:    state.LastErrorMessage,
			LastRequestID:       state.LastRequestID,
			LastFailedAt:        state.LastFailedAt,
			LastRecoveredAt:     state.LastRecoveredAt,
			LastAlertedAt:       state.LastAlertedAt,
			UpdatedAt:           state.UpdatedAt,
			ThresholdReached:    state.ConsecutiveFailures >= cfg.FailureThreshold,
		}
		if item.ThresholdReached {
			overview.ActiveAlertCount++
		}
		if laterTimestamp(item.LastAlertedAt, overview.LatestAlertedAt) {
			overview.LatestAlertedAt = item.LastAlertedAt
		}
		items = append(items, item)
	}
	overview.Items = items
	return overview, nil
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

	recentFailures, err := s.repo.ListRecentFailures(ctx, state.ProviderID, 3)
	if err != nil {
		s.logWarn("list provider recent failures failed", event, err)
		recentFailures = nil
	}

	subject, body := BuildFailureAlertMessage(state, event, recentFailures, cfg.FailureThreshold)
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

func BuildFailureAlertMessage(state State, event Event, recentFailures []FailureSummary, threshold int) (string, string) {
	providerLabel := providerDisplayLabel(state.ProviderName, state.ProviderID)
	subject := fmt.Sprintf(
		"[caipu-miniapp][%s] AI Provider %s 连续异常 %d 次",
		sceneLabel(state.Scene),
		providerSubjectLabel(state.ProviderName, state.ProviderID),
		state.ConsecutiveFailures,
	)
	bodyLines := []string{
		"AI Provider 连续异常告警",
		"",
		fmt.Sprintf("Provider: %s", providerLabel),
		fmt.Sprintf("场景: %s", sceneWithCodeLabel(state.Scene)),
		fmt.Sprintf("模型: %s", fallbackText(state.Model, "-")),
		fmt.Sprintf("触发来源: %s", triggerSourceLabel(event.TriggerSource)),
		fmt.Sprintf("目标对象: %s", targetLabel(event.TargetType, event.TargetID)),
		fmt.Sprintf("连续异常次数: %d", state.ConsecutiveFailures),
		fmt.Sprintf("告警阈值: %d", threshold),
		fmt.Sprintf("最近请求 ID: %s", fallbackText(state.LastRequestID, "-")),
		fmt.Sprintf("最近 HTTP 状态: %d", state.LastHTTPStatus),
		fmt.Sprintf("最近错误类型: %s", fallbackText(state.LastErrorType, "-")),
		fmt.Sprintf("最近错误信息: %s", fallbackText(state.LastErrorMessage, "-")),
		fmt.Sprintf("最近失败时间(UTC): %s", fallbackText(state.LastFailedAt, "-")),
		fmt.Sprintf("最近恢复时间(UTC): %s", fallbackText(state.LastRecoveredAt, "-")),
	}
	if len(recentFailures) > 0 {
		bodyLines = append(bodyLines, "", "最近 3 次失败摘要:")
		for index, item := range recentFailures {
			bodyLines = append(bodyLines,
				fmt.Sprintf(
					"%d. [%s] scene=%s model=%s http=%d type=%s request=%s message=%s",
					index+1,
					fallbackText(item.CreatedAt, "-"),
					fallbackText(item.Scene, "-"),
					fallbackText(item.Model, "-"),
					item.HTTPStatus,
					fallbackText(item.ErrorType, "-"),
					fallbackText(item.RequestID, "-"),
					fallbackText(item.ErrorMessage, "-"),
				),
			)
		}
	}
	bodyLines = append(bodyLines,
		"",
		"排查建议：",
		"- 在后台“AI 任务 / API 调用”里按 Provider 或 Request ID 检索最近失败记录。",
		"- 确认上游 API Key、模型名、余额/限流和网络出口是否正常。",
		"- 同一 Provider 成功一次后，连续失败计数会自动清零。",
	)
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

func providerDisplayLabel(name, id string) string {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if name == "" {
		return fallbackText(id, "-")
	}
	if id == "" {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, id)
}

func providerSubjectLabel(name, id string) string {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if name == "" {
		return fallbackText(id, "-")
	}
	if id == "" {
		return name
	}
	return fmt.Sprintf("%s(%s)", name, id)
}

func sceneLabel(scene string) string {
	switch strings.TrimSpace(scene) {
	case "summary":
		return "做法总结"
	case "title":
		return "标题精修"
	case "flowchart":
		return "流程图"
	default:
		return fallbackText(scene, "未知场景")
	}
}

func sceneWithCodeLabel(scene string) string {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return "-"
	}
	return fmt.Sprintf("%s (%s)", sceneLabel(scene), scene)
}

func triggerSourceLabel(source string) string {
	switch strings.TrimSpace(source) {
	case "worker":
		return "后台 Worker"
	case "manual":
		return "手动触发"
	case "preview":
		return "链接预览"
	default:
		return fallbackText(source, "-")
	}
}

func targetLabel(targetType, targetID string) string {
	targetType = strings.TrimSpace(targetType)
	targetID = strings.TrimSpace(targetID)
	if targetType == "" && targetID == "" {
		return "-"
	}
	if targetID == "" {
		return targetType
	}
	if targetType == "" {
		return targetID
	}
	return fmt.Sprintf("%s / %s", targetType, targetID)
}

func laterTimestamp(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" {
		return false
	}
	if right == "" {
		return true
	}
	leftTime, leftErr := time.Parse(time.RFC3339, left)
	rightTime, rightErr := time.Parse(time.RFC3339, right)
	if leftErr == nil && rightErr == nil {
		return leftTime.After(rightTime)
	}
	return left > right
}
