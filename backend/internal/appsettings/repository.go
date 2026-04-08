package appsettings

import (
	"context"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetBilibiliSession(ctx context.Context) (bilibiliSessionRecord, error) {
	var record bilibiliSessionRecord
	err := r.db.QueryRowContext(ctx, `
SELECT
	COALESCE(sessdata_ciphertext, ''),
	COALESCE(masked_sessdata, ''),
	COALESCE(status, ''),
	COALESCE(last_checked_at, ''),
	COALESCE(last_success_at, ''),
	COALESCE(last_error, ''),
	updated_by,
	COALESCE(updated_at, '')
FROM app_bilibili_settings
WHERE id = 1
LIMIT 1
`).Scan(
		&record.SessdataCiphertext,
		&record.MaskedSessdata,
		&record.Status,
		&record.LastCheckedAt,
		&record.LastSuccessAt,
		&record.LastError,
		&record.UpdatedBy,
		&record.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return bilibiliSessionRecord{}, nil
	}
	return record, err
}

func (r *Repository) UpsertBilibiliSession(ctx context.Context, record bilibiliSessionRecord) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO app_bilibili_settings (
	id,
	sessdata_ciphertext,
	masked_sessdata,
	status,
	last_checked_at,
	last_success_at,
	last_error,
	updated_by,
	updated_at
) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	sessdata_ciphertext = excluded.sessdata_ciphertext,
	masked_sessdata = excluded.masked_sessdata,
	status = excluded.status,
	last_checked_at = excluded.last_checked_at,
	last_success_at = excluded.last_success_at,
	last_error = excluded.last_error,
	updated_by = excluded.updated_by,
	updated_at = excluded.updated_at
`,
		record.SessdataCiphertext,
		record.MaskedSessdata,
		record.Status,
		nullableText(record.LastCheckedAt),
		nullableText(record.LastSuccessAt),
		record.LastError,
		record.UpdatedBy,
		record.UpdatedAt,
	)
	return err
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *Repository) ListRuntimeSettings(ctx context.Context) ([]runtimeSettingRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT
	key,
	group_name,
	COALESCE(value_text, ''),
	COALESCE(value_ciphertext, ''),
	COALESCE(value_type, 'string'),
	is_secret,
	is_restart_required,
	COALESCE(description, ''),
	COALESCE(updated_by_subject, ''),
	COALESCE(updated_at, '')
FROM app_runtime_settings
ORDER BY group_name ASC, key ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]runtimeSettingRecord, 0, 16)
	for rows.Next() {
		var item runtimeSettingRecord
		var isSecret int
		var isRestartRequired int
		if err := rows.Scan(
			&item.Key,
			&item.GroupName,
			&item.ValueText,
			&item.ValueCiphertext,
			&item.ValueType,
			&isSecret,
			&isRestartRequired,
			&item.Description,
			&item.UpdatedBySubject,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.IsSecret = isSecret == 1
		item.IsRestartRequired = isRestartRequired == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) InsertSettingAudit(ctx context.Context, record settingAuditRecord) error {
	_, err := r.db.ExecContext(ctx, `
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
