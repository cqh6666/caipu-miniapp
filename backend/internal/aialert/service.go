package aialert

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Service struct {
	repo           *Repository
	configProvider ConfigProvider
	sender         Sender
	logger         *slog.Logger
	resolver       ProviderStatusResolver
	retester       ProviderRetester
	deliveryWorker *deliveryWorker
}

func NewService(repo *Repository, configProvider ConfigProvider, sender Sender, logger *slog.Logger) *Service {
	service := &Service{
		repo:           repo,
		configProvider: configProvider,
		sender:         sender,
		logger:         logger,
	}
	service.deliveryWorker = newDeliveryWorker(service, 30*time.Second)
	return service
}

// SetProviderStatusResolver 注入 airouter 的运行时状态解析器（打破包级循环依赖）。
func (s *Service) SetProviderStatusResolver(resolver ProviderStatusResolver) {
	if s == nil {
		return
	}
	s.resolver = resolver
}

// SetProviderRetester 注入 airouter 的单节点复测器。
func (s *Service) SetProviderRetester(retester ProviderRetester) {
	if s == nil {
		return
	}
	s.retester = retester
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
	overview.ActiveWindowHours = cfg.ActiveWindowHours
	overview.HasDeliveryConfig = cfg.ValidateForSend() == nil

	if s == nil || s.repo == nil {
		return overview, nil
	}

	states, err := s.repo.ListStates(ctx, cfg.FailureThreshold)
	if err != nil {
		return Overview{}, err
	}

	statuses := map[string]ProviderRuntimeStatus{}
	resolverAvailable := false
	if s.resolver != nil {
		if resolved, resolveErr := s.resolver.ResolveProviderStatuses(ctx); resolveErr == nil {
			statuses = resolved
			resolverAvailable = true
		} else {
			s.logWarn("resolve provider runtime statuses failed", Event{}, resolveErr)
		}
	}

	now := time.Now().UTC()
	items := make([]OverviewItem, 0, len(states))
	for _, state := range states {
		runtime, found := statuses[state.ProviderID]
		item := buildOverviewItem(state, cfg, runtime, found, resolverAvailable, now)
		switch item.AlertStatus {
		case StatusActive:
			overview.ActiveAlertCount++
		case StatusStale:
			overview.StaleAlertCount++
		case StatusPendingVerify:
			overview.PendingVerifyCount++
		case StatusMuted:
			overview.MutedAlertCount++
		case StatusArchived:
			overview.ArchivedAlertCount++
		case StatusRecovered:
			overview.RecoveredCount++
		}
		if laterTimestamp(item.LastAlertedAt, overview.LatestAlertedAt) {
			overview.LatestAlertedAt = item.LastAlertedAt
		}
		items = append(items, item)
	}
	overview.ReviewAlertCount = overview.StaleAlertCount + overview.PendingVerifyCount + overview.MutedAlertCount
	overview.Items = items
	return overview, nil
}

// buildOverviewItem 把持久化状态 + 运行时状态 + 配置合成为前端消费的 OverviewItem。
func buildOverviewItem(state State, cfg Config, runtime ProviderRuntimeStatus, found, resolverAvailable bool, now time.Time) OverviewItem {
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
		MutedUntil:          state.MutedUntil,
		MuteReason:          state.MuteReason,
		ArchivedAt:          state.ArchivedAt,
		ArchivedBy:          state.ArchivedBy,
		ArchiveReason:       state.ArchiveReason,
		LastConfigChangedAt: state.LastConfigChangedAt,
		UpdatedAt:           state.UpdatedAt,
		ThresholdReached:    state.ConsecutiveFailures >= cfg.FailureThreshold,
	}

	// 运行时存在性：解析器可用且命中则以其为准；解析器不可用时按“仍在路由”兜底（保持旧行为）。
	enabled := true
	inRoute := true
	providerExists := true
	if resolverAvailable {
		enabled = runtime.Enabled
		inRoute = runtime.InEffectiveRoute
		providerExists = found
		if found {
			if strings.TrimSpace(runtime.ProviderName) != "" {
				item.ProviderName = runtime.ProviderName
			}
			if strings.TrimSpace(runtime.Model) != "" {
				item.Model = runtime.Model
			}
			if strings.TrimSpace(runtime.Scene) != "" {
				item.Scene = runtime.Scene
			}
		} else {
			enabled = false
			inRoute = false
		}
	}
	item.IsProviderEnabled = enabled
	item.IsInEffectiveRoute = inRoute

	status, reason, activeUntil := deriveAlertStatus(state, cfg, enabled, inRoute, now)
	item.AlertStatus = status
	item.StatusReason = reason
	item.ActiveUntil = activeUntil

	retestable := status == StatusActive || status == StatusStale || status == StatusPendingVerify || status == StatusMuted
	item.CanRetest = retestable && providerExists
	item.CanArchive = retestable
	item.CanMute = status == StatusActive
	item.CanUnmute = status == StatusMuted
	return item
}

// deriveAlertStatus 依据 §7 的判定规则返回状态、原因文案与活跃窗口截止时刻。
func deriveAlertStatus(state State, cfg Config, enabled, inRoute bool, now time.Time) (AlertStatus, string, string) {
	reached := state.ConsecutiveFailures >= cfg.FailureThreshold

	if !strings.EqualFold(strings.TrimSpace(state.LastStatus), "failed") || state.ConsecutiveFailures == 0 {
		if withinHours(state.LastRecoveredAt, now, 24) {
			return StatusRecovered, "最近一次调用/复测已成功恢复", ""
		}
		return StatusNormal, "", ""
	}

	if !reached {
		return StatusNormal, fmt.Sprintf("连续失败 %d 次，未达阈值 %d 次", state.ConsecutiveFailures, cfg.FailureThreshold), ""
	}

	if timeInFuture(state.MutedUntil, now) {
		reason := "已静默"
		if until := formatLocalHint(state.MutedUntil); until != "" {
			reason = "已静默至 " + until
		}
		return StatusMuted, reason, ""
	}

	if strings.TrimSpace(state.ArchivedAt) != "" {
		reason := "管理员已归档"
		if r := strings.TrimSpace(state.ArchiveReason); r != "" {
			reason = "管理员已归档：" + r
		}
		return StatusArchived, reason, ""
	}

	if timestampAfter(state.LastConfigChangedAt, state.LastFailedAt) {
		return StatusPendingVerify, "配置/密钥/启停已变更，需复测确认", ""
	}

	if !enabled {
		return StatusStale, "节点已禁用，不再计入当前告警", ""
	}
	if !inRoute {
		return StatusStale, "节点不在当前生效路由，不再计入当前告警", ""
	}
	if failedAt, ok := parseTimestamp(state.LastFailedAt); ok {
		window := time.Duration(cfg.ActiveWindowHours) * time.Hour
		if now.Sub(failedAt) > window {
			return StatusStale, fmt.Sprintf("最后失败已超过 %d 小时活跃窗口，需复测确认", cfg.ActiveWindowHours), ""
		}
		activeUntil := failedAt.Add(window).UTC().Format(time.RFC3339)
		return StatusActive, "节点仍启用且在线上路由中，持续失败", activeUntil
	}
	return StatusActive, "节点仍启用且在线上路由中，持续失败", ""
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

	cfg := Config{}
	if s.configProvider != nil {
		cfg = s.configProvider.AIProviderAlert(ctx)
	}
	cfg = cfg.Normalized()
	alertThreshold := 0
	if cfg.Enabled {
		alertThreshold = cfg.FailureThreshold
	}

	_, enqueued, err := s.repo.RecordFailure(ctx, event, alertThreshold)
	if err != nil {
		s.logWarn("record provider alert failure state failed", event, err)
		return
	}
	if enqueued {
		dispatchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		defer cancel()
		if _, err := s.DispatchPending(dispatchCtx, 1); err != nil {
			s.logWarn("dispatch provider alert delivery failed", event, err)
		}
	}
}

// Retest 对已保存线上配置执行单节点复测；成功恢复告警，失败更新错误但不加深计数。
func (s *Service) Retest(ctx context.Context, providerID, subject string) (MutationResult, error) {
	state, err := s.requireState(ctx, providerID)
	if err != nil {
		return MutationResult{}, err
	}
	if s.retester == nil {
		return MutationResult{}, common.NewAppError(common.CodeInternalServer, "复测能力未启用", http.StatusServiceUnavailable)
	}
	outcome, exists, err := s.retester.RetestProvider(ctx, providerID)
	if err != nil {
		return MutationResult{}, err
	}
	if !exists {
		return s.buildMutationResult(ctx, false, "节点已从当前路由中删除，无法复测，只能归档")
	}
	if outcome.OK {
		if err := s.repo.MarkRecovered(ctx, providerID, outcome.RequestID); err != nil {
			return MutationResult{}, err
		}
		_ = s.repo.InsertEvent(ctx, providerID, state.Scene, "retest_succeeded", outcome.Message, subject)
		return s.buildMutationResult(ctx, true, "复测通过，告警已恢复")
	}
	if err := s.repo.RecordRetestFailure(ctx, providerID, outcome); err != nil {
		return MutationResult{}, err
	}
	_ = s.repo.InsertEvent(ctx, providerID, state.Scene, "retest_failed", outcome.Message, subject)
	return s.buildMutationResult(ctx, false, fallbackText(outcome.Message, "复测失败，已更新最后错误"))
}

// Archive 归档告警，不再计入当前告警或待复核。
func (s *Service) Archive(ctx context.Context, providerID, subject, reason string) (MutationResult, error) {
	state, err := s.requireState(ctx, providerID)
	if err != nil {
		return MutationResult{}, err
	}
	if err := s.repo.Archive(ctx, providerID, subject, reason); err != nil {
		return MutationResult{}, err
	}
	_ = s.repo.InsertEvent(ctx, providerID, state.Scene, "archived", reason, subject)
	return s.buildMutationResult(ctx, true, "已归档，该告警不再计入当前状态")
}

// Mute 静默指定时长（小时），静默期内不计入红色告警。
func (s *Service) Mute(ctx context.Context, providerID, subject string, durationHours int, reason string) (MutationResult, error) {
	state, err := s.requireState(ctx, providerID)
	if err != nil {
		return MutationResult{}, err
	}
	if durationHours <= 0 {
		durationHours = 24
	}
	if durationHours > 720 {
		durationHours = 720
	}
	until := time.Now().UTC().Add(time.Duration(durationHours) * time.Hour).Format(time.RFC3339)
	if err := s.repo.Mute(ctx, providerID, subject, until, reason); err != nil {
		return MutationResult{}, err
	}
	_ = s.repo.InsertEvent(ctx, providerID, state.Scene, "muted", reason, subject)
	message := "已静默"
	if hint := formatLocalHint(until); hint != "" {
		message = "已静默至 " + hint
	}
	return s.buildMutationResult(ctx, true, message)
}

// Unmute 立即解除静默。
func (s *Service) Unmute(ctx context.Context, providerID, subject string) (MutationResult, error) {
	state, err := s.requireState(ctx, providerID)
	if err != nil {
		return MutationResult{}, err
	}
	if err := s.repo.Unmute(ctx, providerID); err != nil {
		return MutationResult{}, err
	}
	_ = s.repo.InsertEvent(ctx, providerID, state.Scene, "unmuted", "", subject)
	return s.buildMutationResult(ctx, true, "已解除静默")
}

// NoteSceneConfigChanged 在场景配置保存后，对该场景下已有失败记录的节点标记配置变更，供 pending_verify 判定。
func (s *Service) NoteSceneConfigChanged(ctx context.Context, subject, scene string) error {
	if s == nil || s.repo == nil || strings.TrimSpace(scene) == "" {
		return nil
	}
	if err := s.repo.NoteSceneConfigChanged(ctx, scene); err != nil {
		return err
	}
	ids, err := s.repo.ListProviderIDsByScene(ctx, scene)
	if err != nil {
		return err
	}
	for _, id := range ids {
		_ = s.repo.InsertEvent(ctx, id, scene, "config_changed", "", subject)
	}
	return nil
}

func (s *Service) requireState(ctx context.Context, providerID string) (State, error) {
	if s == nil || s.repo == nil {
		return State{}, common.ErrInternal
	}
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return State{}, common.NewAppError(common.CodeBadRequest, "providerId is required", http.StatusBadRequest)
	}
	state, found, err := s.repo.GetState(ctx, providerID)
	if err != nil {
		return State{}, err
	}
	if !found {
		return State{}, common.ErrNotFound
	}
	return state, nil
}

func (s *Service) buildMutationResult(ctx context.Context, ok bool, message string) (MutationResult, error) {
	overview, err := s.Overview(ctx)
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{OK: ok, Message: message, Overview: overview}, nil
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

func parseTimestamp(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), true
	}
	return time.Time{}, false
}

// timestampAfter 判断 left 是否严格晚于 right（任一无法解析则返回 false）。
func timestampAfter(left, right string) bool {
	leftTime, leftOK := parseTimestamp(left)
	rightTime, rightOK := parseTimestamp(right)
	if !leftOK || !rightOK {
		return false
	}
	return leftTime.After(rightTime)
}

func timeInFuture(value string, now time.Time) bool {
	parsed, ok := parseTimestamp(value)
	if !ok {
		return false
	}
	return parsed.After(now)
}

func withinHours(value string, now time.Time, hours int) bool {
	parsed, ok := parseTimestamp(value)
	if !ok {
		return false
	}
	diff := now.Sub(parsed)
	return diff >= 0 && diff <= time.Duration(hours)*time.Hour
}

// formatLocalHint 把 UTC 时间戳转成东八区可读串，仅用于原因文案提示。
func formatLocalHint(value string) string {
	parsed, ok := parseTimestamp(value)
	if !ok {
		return ""
	}
	loc := time.FixedZone("CST", 8*3600)
	return parsed.In(loc).Format("01-02 15:04")
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
