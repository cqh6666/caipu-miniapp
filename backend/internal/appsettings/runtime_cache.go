package appsettings

import (
	"context"
	"strconv"
	"strings"
	"time"
)

const runtimeCacheTTL = 15 * time.Second

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

func (p *RuntimeProvider) loadSettingsFresh(ctx context.Context) (map[string]runtimeSettingRecord, error) {
	records, err := p.repo.ListRuntimeSettings(ctx)
	if err != nil {
		return nil, err
	}
	settings := make(map[string]runtimeSettingRecord, len(records))
	for _, record := range records {
		settings[record.Key] = record
	}
	return settings, nil
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
