package airouter

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func (s *Service) routeChat(ctx context.Context, config SceneConfig, input ChatCompletionInput) (ChatCompletionResult, error) {
	normalizeSceneConfig(&config)
	providers := enabledProviders(config.Providers)
	if !config.Enabled || len(providers) == 0 {
		return ChatCompletionResult{}, common.NewAppError(common.CodeInternalServer, "ai routing is not configured for this scene", http.StatusServiceUnavailable)
	}

	order := buildAttemptOrder(config.Scene, config.Strategy, providers, s.currentRoundRobinStart(config.Scene, len(providers)))
	if len(order) == 0 {
		return ChatCompletionResult{}, common.NewAppError(common.CodeInternalServer, "ai routing has no enabled providers", http.StatusServiceUnavailable)
	}

	maxAttempts := config.MaxAttempts
	if maxAttempts <= 0 || maxAttempts > len(order) {
		maxAttempts = len(order)
	}

	now := time.Now()
	result := ChatCompletionResult{
		Strategy: config.Strategy,
		Attempts: make([]AttemptResult, 0, len(order)),
	}
	actualAttempts := 0
	var lastErr error

	for _, candidate := range order {
		if actualAttempts >= maxAttempts {
			break
		}

		if open, openUntil := s.breaker.isOpen(config.Scene, candidate.ID, now); open {
			result.Attempts = append(result.Attempts, AttemptResult{
				ProviderID:       candidate.ID,
				ProviderName:     candidate.Name,
				Model:            candidate.Model,
				Status:           audit.CallStatusFailed,
				ErrorType:        ErrorTypeBreakerOpen,
				ErrorMessage:     "provider skipped by breaker",
				SkippedByBreaker: true,
				BreakerOpenUntil: openUntil.UTC().Format(time.RFC3339),
			})
			continue
		}

		actualAttempts++
		if result.StartedProvider == "" {
			result.StartedProvider = candidate.ID
		}

		content, endpoint, httpStatus, latencyMS, callErr := s.callOpenAICompatible(ctx, config, candidate, input)
		if callErr == nil && input.ValidateContent != nil {
			callErr = normalizeValidationError(input.ValidateContent(content))
		}
		if callErr == nil {
			s.breaker.markSuccess(config.Scene, candidate.ID)
			result.Content = content
			result.ProviderID = candidate.ID
			result.ProviderName = candidate.Name
			result.Model = candidate.Model
			result.FallbackUsed = actualAttempts > 1
			result.AttemptCount = actualAttempts
			result.Attempts = append(result.Attempts, AttemptResult{
				ProviderID:   candidate.ID,
				ProviderName: candidate.Name,
				Model:        candidate.Model,
				Status:       audit.CallStatusSuccess,
				HTTPStatus:   httpStatus,
				LatencyMS:    latencyMS,
			})
			s.logCall(ctx, config, candidate, actualAttempts, endpoint, httpStatus, latencyMS, nil, input)
			s.trackProviderAlert(ctx, config, candidate, httpStatus, nil, input)
			if config.Strategy == StrategyRoundRobinFailover {
				s.setRoundRobinNext(config.Scene, candidate.originalIndex+1, len(providers))
			}
			return result, nil
		}

		s.logCall(ctx, config, candidate, actualAttempts, endpoint, httpStatus, latencyMS, callErr, input)
		s.trackProviderAlert(ctx, config, candidate, httpStatus, callErr, input)
		errorType := routeErrorType(callErr)
		attempt := AttemptResult{
			ProviderID:   candidate.ID,
			ProviderName: candidate.Name,
			Model:        candidate.Model,
			Status:       audit.CallStatusFromError(callErr),
			HTTPStatus:   httpStatus,
			LatencyMS:    latencyMS,
			ErrorType:    errorType,
			ErrorMessage: callErr.Error(),
		}
		if shouldRetry(config.RetryOn, errorType) {
			if openUntil := s.breaker.markFailure(config.Scene, candidate.ID, config.Breaker, time.Now()); !openUntil.IsZero() {
				attempt.BreakerOpenUntil = openUntil.UTC().Format(time.RFC3339)
			}
		}
		result.Attempts = append(result.Attempts, attempt)
		lastErr = callErr
		if !shouldRetry(config.RetryOn, errorType) {
			break
		}
	}

	result.AttemptCount = actualAttempts
	result.FallbackUsed = actualAttempts > 1
	if lastErr == nil {
		lastErr = common.NewAppError(common.CodeInternalServer, "all providers are cooling down", http.StatusBadGateway)
	}
	return result, lastErr
}

func (s *Service) trackProviderAlert(ctx context.Context, config SceneConfig, provider orderedProvider, httpStatus int, err error, input ChatCompletionInput) {
	if s == nil || s.alertTracker == nil || input.ContentKind == "route_test" {
		return
	}

	event := aialert.Event{
		Scene:        string(config.Scene),
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		Model:        provider.Model,
		HTTPStatus:   httpStatus,
		ErrorType:    routeErrorType(err),
		ErrorMessage: errorMessage(err),
		RequestID:    common.RequestID(ctx),
		OccurredAt:   time.Now().UTC().Format(time.RFC3339),
	}
	if meta, ok := audit.CurrentRequestMeta(ctx); ok {
		event.TriggerSource = meta.TriggerSource
		event.TargetType = meta.TargetType
		event.TargetID = meta.TargetID
	}
	if err == nil {
		s.alertTracker.RecordSuccess(ctx, event)
		return
	}
	s.alertTracker.RecordFailure(ctx, event)
}

// ResolveProviderStatuses 遍历各场景当前生效配置，给出每个 Provider 的启用/在生效路由状态。
// 由 aialert.Service 通过 ProviderStatusResolver 接口调用，避免包级循环依赖。
func (s *Service) ResolveProviderStatuses(ctx context.Context) (map[string]aialert.ProviderRuntimeStatus, error) {
	result := make(map[string]aialert.ProviderRuntimeStatus)
	if s == nil {
		return result, nil
	}
	for _, scene := range AllScenes() {
		config, err := s.GetScene(ctx, scene)
		if err != nil {
			return nil, err
		}
		sceneEffective := config.Enabled && len(enabledProviders(config.Providers)) > 0
		for _, provider := range config.Providers {
			result[provider.ID] = aialert.ProviderRuntimeStatus{
				Enabled:          provider.Enabled,
				InEffectiveRoute: sceneEffective && provider.Enabled,
				Scene:            string(scene),
				ProviderName:     provider.Name,
				Model:            provider.Model,
			}
		}
	}
	return result, nil
}

// RetestProvider 定位 Provider 所属场景，构造单节点配置执行一次真实复测。
// 使用 route_test 输入以跳过自动告警追踪，由 aialert 显式落状态。
func (s *Service) RetestProvider(ctx context.Context, providerID string) (aialert.ProviderRetestOutcome, bool, error) {
	providerID = strings.TrimSpace(providerID)
	if s == nil || providerID == "" {
		return aialert.ProviderRetestOutcome{}, false, nil
	}
	for _, scene := range AllScenes() {
		config, err := s.GetScene(ctx, scene)
		if err != nil {
			return aialert.ProviderRetestOutcome{}, false, err
		}
		for _, provider := range config.Providers {
			if provider.ID != providerID {
				continue
			}
			single := config
			single.Enabled = true
			forced := provider
			forced.Enabled = true
			single.Providers = []ProviderConfig{forced}
			single.MaxAttempts = 1
			// 复测是显式动作，清掉熔断避免被跳过。
			s.breaker.markSuccess(scene, providerID)

			input := s.sceneTestInput(scene)
			input.ContentKind = "route_test"
			result, routeErr := s.routeChat(ctx, single, input)

			outcome := aialert.ProviderRetestOutcome{
				OK:        routeErr == nil,
				Model:     firstNonEmpty(result.Model, provider.Model),
				RequestID: common.RequestID(ctx),
			}
			if routeErr == nil {
				outcome.Message = "复测成功"
				if n := len(result.Attempts); n > 0 {
					outcome.HTTPStatus = result.Attempts[n-1].HTTPStatus
				}
				return outcome, true, nil
			}
			outcome.Message = routeErr.Error()
			outcome.ErrorType = routeErrorType(routeErr)
			outcome.ErrorMessage = errorMessage(routeErr)
			if n := len(result.Attempts); n > 0 {
				last := result.Attempts[n-1]
				outcome.HTTPStatus = last.HTTPStatus
				if strings.TrimSpace(last.ErrorType) != "" {
					outcome.ErrorType = last.ErrorType
				}
				if strings.TrimSpace(last.ErrorMessage) != "" {
					outcome.ErrorMessage = last.ErrorMessage
				}
			}
			return outcome, true, nil
		}
	}
	return aialert.ProviderRetestOutcome{}, false, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sceneUsesCompatibility(config SceneConfig) bool {
	return !config.Enabled || len(enabledProviders(config.Providers)) == 0
}

func buildSceneTestInput(scene Scene) ChatCompletionInput {
	switch scene {
	case SceneTitle:
		return ChatCompletionInput{
			Messages: []ChatMessage{
				{
					Role:    "system",
					Content: "你是一个菜谱标题清洗助手。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\"}。",
				},
				{
					Role:    "user",
					Content: "请只返回一个 JSON，title 填写“西红柿炒鸡蛋”。",
				},
			},
			MaxTokens:       intPtr(64),
			ContentKind:     "route_test",
			ValidateContent: validateTitleTestContent,
		}
	case SceneFlowchart:
		return ChatCompletionInput{
			Messages: []ChatMessage{
				{
					Role:    "system",
					Content: "你是一个流程图生成测试助手。请生成一张最小可用的测试流程图，允许返回图片 URL、markdown 图片或 data url，不要输出额外解释。",
				},
				{
					Role:    "user",
					Content: "请生成一张“西红柿炒鸡蛋”测试流程图，内容尽量简单，只要能验证出图链路即可。",
				},
			},
			MaxTokens:       intPtr(256),
			ContentKind:     "route_test",
			ValidateContent: validateFlowchartTestContent,
		}
	default:
		return ChatCompletionInput{
			Messages: []ChatMessage{
				{
					Role:    "system",
					Content: "你是一个菜谱整理助手。必须只返回 JSON，不要输出额外说明。JSON 结构必须是 {\"title\":\"\",\"ingredient\":\"\",\"summary\":\"\",\"mainIngredients\":[],\"secondaryIngredients\":[],\"steps\":[{\"title\":\"\",\"detail\":\"\"}],\"note\":\"\"}。",
				},
				{
					Role:    "user",
					Content: "请返回一个最小可用的测试菜谱 JSON，主题是西红柿炒鸡蛋。",
				},
			},
			MaxTokens:       intPtr(1024),
			ContentKind:     "route_test",
			ValidateContent: validateSummaryTestContent,
		}
	}
}

type orderedProvider struct {
	ProviderConfig
	originalIndex int
}

func buildAttemptOrder(scene Scene, strategy Strategy, providers []ProviderConfig, start int) []orderedProvider {
	ordered := make([]orderedProvider, 0, len(providers))
	for index, provider := range providers {
		ordered = append(ordered, orderedProvider{
			ProviderConfig: provider,
			originalIndex:  index,
		})
	}
	if strategy != StrategyRoundRobinFailover || len(ordered) == 0 {
		return ordered
	}
	if start < 0 {
		start = 0
	}
	start = start % len(ordered)
	return append(ordered[start:], ordered[:start]...)
}

func (s *Service) currentRoundRobinStart(scene Scene, count int) int {
	if count <= 0 {
		return 0
	}
	s.roundRobinMu.Lock()
	defer s.roundRobinMu.Unlock()
	return s.roundRobinNext[scene] % count
}

func (s *Service) setRoundRobinNext(scene Scene, next int, count int) {
	if count <= 0 {
		return
	}
	s.roundRobinMu.Lock()
	defer s.roundRobinMu.Unlock()
	s.roundRobinNext[scene] = next % count
}

func (s *Service) resetRoundRobin(scene Scene) {
	s.roundRobinMu.Lock()
	defer s.roundRobinMu.Unlock()
	delete(s.roundRobinNext, scene)
}
