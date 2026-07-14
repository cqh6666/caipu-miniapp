package appsettings

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
)

func buildRuntimeGroups(cfg config.Config) []runtimeGroupDefinition {
	return []runtimeGroupDefinition{
		{
			Name:            "ai.summary",
			Title:           "AI 总结",
			Description:     "自动解析里的菜谱总结调用配置。",
			HiddenFromAdmin: true,
			Fields: []runtimeFieldDefinition{
				{Group: "ai.summary", Key: "base_url", Label: "Base URL", Description: "OpenAI-compatible 接口地址。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIBaseURL)},
				{Group: "ai.summary", Key: "api_key", Label: "API Key", Description: "AI 总结使用的密钥。", ValueType: "string", IsSecret: true, DefaultValue: strings.TrimSpace(cfg.AIAPIKey)},
				{Group: "ai.summary", Key: "model", Label: "Model", Description: "自动解析总结使用的模型。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIModel)},
				{Group: "ai.summary", Key: "timeout_seconds", Label: "Timeout", Description: "请求超时时间（秒）。", ValueType: "int", DefaultValue: strconv.Itoa(cfg.AITimeoutSeconds)},
			},
		},
		{
			Name:            "ai.flowchart",
			Title:           "流程图生成",
			Description:     "步骤图生成调用配置。",
			HiddenFromAdmin: true,
			Fields: []runtimeFieldDefinition{
				{Group: "ai.flowchart", Key: "base_url", Label: "Base URL", Description: "流程图模型接口地址。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIFlowchartBaseURL)},
				{Group: "ai.flowchart", Key: "api_key", Label: "API Key", Description: "流程图模型密钥。", ValueType: "string", IsSecret: true, DefaultValue: strings.TrimSpace(cfg.AIFlowchartAPIKey)},
				{Group: "ai.flowchart", Key: "model", Label: "Model", Description: "流程图生成模型。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIFlowchartModel)},
				{Group: "ai.flowchart", Key: "endpoint_mode", Label: "Endpoint Mode", Description: "流程图节点请求路径：chat_completions 或 images_generations。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIFlowchartEndpointMode)},
				{Group: "ai.flowchart", Key: "response_format", Label: "Response Format", Description: "images_generations 响应偏好：auto / image_url / b64_json；GPT image 模型默认 b64_json，不随请求发送该字段。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIFlowchartResponseFormat)},
				{Group: "ai.flowchart", Key: "timeout_seconds", Label: "Timeout", Description: "请求超时时间（秒）。", ValueType: "int", DefaultValue: strconv.Itoa(cfg.AIFlowchartTimeoutSeconds)},
			},
		},
		{
			Name:            "ai.title",
			Title:           "标题精修",
			Description:     "链接预览里的 AI 标题清洗配置。",
			HiddenFromAdmin: true,
			Fields: []runtimeFieldDefinition{
				{Group: "ai.title", Key: "enabled", Label: "Enabled", Description: "是否启用 AI 标题精修。", ValueType: "bool", DefaultValue: strconv.FormatBool(cfg.AITitleEnabled)},
				{Group: "ai.title", Key: "base_url", Label: "Base URL", Description: "标题精修接口地址。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AITitleBaseURL)},
				{Group: "ai.title", Key: "api_key", Label: "API Key", Description: "标题精修密钥。", ValueType: "string", IsSecret: true, DefaultValue: strings.TrimSpace(cfg.AITitleAPIKey)},
				{Group: "ai.title", Key: "model", Label: "Model", Description: "标题精修模型。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AITitleModel)},
				{Group: "ai.title", Key: "stream", Label: "Stream", Description: "是否使用流式响应。", ValueType: "bool", DefaultValue: strconv.FormatBool(cfg.AITitleStream)},
				{Group: "ai.title", Key: "temperature", Label: "Temperature", Description: "标题精修温度参数。", ValueType: "float", DefaultValue: strconv.FormatFloat(cfg.AITitleTemperature, 'f', -1, 64)},
				{Group: "ai.title", Key: "max_tokens", Label: "Max Tokens", Description: "标题精修最大输出 token。", ValueType: "int", DefaultValue: strconv.Itoa(cfg.AITitleMaxTokens)},
				{Group: "ai.title", Key: "timeout_seconds", Label: "Timeout", Description: "请求超时时间（秒）。", ValueType: "int", DefaultValue: strconv.Itoa(cfg.AITitleTimeoutSeconds)},
			},
		},
		{
			Name:        "ai.provider_alert",
			Title:       "AI Provider 告警",
			Description: "按 provider 连续异常次数发送邮件告警，默认适配 QQ 邮箱 SMTP。",
			Fields: []runtimeFieldDefinition{
				{Group: "ai.provider_alert", Key: "enabled", Label: "Enabled", Description: "是否启用连续异常邮件告警。", ValueType: "bool", DefaultValue: strconv.FormatBool(cfg.AIAlertEnabled)},
				{Group: "ai.provider_alert", Key: "failure_threshold", Label: "Failure Threshold", Description: "同一 Provider 连续异常达到该次数后触发一次告警。", ValueType: "int", DefaultValue: strconv.Itoa(cfg.AIAlertFailureThreshold)},
				{Group: "ai.provider_alert", Key: "active_window_hours", Label: "Active Window Hours", Description: "最后失败超过该小时数后，告警自动从红色降级为黄色“待复测（已过期）”。默认 72。", ValueType: "int", DefaultValue: strconv.Itoa(72)},
				{Group: "ai.provider_alert", Key: "smtp_host", Label: "SMTP Host", Description: "SMTP 主机，QQ 邮箱默认 smtp.qq.com。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIAlertSMTPHost)},
				{Group: "ai.provider_alert", Key: "smtp_port", Label: "SMTP Port", Description: "SMTP 端口，推荐 587（STARTTLS）或 465（SSL）。", ValueType: "int", DefaultValue: strconv.Itoa(cfg.AIAlertSMTPPort)},
				{Group: "ai.provider_alert", Key: "smtp_username", Label: "SMTP Username", Description: "发件邮箱账号。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIAlertSMTPUsername)},
				{Group: "ai.provider_alert", Key: "smtp_password", Label: "SMTP Password", Description: "SMTP 授权码，不是邮箱登录密码。", ValueType: "string", IsSecret: true, DefaultValue: strings.TrimSpace(cfg.AIAlertSMTPPassword)},
				{Group: "ai.provider_alert", Key: "from_email", Label: "From Email", Description: "发件邮箱，留空时回退到 SMTP Username。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIAlertFromEmail)},
				{Group: "ai.provider_alert", Key: "to_emails", Label: "To Emails", Description: "收件邮箱，支持多个，逗号分隔。", ValueType: "string", DefaultValue: strings.Join(cfg.AIAlertToEmails, ",")},
			},
		},
		{
			Name:        "sidecar.linkparse",
			Title:       "Linkparse Sidecar",
			Description: "小红书 / B 站 sidecar 调用配置。",
			Fields: []runtimeFieldDefinition{
				{Group: "sidecar.linkparse", Key: "enabled", Label: "Enabled", Description: "是否启用 sidecar。", ValueType: "bool", DefaultValue: strconv.FormatBool(cfg.LinkparseSidecarEnabled)},
				{Group: "sidecar.linkparse", Key: "base_url", Label: "Base URL", Description: "sidecar 服务地址。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.LinkparseSidecarBaseURL)},
				{Group: "sidecar.linkparse", Key: "api_key", Label: "API Key", Description: "sidecar 内部认证密钥。", ValueType: "string", IsSecret: true, DefaultValue: strings.TrimSpace(cfg.LinkparseSidecarAPIKey)},
				{Group: "sidecar.linkparse", Key: "timeout_seconds", Label: "Timeout", Description: "sidecar 请求超时时间（秒）。", ValueType: "int", DefaultValue: strconv.Itoa(cfg.LinkparseSidecarTimeoutSec)},
			},
		},
		{
			Name:        "miniapp.features",
			Title:       "小程序功能开关",
			Description: "控制小程序端入口级功能的展示与点击行为，保存后小程序下次拉取配置生效。",
			Fields: []runtimeFieldDefinition{
				{
					Group:        "miniapp.features",
					Key:          "diet_assistant_enabled",
					Label:        "AI 助手入口",
					Description:  "开启后首页底部中间按钮打开饮食管家；关闭后同一按钮打开添加菜谱弹层。",
					ValueType:    "bool",
					DefaultValue: "false",
				},
			},
		},
	}
}

func normalizeRuntimeValue(value any, valueType string) (string, error) {
	switch strings.TrimSpace(valueType) {
	case "bool":
		switch typed := value.(type) {
		case bool:
			return strconv.FormatBool(typed), nil
		case string:
			typed = strings.TrimSpace(typed)
			if typed == "" {
				return "", common.NewAppError(common.CodeBadRequest, "bool value is required", http.StatusBadRequest)
			}
			parsed, err := strconv.ParseBool(typed)
			if err != nil {
				return "", common.NewAppError(common.CodeBadRequest, "invalid bool value", http.StatusBadRequest).WithErr(err)
			}
			return strconv.FormatBool(parsed), nil
		default:
			return "", common.NewAppError(common.CodeBadRequest, "invalid bool value", http.StatusBadRequest)
		}
	case "int":
		switch typed := value.(type) {
		case float64:
			return strconv.Itoa(int(typed)), nil
		case int:
			return strconv.Itoa(typed), nil
		case string:
			typed = strings.TrimSpace(typed)
			if typed == "" {
				return "", common.NewAppError(common.CodeBadRequest, "int value is required", http.StatusBadRequest)
			}
			parsed, err := strconv.Atoi(typed)
			if err != nil {
				return "", common.NewAppError(common.CodeBadRequest, "invalid int value", http.StatusBadRequest).WithErr(err)
			}
			return strconv.Itoa(parsed), nil
		default:
			return "", common.NewAppError(common.CodeBadRequest, "invalid int value", http.StatusBadRequest)
		}
	case "float":
		switch typed := value.(type) {
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64), nil
		case string:
			typed = strings.TrimSpace(typed)
			if typed == "" {
				return "", common.NewAppError(common.CodeBadRequest, "float value is required", http.StatusBadRequest)
			}
			parsed, err := strconv.ParseFloat(typed, 64)
			if err != nil {
				return "", common.NewAppError(common.CodeBadRequest, "invalid float value", http.StatusBadRequest).WithErr(err)
			}
			return strconv.FormatFloat(parsed, 'f', -1, 64), nil
		default:
			return "", common.NewAppError(common.CodeBadRequest, "invalid float value", http.StatusBadRequest)
		}
	default:
		return strings.TrimSpace(stringifyRuntimeValue(value)), nil
	}
}

func parseRuntimeInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	if parsed <= 0 {
		return fallback
	}
	return parsed
}

func stringifyRuntimeValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(data)
	}
}

func isEmptyStringValue(value any) bool {
	raw, ok := value.(string)
	return ok && strings.TrimSpace(raw) == ""
}

func buildEffectiveClearSet(clearKeys []string, values map[string]any) map[string]struct{} {
	clearSet := make(map[string]struct{}, len(clearKeys))
	for _, item := range clearKeys {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		rawValue, exists := values[key]
		if exists && !isEmptyStringValue(rawValue) && rawValue != nil {
			continue
		}
		clearSet[key] = struct{}{}
	}
	return clearSet
}

func sourceFromDefault(value string, valueType string) string {
	if valueType != "string" {
		return "env"
	}
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return "env"
}

func normalizeAuditPagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
