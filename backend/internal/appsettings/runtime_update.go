package appsettings

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

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

func (p *RuntimeProvider) ListSettingAudits(ctx context.Context, filter SettingAuditFilter) (SettingAuditList, error) {
	filter.Page, filter.PageSize = normalizeAuditPagination(filter.Page, filter.PageSize)
	whereParts := make([]string, 0, 6)
	args := make([]any, 0, 6)
	if value := strings.TrimSpace(filter.GroupName); value != "" {
		whereParts = append(whereParts, "group_name = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.Action); value != "" {
		whereParts = append(whereParts, "action = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.OperatorSubject); value != "" {
		whereParts = append(whereParts, "operator_subject LIKE ?")
		args = append(args, "%"+value+"%")
	}
	if value := strings.TrimSpace(filter.SettingKey); value != "" {
		whereParts = append(whereParts, "setting_key LIKE ?")
		args = append(args, "%"+value+"%")
	}
	if value := strings.TrimSpace(filter.TimeFrom); value != "" {
		whereParts = append(whereParts, "created_at >= ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.TimeTo); value != "" {
		whereParts = append(whereParts, "created_at <= ?")
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
