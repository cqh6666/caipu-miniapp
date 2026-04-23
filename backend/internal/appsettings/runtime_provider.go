package appsettings

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/aialert"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
)

const runtimeCacheTTL = 15 * time.Second

type runtimeFieldDefinition struct {
	Group             string
	Key               string
	Label             string
	Description       string
	ValueType         string
	IsSecret          bool
	IsRestartRequired bool
	DefaultValue      string
}

type runtimeGroupDefinition struct {
	Name        string
	Title       string
	Description string
	Fields      []runtimeFieldDefinition
}

type RuntimeProvider struct {
	repo             *Repository
	cipherBox        *cipherBox
	groups           []runtimeGroupDefinition
	groupIndex       map[string]runtimeGroupDefinition
	fieldIndex       map[string]runtimeFieldDefinition
	mu               sync.RWMutex
	cachedAt         time.Time
	cachedSettings   map[string]runtimeSettingRecord
	bilibiliVerifier func(context.Context, string) error
	alertSender      aialert.Sender
}

func NewRuntimeProvider(repo *Repository, secret string, cfg config.Config) *RuntimeProvider {
	groups := buildRuntimeGroups(cfg)
	groupIndex := make(map[string]runtimeGroupDefinition, len(groups))
	fieldIndex := make(map[string]runtimeFieldDefinition)
	for _, group := range groups {
		groupIndex[group.Name] = group
		for _, field := range group.Fields {
			fieldIndex[field.Group+"."+field.Key] = field
		}
	}

	return &RuntimeProvider{
		repo:           repo,
		cipherBox:      newCipherBox(secret),
		groups:         groups,
		groupIndex:     groupIndex,
		fieldIndex:     fieldIndex,
		cachedSettings: make(map[string]runtimeSettingRecord),
	}
}

func (p *RuntimeProvider) SetBilibiliVerifier(verify func(context.Context, string) error) {
	p.bilibiliVerifier = verify
}

func (p *RuntimeProvider) SetAIAlertSender(sender aialert.Sender) {
	p.alertSender = sender
}

func (p *RuntimeProvider) SummaryAI(ctx context.Context) SummaryAIConfig {
	return SummaryAIConfig{
		BaseURL: p.getString(ctx, "ai.summary.base_url"),
		APIKey:  p.getString(ctx, "ai.summary.api_key"),
		Model:   p.getString(ctx, "ai.summary.model"),
		Timeout: time.Duration(p.getInt(ctx, "ai.summary.timeout_seconds", 30)) * time.Second,
	}
}

func (p *RuntimeProvider) FlowchartAI(ctx context.Context) FlowchartAIConfig {
	return FlowchartAIConfig{
		BaseURL:        p.getString(ctx, "ai.flowchart.base_url"),
		APIKey:         p.getString(ctx, "ai.flowchart.api_key"),
		Model:          p.getString(ctx, "ai.flowchart.model"),
		EndpointMode:   p.getString(ctx, "ai.flowchart.endpoint_mode"),
		ResponseFormat: p.getString(ctx, "ai.flowchart.response_format"),
		Timeout:        time.Duration(p.getInt(ctx, "ai.flowchart.timeout_seconds", 45)) * time.Second,
	}
}

func (p *RuntimeProvider) TitleAI(ctx context.Context) TitleAIConfig {
	return TitleAIConfig{
		Enabled:     p.getBool(ctx, "ai.title.enabled", false),
		BaseURL:     p.getString(ctx, "ai.title.base_url"),
		APIKey:      p.getString(ctx, "ai.title.api_key"),
		Model:       p.getString(ctx, "ai.title.model"),
		Stream:      p.getBool(ctx, "ai.title.stream", false),
		Temperature: p.getFloat(ctx, "ai.title.temperature", 0),
		MaxTokens:   p.getInt(ctx, "ai.title.max_tokens", 64),
		Timeout:     time.Duration(p.getInt(ctx, "ai.title.timeout_seconds", 3)) * time.Second,
	}
}

func (p *RuntimeProvider) AIProviderAlert(ctx context.Context) aialert.Config {
	return aialert.Config{
		Enabled:          p.getBool(ctx, "ai.provider_alert.enabled", false),
		FailureThreshold: p.getInt(ctx, "ai.provider_alert.failure_threshold", 3),
		SMTPHost:         p.getString(ctx, "ai.provider_alert.smtp_host"),
		SMTPPort:         p.getInt(ctx, "ai.provider_alert.smtp_port", 587),
		SMTPUsername:     p.getString(ctx, "ai.provider_alert.smtp_username"),
		SMTPPassword:     p.getString(ctx, "ai.provider_alert.smtp_password"),
		FromEmail:        p.getString(ctx, "ai.provider_alert.from_email"),
		ToEmails:         p.getString(ctx, "ai.provider_alert.to_emails"),
	}
}

func (p *RuntimeProvider) LinkparseSidecar(ctx context.Context) LinkparseSidecarConfig {
	return LinkparseSidecarConfig{
		Enabled: p.getBool(ctx, "sidecar.linkparse.enabled", false),
		BaseURL: p.getString(ctx, "sidecar.linkparse.base_url"),
		APIKey:  p.getString(ctx, "sidecar.linkparse.api_key"),
		Timeout: time.Duration(p.getInt(ctx, "sidecar.linkparse.timeout_seconds", 150)) * time.Second,
	}
}

func (p *RuntimeProvider) ListRuntimeGroups(ctx context.Context) ([]RuntimeSettingGroupView, error) {
	settings, err := p.loadSettings(ctx)
	if err != nil {
		return nil, err
	}

	groups := make([]RuntimeSettingGroupView, 0, len(p.groups))
	for _, group := range p.groups {
		view := RuntimeSettingGroupView{
			Name:        group.Name,
			Title:       group.Title,
			Description: group.Description,
			Fields:      make([]RuntimeSettingFieldView, 0, len(group.Fields)),
		}
		for _, field := range group.Fields {
			view.Fields = append(view.Fields, p.buildFieldView(field, settings))
		}
		groups = append(groups, view)
	}
	return groups, nil
}

func (p *RuntimeProvider) GetRuntimeGroup(ctx context.Context, groupName string) (RuntimeSettingGroupView, error) {
	groups, err := p.ListRuntimeGroups(ctx)
	if err != nil {
		return RuntimeSettingGroupView{}, err
	}
	for _, group := range groups {
		if group.Name == groupName {
			return group, nil
		}
	}
	return RuntimeSettingGroupView{}, common.ErrNotFound
}

func (p *RuntimeProvider) UpdateRuntimeGroup(ctx context.Context, subject, requestID, groupName string, values map[string]any, clearKeys []string) (RuntimeSettingGroupView, error) {
	group, ok := p.groupIndex[groupName]
	if !ok {
		return RuntimeSettingGroupView{}, common.ErrNotFound
	}
	settings, err := p.loadSettings(ctx)
	if err != nil {
		return RuntimeSettingGroupView{}, err
	}

	tx, err := p.repo.db.BeginTx(ctx, nil)
	if err != nil {
		return RuntimeSettingGroupView{}, err
	}

	clearSet := buildEffectiveClearSet(clearKeys, values)

	for _, field := range group.Fields {
		fullKey := field.Group + "." + field.Key
		_, shouldClear := clearSet[field.Key]
		rawValue, exists := values[field.Key]
		if !exists && !shouldClear {
			continue
		}

		current := settings[fullKey]
		oldMasked := p.maskRecordValue(current, field)

		if shouldClear || isEmptyStringValue(rawValue) {
			if _, err := tx.ExecContext(ctx, `DELETE FROM app_runtime_settings WHERE key = ?`, fullKey); err != nil {
				_ = tx.Rollback()
				return RuntimeSettingGroupView{}, err
			}
			if err := p.insertAuditTx(ctx, tx, settingAuditRecord{
				GroupName:       group.Name,
				SettingKey:      fullKey,
				Action:          "update",
				OldValueMasked:  oldMasked,
				NewValueMasked:  "",
				OperatorSubject: subject,
				RequestID:       requestID,
				CreatedAt:       time.Now().UTC().Format(time.RFC3339),
			}); err != nil {
				_ = tx.Rollback()
				return RuntimeSettingGroupView{}, err
			}
			delete(settings, fullKey)
			continue
		}

		normalized, err := normalizeRuntimeValue(rawValue, field.ValueType)
		if err != nil {
			_ = tx.Rollback()
			return RuntimeSettingGroupView{}, err
		}

		record := runtimeSettingRecord{
			Key:               fullKey,
			GroupName:         group.Name,
			ValueType:         field.ValueType,
			IsSecret:          field.IsSecret,
			IsRestartRequired: field.IsRestartRequired,
			Description:       field.Description,
			UpdatedBySubject:  strings.TrimSpace(subject),
			UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
		}
		if field.IsSecret {
			ciphertext, err := p.cipherBox.Encrypt(normalized)
			if err != nil {
				_ = tx.Rollback()
				return RuntimeSettingGroupView{}, common.ErrInternal.WithErr(err)
			}
			record.ValueCiphertext = ciphertext
		} else {
			record.ValueText = normalized
		}

		if err := p.upsertRuntimeSettingTx(ctx, tx, record); err != nil {
			_ = tx.Rollback()
			return RuntimeSettingGroupView{}, err
		}
		if err := p.insertAuditTx(ctx, tx, settingAuditRecord{
			GroupName:       group.Name,
			SettingKey:      fullKey,
			Action:          "update",
			OldValueMasked:  oldMasked,
			NewValueMasked:  p.maskValue(normalized, field.IsSecret),
			OperatorSubject: subject,
			RequestID:       requestID,
			CreatedAt:       record.UpdatedAt,
		}); err != nil {
			_ = tx.Rollback()
			return RuntimeSettingGroupView{}, err
		}
		settings[fullKey] = record
	}

	if err := tx.Commit(); err != nil {
		return RuntimeSettingGroupView{}, err
	}

	p.Invalidate()
	return p.GetRuntimeGroup(ctx, groupName)
}

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

func (p *RuntimeProvider) ListSettingAudits(ctx context.Context, filter SettingAuditFilter) (SettingAuditList, error) {
	filter.Page, filter.PageSize = normalizeAuditPagination(filter.Page, filter.PageSize)
	whereParts := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if value := strings.TrimSpace(filter.GroupName); value != "" {
		whereParts = append(whereParts, "group_name = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.Action); value != "" {
		whereParts = append(whereParts, "action = ?")
		args = append(args, value)
	}

	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}

	var total int
	if err := p.repo.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM app_setting_audits"+where, args...).Scan(&total); err != nil {
		return SettingAuditList{}, err
	}

	queryArgs := append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := p.repo.db.QueryContext(ctx, `
SELECT
	id,
	group_name,
	setting_key,
	action,
	old_value_masked,
	new_value_masked,
	operator_subject,
	request_id,
	created_at
FROM app_setting_audits`+where+`
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?
`, queryArgs...)
	if err != nil {
		return SettingAuditList{}, err
	}
	defer rows.Close()

	items := make([]SettingAuditRecord, 0, filter.PageSize)
	for rows.Next() {
		var item SettingAuditRecord
		if err := rows.Scan(
			&item.ID,
			&item.GroupName,
			&item.SettingKey,
			&item.Action,
			&item.OldValueMasked,
			&item.NewValueMasked,
			&item.OperatorSubject,
			&item.RequestID,
			&item.CreatedAt,
		); err != nil {
			return SettingAuditList{}, err
		}
		items = append(items, item)
	}

	return SettingAuditList{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, rows.Err()
}

func (p *RuntimeProvider) Invalidate() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cachedAt = time.Time{}
	p.cachedSettings = make(map[string]runtimeSettingRecord)
}

func (p *RuntimeProvider) loadSettings(ctx context.Context) (map[string]runtimeSettingRecord, error) {
	p.mu.RLock()
	if time.Since(p.cachedAt) < runtimeCacheTTL && !p.cachedAt.IsZero() {
		copied := make(map[string]runtimeSettingRecord, len(p.cachedSettings))
		for key, value := range p.cachedSettings {
			copied[key] = value
		}
		p.mu.RUnlock()
		return copied, nil
	}
	p.mu.RUnlock()

	records, err := p.repo.ListRuntimeSettings(ctx)
	if err != nil {
		return nil, err
	}
	settings := make(map[string]runtimeSettingRecord, len(records))
	for _, record := range records {
		settings[record.Key] = record
	}

	p.mu.Lock()
	p.cachedAt = time.Now()
	p.cachedSettings = settings
	p.mu.Unlock()

	copied := make(map[string]runtimeSettingRecord, len(settings))
	for key, value := range settings {
		copied[key] = value
	}
	return copied, nil
}

func (p *RuntimeProvider) getString(ctx context.Context, key string) string {
	value, _ := p.getValue(ctx, key)
	return value
}

func (p *RuntimeProvider) getInt(ctx context.Context, key string, fallback int) int {
	value, _ := p.getValue(ctx, key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (p *RuntimeProvider) getBool(ctx context.Context, key string, fallback bool) bool {
	value, _ := p.getValue(ctx, key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (p *RuntimeProvider) getFloat(ctx context.Context, key string, fallback float64) float64 {
	value, _ := p.getValue(ctx, key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func (p *RuntimeProvider) getValue(ctx context.Context, key string) (string, string) {
	settings, err := p.loadSettings(ctx)
	if err != nil {
		field, ok := p.fieldIndex[key]
		if !ok {
			return "", "none"
		}
		return field.DefaultValue, sourceFromDefault(field.DefaultValue, field.ValueType)
	}

	field, ok := p.fieldIndex[key]
	if !ok {
		return "", "none"
	}

	record, ok := settings[key]
	if ok {
		value := p.resolveFieldValue(record, field)
		if value != "" || field.ValueType != "string" {
			return value, "db"
		}
	}

	return field.DefaultValue, sourceFromDefault(field.DefaultValue, field.ValueType)
}

func (p *RuntimeProvider) buildFieldView(field runtimeFieldDefinition, settings map[string]runtimeSettingRecord) RuntimeSettingFieldView {
	record := settings[field.Group+"."+field.Key]
	value := p.resolveFieldValue(record, field)
	source := sourceFromDefault(field.DefaultValue, field.ValueType)
	if record.Key != "" && (value != "" || field.ValueType != "string") {
		source = "db"
	}

	view := RuntimeSettingFieldView{
		Key:               field.Key,
		Label:             field.Label,
		Description:       field.Description,
		ValueType:         field.ValueType,
		IsSecret:          field.IsSecret,
		IsRestartRequired: field.IsRestartRequired,
		HasValue:          value != "" || field.ValueType != "string",
		Source:            source,
		UpdatedAt:         record.UpdatedAt,
		UpdatedBySubject:  record.UpdatedBySubject,
	}

	if field.IsSecret {
		view.MaskedValue = p.maskValue(value, true)
		return view
	}

	view.Value = value
	view.MaskedValue = value
	return view
}

func (p *RuntimeProvider) resolveFieldValue(record runtimeSettingRecord, field runtimeFieldDefinition) string {
	if record.Key == "" {
		return field.DefaultValue
	}
	if field.IsSecret {
		if strings.TrimSpace(record.ValueCiphertext) == "" {
			return field.DefaultValue
		}
		value, err := p.cipherBox.Decrypt(record.ValueCiphertext)
		if err != nil {
			return field.DefaultValue
		}
		return strings.TrimSpace(value)
	}
	if strings.TrimSpace(record.ValueText) == "" {
		return field.DefaultValue
	}
	return strings.TrimSpace(record.ValueText)
}

func (p *RuntimeProvider) maskRecordValue(record runtimeSettingRecord, field runtimeFieldDefinition) string {
	return p.maskValue(p.resolveFieldValue(record, field), field.IsSecret)
}

func (p *RuntimeProvider) maskValue(value string, secret bool) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !secret {
		return value
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func (p *RuntimeProvider) upsertRuntimeSettingTx(ctx context.Context, tx *sql.Tx, record runtimeSettingRecord) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO app_runtime_settings (
	key,
	group_name,
	value_text,
	value_ciphertext,
	value_type,
	is_secret,
	is_restart_required,
	description,
	updated_by_subject,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(key) DO UPDATE SET
	group_name = excluded.group_name,
	value_text = excluded.value_text,
	value_ciphertext = excluded.value_ciphertext,
	value_type = excluded.value_type,
	is_secret = excluded.is_secret,
	is_restart_required = excluded.is_restart_required,
	description = excluded.description,
	updated_by_subject = excluded.updated_by_subject,
	updated_at = excluded.updated_at
`,
		record.Key,
		record.GroupName,
		record.ValueText,
		record.ValueCiphertext,
		record.ValueType,
		boolToInt(record.IsSecret),
		boolToInt(record.IsRestartRequired),
		record.Description,
		record.UpdatedBySubject,
		record.UpdatedAt,
	)
	return err
}

func (p *RuntimeProvider) insertAuditTx(ctx context.Context, tx *sql.Tx, record settingAuditRecord) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO app_setting_audits (
	group_name,
	setting_key,
	action,
	old_value_masked,
	new_value_masked,
	operator_subject,
	request_id,
	created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
		record.GroupName,
		record.SettingKey,
		record.Action,
		record.OldValueMasked,
		record.NewValueMasked,
		record.OperatorSubject,
		record.RequestID,
		record.CreatedAt,
	)
	return err
}

func buildRuntimeGroups(cfg config.Config) []runtimeGroupDefinition {
	return []runtimeGroupDefinition{
		{
			Name:        "ai.summary",
			Title:       "AI 总结",
			Description: "自动解析里的菜谱总结调用配置。",
			Fields: []runtimeFieldDefinition{
				{Group: "ai.summary", Key: "base_url", Label: "Base URL", Description: "OpenAI-compatible 接口地址。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIBaseURL)},
				{Group: "ai.summary", Key: "api_key", Label: "API Key", Description: "AI 总结使用的密钥。", ValueType: "string", IsSecret: true, DefaultValue: strings.TrimSpace(cfg.AIAPIKey)},
				{Group: "ai.summary", Key: "model", Label: "Model", Description: "自动解析总结使用的模型。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIModel)},
				{Group: "ai.summary", Key: "timeout_seconds", Label: "Timeout", Description: "请求超时时间（秒）。", ValueType: "int", DefaultValue: strconv.Itoa(cfg.AITimeoutSeconds)},
			},
		},
		{
			Name:        "ai.flowchart",
			Title:       "流程图生成",
			Description: "步骤图生成调用配置。",
			Fields: []runtimeFieldDefinition{
				{Group: "ai.flowchart", Key: "base_url", Label: "Base URL", Description: "流程图模型接口地址。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIFlowchartBaseURL)},
				{Group: "ai.flowchart", Key: "api_key", Label: "API Key", Description: "流程图模型密钥。", ValueType: "string", IsSecret: true, DefaultValue: strings.TrimSpace(cfg.AIFlowchartAPIKey)},
				{Group: "ai.flowchart", Key: "model", Label: "Model", Description: "流程图生成模型。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIFlowchartModel)},
				{Group: "ai.flowchart", Key: "endpoint_mode", Label: "Endpoint Mode", Description: "流程图节点请求路径：chat_completions 或 images_generations。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIFlowchartEndpointMode)},
				{Group: "ai.flowchart", Key: "response_format", Label: "Response Format", Description: "images_generations 返回格式：auto / image_url / b64_json。", ValueType: "string", DefaultValue: strings.TrimSpace(cfg.AIFlowchartResponseFormat)},
				{Group: "ai.flowchart", Key: "timeout_seconds", Label: "Timeout", Description: "请求超时时间（秒）。", ValueType: "int", DefaultValue: strconv.Itoa(cfg.AIFlowchartTimeoutSeconds)},
			},
		},
		{
			Name:        "ai.title",
			Title:       "标题精修",
			Description: "链接预览里的 AI 标题清洗配置。",
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
			"quality":       "high",
			"output_format": "png",
		}
		switch strings.ToLower(strings.TrimSpace(responseFormat)) {
		case "", "auto":
		case "image_url", "image-url", "url":
			payload["response_format"] = "image_url"
		case "b64_json", "b64-json", "base64":
			payload["response_format"] = "b64_json"
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
