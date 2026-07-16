package appsettings

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func listRuntimeGroupVersions(ctx context.Context, queryer runtimeSettingsQueryer) (map[string]int, error) {
	rows, err := queryer.QueryContext(ctx, `SELECT group_name, version FROM app_runtime_setting_groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	versions := make(map[string]int)
	for rows.Next() {
		var groupName string
		var version int
		if err := rows.Scan(&groupName, &version); err != nil {
			return nil, err
		}
		versions[groupName] = version
	}
	return versions, rows.Err()
}

func bumpRuntimeGroupVersionTx(
	ctx context.Context,
	tx *sql.Tx,
	groupName string,
	expectedVersion *int,
	subject string,
	updatedAt string,
) (int, error) {
	groupName = strings.TrimSpace(groupName)
	if expectedVersion == nil {
		var version int
		err := tx.QueryRowContext(ctx, `
INSERT INTO app_runtime_setting_groups (group_name, version, updated_by_subject, updated_at)
VALUES (?, 1, ?, ?)
ON CONFLICT(group_name) DO UPDATE SET
	version = app_runtime_setting_groups.version + 1,
	updated_by_subject = excluded.updated_by_subject,
	updated_at = excluded.updated_at
RETURNING version
`, groupName, strings.TrimSpace(subject), updatedAt).Scan(&version)
		return version, err
	}
	if *expectedVersion < 0 {
		return 0, common.NewAppError(common.CodeBadRequest, "expectedVersion must not be negative", http.StatusBadRequest)
	}

	if *expectedVersion == 0 {
		result, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO app_runtime_setting_groups (
	group_name,
	version,
	updated_by_subject,
	updated_at
) VALUES (?, 1, ?, ?)
`, groupName, strings.TrimSpace(subject), updatedAt)
		if err != nil {
			return 0, err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		if affected == 1 {
			return 1, nil
		}
		return 0, runtimeGroupConflictError()
	}

	result, err := tx.ExecContext(ctx, `
UPDATE app_runtime_setting_groups
SET
	version = version + 1,
	updated_by_subject = ?,
	updated_at = ?
WHERE group_name = ? AND version = ?
`, strings.TrimSpace(subject), updatedAt, groupName, *expectedVersion)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected != 1 {
		return 0, runtimeGroupConflictError()
	}
	return *expectedVersion + 1, nil
}

func runtimeGroupConflictError() error {
	return common.NewAppError(common.CodeConflict, "运行时配置已被其他会话更新，请刷新后重试", http.StatusConflict)
}
