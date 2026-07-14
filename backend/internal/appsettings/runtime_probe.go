package appsettings

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/airouter"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func (p *RuntimeProvider) TestRuntimeGroup(ctx context.Context, subject, requestID, groupName string, values map[string]any, clearKeys []string) (GroupTestResult, error) {
	group, ok := p.groupIndex[groupName]
	if !ok {
		return GroupTestResult{}, common.ErrNotFound
	}
	settings, err := p.loadSettings(ctx)
	if err != nil {
		return GroupTestResult{}, err
	}

	clearSet := buildEffectiveClearSet(clearKeys, values)

	resolved := make(map[string]string, len(group.Fields))
	for _, field := range group.Fields {
		fullKey := field.Group + "." + field.Key
		if _, ok := clearSet[field.Key]; ok {
			resolved[field.Key] = ""
			continue
		}
		if rawValue, exists := values[field.Key]; exists && !isEmptyStringValue(rawValue) {
			normalized, err := normalizeRuntimeValue(rawValue, field.ValueType)
			if err != nil {
				return GroupTestResult{}, err
			}
			resolved[field.Key] = normalized
			continue
		}
		resolved[field.Key] = p.resolveFieldValue(settings[fullKey], field)
	}

	result := GroupTestResult{
		OK:      false,
		Message: "当前配置无法完成测试",
	}
	startedAt := time.Now()

	switch groupName {
	case "ai.summary", "ai.title":
		timeoutSeconds, _ := strconv.Atoi(strings.TrimSpace(resolved["timeout_seconds"]))
		if timeoutSeconds <= 0 {
			timeoutSeconds = 10
		}
		result = testOpenAICompatible(ctx, resolved["base_url"], resolved["api_key"], resolved["model"], time.Duration(timeoutSeconds)*time.Second)
	case "ai.flowchart":
		timeoutSeconds, _ := strconv.Atoi(strings.TrimSpace(resolved["timeout_seconds"]))
		if timeoutSeconds <= 0 {
			timeoutSeconds = 10
		}
		result = testFlowchartCompatible(
			ctx,
			resolved["base_url"],
			resolved["api_key"],
			resolved["model"],
			resolved["endpoint_mode"],
			resolved["response_format"],
			time.Duration(timeoutSeconds)*time.Second,
		)
	case "sidecar.linkparse":
		timeoutSeconds, _ := strconv.Atoi(strings.TrimSpace(resolved["timeout_seconds"]))
		if timeoutSeconds <= 0 {
			timeoutSeconds = 10
		}
		result = testSidecarHealth(ctx, resolved["base_url"], resolved["api_key"], time.Duration(timeoutSeconds)*time.Second)
	case "ai.provider_alert":
		alertConfig := aialert.Config{
			Enabled:          strings.EqualFold(strings.TrimSpace(resolved["enabled"]), "true"),
			FailureThreshold: parseRuntimeInt(resolved["failure_threshold"], 3),
			SMTPHost:         resolved["smtp_host"],
			SMTPPort:         parseRuntimeInt(resolved["smtp_port"], 587),
			SMTPUsername:     resolved["smtp_username"],
			SMTPPassword:     resolved["smtp_password"],
			FromEmail:        resolved["from_email"],
			ToEmails:         resolved["to_emails"],
		}
		sender := p.alertSender
		if sender == nil {
			sender = aialert.NewSMTPSender()
		}
		subject, body := aialert.BuildTestMessage()
		err := sender.Send(ctx, aialert.SendRequest{
			Config:  alertConfig,
			Subject: subject,
			Body:    body,
		})
		result = GroupTestResult{
			OK:      err == nil,
			Message: "测试邮件已发送，请检查收件箱和垃圾箱",
		}
		if err != nil {
			result.Message = err.Error()
		}
	default:
		return GroupTestResult{}, common.ErrNotFound
	}
	result.LatencyMS = time.Since(startedAt).Milliseconds()

	auditValue := ""
	keys := make([]string, 0, len(resolved))
	for key := range resolved {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		field := p.fieldIndex[groupName+"."+key]
		if auditValue != "" {
			auditValue += ", "
		}
		auditValue += key + "=" + p.maskValue(resolved[key], field.IsSecret)
	}
	_ = p.repo.InsertSettingAudit(ctx, settingAuditRecord{
		GroupName:       groupName,
		SettingKey:      "__test__",
		Action:          "test",
		OldValueMasked:  "",
		NewValueMasked:  truncateRuntimeMessage(auditValue, 240),
		OperatorSubject: strings.TrimSpace(subject),
		RequestID:       strings.TrimSpace(requestID),
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	})

	return result, nil
}

func testOpenAICompatible(ctx context.Context, baseURL, apiKey, model string, timeout time.Duration) GroupTestResult {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	model = strings.TrimSpace(model)
	if baseURL == "" || model == "" {
		return GroupTestResult{OK: false, Message: "缺少 base_url 或 model，无法测试。"}
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	body, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"max_tokens": 1,
		"stream":     false,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return GroupTestResult{OK: false, Message: "创建测试请求失败: " + err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return GroupTestResult{OK: false, Message: "请求失败: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = "状态码 " + strconv.Itoa(resp.StatusCode)
		}
		return GroupTestResult{OK: false, Message: "测试失败: " + message}
	}

	return GroupTestResult{OK: true, Message: "连接成功"}
}

func testFlowchartCompatible(ctx context.Context, baseURL, apiKey, model, endpointMode, responseFormat string, timeout time.Duration) GroupTestResult {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	model = strings.TrimSpace(model)
	if baseURL == "" || model == "" {
		return GroupTestResult{OK: false, Message: "缺少 base_url 或 model，无法测试。"}
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	path := "/chat/completions"
	body := []byte{}
	switch strings.ToLower(strings.TrimSpace(endpointMode)) {
	case "", "chat", "chat_completions", "chat/completions":
		body, _ = json.Marshal(map[string]any{
			"model": model,
			"messages": []map[string]string{
				{"role": "user", "content": "ping"},
			},
			"max_tokens": 1,
			"stream":     false,
		})
	case "images", "images_generations", "images/generations":
		path = "/images/generations"
		payload := map[string]any{
			"model":         model,
			"prompt":        "请生成一张最简单的测试流程图图片，只用于验证链路。",
			"output_format": "png",
		}
		switch strings.ToLower(strings.TrimSpace(responseFormat)) {
		case "", "auto":
		case "image_url", "image-url", "url":
			if !airouter.IsGPTImageModel(model) {
				payload["response_format"] = "image_url"
			}
		case "b64_json", "b64-json", "base64":
			if !airouter.IsGPTImageModel(model) {
				payload["response_format"] = "b64_json"
			}
		default:
			return GroupTestResult{OK: false, Message: "response_format 非法，应为 auto / image_url / b64_json"}
		}
		body, _ = json.Marshal(payload)
	default:
		return GroupTestResult{OK: false, Message: "endpoint_mode 非法，应为 chat_completions 或 images_generations"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return GroupTestResult{OK: false, Message: "创建测试请求失败: " + err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return GroupTestResult{OK: false, Message: "请求失败: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = "状态码 " + strconv.Itoa(resp.StatusCode)
		}
		return GroupTestResult{OK: false, Message: "测试失败: " + message}
	}

	return GroupTestResult{OK: true, Message: "连接成功"}
}

func testSidecarHealth(ctx context.Context, baseURL, apiKey string, timeout time.Duration) GroupTestResult {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return GroupTestResult{OK: false, Message: "缺少 sidecar base_url，无法测试。"}
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/health", nil)
	if err != nil {
		return GroupTestResult{OK: false, Message: "创建测试请求失败: " + err.Error()}
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return GroupTestResult{OK: false, Message: "请求失败: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = "状态码 " + strconv.Itoa(resp.StatusCode)
		}
		return GroupTestResult{OK: false, Message: "sidecar 健康检查失败: " + message}
	}

	return GroupTestResult{OK: true, Message: "sidecar 健康检查通过"}
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func truncateRuntimeMessage(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len([]rune(value)) <= limit {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit])) + "..."
}
