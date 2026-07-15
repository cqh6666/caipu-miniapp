package credentialrotate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cqh6666/caipu-miniapp/backend/internal/credentialcipher"
)

type Result struct {
	Scanned int
	Changed int
}

func Rotate(ctx context.Context, db *sql.DB, box *credentialcipher.Box, apply bool) (Result, error) {
	if db == nil || box == nil {
		return Result{}, fmt.Errorf("database and credential cipher are required")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, fmt.Errorf("begin credential rotation: %w", err)
	}
	defer tx.Rollback()

	result := Result{}
	targets := []struct {
		table  string
		key    string
		column string
	}{
		{table: "app_bilibili_settings", key: "id", column: "sessdata_ciphertext"},
		{table: "app_runtime_settings", key: "key", column: "value_ciphertext"},
		{table: "ai_route_providers", key: "id", column: "api_key_ciphertext"},
	}
	for _, target := range targets {
		rotated, err := rotateTable(ctx, tx, box, target.table, target.key, target.column, apply)
		if err != nil {
			return Result{}, err
		}
		result.Scanned += rotated.Scanned
		result.Changed += rotated.Changed
	}

	if !apply {
		return result, nil
	}
	if err := tx.Commit(); err != nil {
		return Result{}, fmt.Errorf("commit credential rotation: %w", err)
	}
	return result, nil
}

func rotateTable(ctx context.Context, tx *sql.Tx, box *credentialcipher.Box, table, keyColumn, cipherColumn string, apply bool) (Result, error) {
	query := fmt.Sprintf("SELECT %s, %s FROM %s WHERE COALESCE(%s, '') <> ''", keyColumn, cipherColumn, table, cipherColumn)
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return Result{}, fmt.Errorf("scan %s credentials: %w", table, err)
	}
	type item struct {
		key        any
		ciphertext string
	}
	items := make([]item, 0)
	for rows.Next() {
		var current item
		if err := rows.Scan(&current.key, &current.ciphertext); err != nil {
			rows.Close()
			return Result{}, fmt.Errorf("read %s credential: %w", table, err)
		}
		items = append(items, current)
	}
	if err := rows.Close(); err != nil {
		return Result{}, fmt.Errorf("close %s credential scan: %w", table, err)
	}

	result := Result{Scanned: len(items)}
	for _, current := range items {
		rotated, changed, err := box.Reencrypt(current.ciphertext)
		if err != nil {
			return Result{}, fmt.Errorf("rotate %s credential %v: %w", table, current.key, err)
		}
		if !changed {
			continue
		}
		result.Changed++
		if !apply {
			continue
		}
		update := fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ? AND %s = ?", table, cipherColumn, keyColumn, cipherColumn)
		updated, err := tx.ExecContext(ctx, update, rotated, current.key, current.ciphertext)
		if err != nil {
			return Result{}, fmt.Errorf("update %s credential %v: %w", table, current.key, err)
		}
		rowsAffected, err := updated.RowsAffected()
		if err != nil || rowsAffected != 1 {
			return Result{}, fmt.Errorf("credential %s/%v changed concurrently", table, current.key)
		}
	}
	return result, nil
}
