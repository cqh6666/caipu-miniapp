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
	return getBilibiliSession(ctx, r.db)
}

type bilibiliStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func getBilibiliSession(ctx context.Context, store bilibiliStore) (bilibiliSessionRecord, error) {
	var record bilibiliSessionRecord
	var updatedBy sql.NullInt64
	err := store.QueryRowContext(ctx, `
SELECT
	COALESCE(sessdata_ciphertext, ''),
	COALESCE(masked_sessdata, ''),
	COALESCE(status, ''),
	COALESCE(last_checked_at, ''),
	COALESCE(last_success_at, ''),
	COALESCE(last_error, ''),
	updated_by,
	COALESCE(updated_by_subject, ''),
	COALESCE(updated_at, ''),
	COALESCE((
		SELECT version
		FROM app_runtime_setting_groups
		WHERE group_name = 'bilibili.session'
	), 0)
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
		&updatedBy,
		&record.UpdatedBySubject,
		&record.UpdatedAt,
		&record.Version,
	)
	if err == sql.ErrNoRows {
		return bilibiliSessionRecord{}, nil
	}
	if err != nil {
		return bilibiliSessionRecord{}, err
	}
	if updatedBy.Valid {
		value := updatedBy.Int64
		record.UpdatedBy = &value
	}
	return record, nil
}

func (r *Repository) UpsertBilibiliSession(ctx context.Context, record bilibiliSessionRecord) error {
	return upsertBilibiliSession(ctx, r.db, record)
}

func upsertBilibiliSession(ctx context.Context, store bilibiliStore, record bilibiliSessionRecord) error {
	_, err := store.ExecContext(ctx, `
INSERT INTO app_bilibili_settings (
	id,
	sessdata_ciphertext,
	masked_sessdata,
	status,
	last_checked_at,
	last_success_at,
	last_error,
	updated_by,
	updated_by_subject,
	updated_at
) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	sessdata_ciphertext = excluded.sessdata_ciphertext,
	masked_sessdata = excluded.masked_sessdata,
	status = excluded.status,
	last_checked_at = excluded.last_checked_at,
	last_success_at = excluded.last_success_at,
	last_error = excluded.last_error,
	updated_by = excluded.updated_by,
	updated_by_subject = excluded.updated_by_subject,
	updated_at = excluded.updated_at
`,
		record.SessdataCiphertext,
		record.MaskedSessdata,
		record.Status,
		nullableText(record.LastCheckedAt),
		nullableText(record.LastSuccessAt),
		record.LastError,
		record.UpdatedBy,
		record.UpdatedBySubject,
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

type runtimeSettingsQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (r *Repository) ListRuntimeSettings(ctx context.Context) ([]runtimeSettingRecord, error) {
	return listRuntimeSettings(ctx, r.db)
}

func listRuntimeSettings(ctx context.Context, queryer runtimeSettingsQueryer) ([]runtimeSettingRecord, error) {
	rows, err := queryer.QueryContext(ctx, `
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

func (r *Repository) ListRuntimeSettingsSnapshot(ctx context.Context) ([]runtimeSettingRecord, map[string]int, error) {
	if r == nil || r.db == nil {
		return nil, map[string]int{}, nil
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, nil, err
	}
	records, err := listRuntimeSettings(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	versions, err := listRuntimeGroupVersions(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return records, versions, nil
}

func (r *Repository) InsertSettingAudit(ctx context.Context, record settingAuditRecord) error {
	return insertSettingAudit(ctx, r.db, record)
}

func insertSettingAudit(ctx context.Context, store bilibiliStore, record settingAuditRecord) error {
	_, err := store.ExecContext(ctx, `
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

func (r *Repository) SaveBilibiliSessionWithAudit(ctx context.Context, record bilibiliSessionRecord, audit settingAuditRecord, expectedVersion *int) (int, error) {
	if r == nil || r.db == nil {
		return 0, sql.ErrConnDone
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	previous, err := getBilibiliSession(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	audit.OldValueMasked = previous.MaskedSessdata
	version, err := bumpRuntimeGroupVersionTx(ctx, tx, "bilibili.session", expectedVersion, record.UpdatedBySubject, record.UpdatedAt)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := upsertBilibiliSession(ctx, tx, record); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := insertSettingAudit(ctx, tx, audit); err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return version, nil
}
