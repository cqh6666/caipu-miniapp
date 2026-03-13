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
