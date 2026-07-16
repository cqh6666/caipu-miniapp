package recipe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// GetShareToken 读取菜谱当前已有的 share_token；菜谱不存在或已软删返回 sql.ErrNoRows
func (r *Repository) GetShareToken(ctx context.Context, recipeID string) (string, error) {
	const query = `
	SELECT COALESCE(share_token, '')
	FROM recipes
	WHERE id = ? AND deleted_at IS NULL
	LIMIT 1
	`
	var token string
	if err := r.db.QueryRowContext(ctx, query, recipeID).Scan(&token); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		return "", fmt.Errorf("get share token: %w", err)
	}
	return strings.TrimSpace(token), nil
}

// SetShareToken 写入新的 share_token 和创建时间
// P1 修复（并发原子化）：仅当 share_token 为空时才写入，避免并发覆盖
//   - 返回 true：本次成功写入（调用方拿到的 token 即生效 token）
//   - 返回 false：另一并发请求已先写入，调用方应回查库里现有 token
//   - 返回 sql.ErrNoRows：菜谱不存在或已软删
func (r *Repository) SetShareToken(ctx context.Context, recipeID, token, createdAt string) (bool, error) {
	const query = `
	UPDATE recipes
	SET share_token = ?, share_token_created_at = ?
	WHERE id = ? AND deleted_at IS NULL AND (share_token IS NULL OR share_token = '')
	`
	result, err := r.db.ExecContext(ctx, query, strings.TrimSpace(token), strings.TrimSpace(createdAt), recipeID)
	if err != nil {
		return false, fmt.Errorf("set share token: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("set share token rows affected: %w", err)
	}
	if affected == 1 {
		return true, nil
	}
	// affected == 0：可能菜谱已删，也可能 share_token 已有值（并发竞争失败）
	// 调用方需通过 GetShareToken 区分这两种情况
	return false, nil
}

// FindByShareToken 通过 share_token 查菜谱（不需要鉴权，公开只读接口使用）
// 走 scanRecipe 主流程，返回的 Recipe 不带 ShareToken 字段
func (r *Repository) FindByShareToken(ctx context.Context, token string) (Recipe, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return Recipe{}, sql.ErrNoRows
	}
	const query = `
	SELECT id, kitchen_id, title, COALESCE(title_source, 'manual'), COALESCE(ingredient, ''), COALESCE(summary, ''), COALESCE(link, ''), COALESCE(image_url, ''), COALESCE(image_urls_json, '[]'), COALESCE(image_meta_json, '[]'),
	       COALESCE(flowchart_image_url, ''), COALESCE(flowchart_provider, ''), COALESCE(flowchart_model, ''), COALESCE(flowchart_updated_at, ''), COALESCE(flowchart_source_hash, ''),
	       COALESCE(flowchart_status, ''), COALESCE(flowchart_error, ''), COALESCE(flowchart_requested_at, ''), COALESCE(flowchart_finished_at, ''),
	       meal_type, status, COALESCE(note, ''), ingredients_json, steps_json,
	       COALESCE(parse_status, ''), COALESCE(parse_source, ''), COALESCE(parse_error, ''),
	       COALESCE(parse_requested_at, ''), COALESCE(parse_finished_at, ''), COALESCE(parse_attempts, 0), COALESCE(parse_next_attempt_at, ''), COALESCE(parse_last_error_type, ''), COALESCE(parse_processing_started_at, ''), COALESCE(parsed_content_edited, 0), COALESCE(pinned_at, ''), COALESCE(done_at, ''),
	       created_by, updated_by, created_at, updated_at, COALESCE(version, 1)
	FROM recipes
	WHERE share_token = ? AND deleted_at IS NULL
	LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, trimmed)
	item, err := scanRecipe(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, err
	}
	if err != nil {
		return Recipe{}, fmt.Errorf("find recipe by share token: %w", err)
	}
	return item, nil
}

// FindKitchenAndCreatorMeta 取菜谱所属空间名 + 创建者昵称，用于公开只读接口附加上下文
// 不存在或软删返回 sql.ErrNoRows；nickname 缺失会回退为空字符串
func (r *Repository) FindKitchenAndCreatorMeta(ctx context.Context, recipeID string) (kitchenName, ownerNickname string, err error) {
	const query = `
	SELECT COALESCE(k.name, ''), COALESCE(u.nickname, '')
	FROM recipes r
	JOIN kitchens k ON k.id = r.kitchen_id
	LEFT JOIN users u ON u.id = r.created_by
	WHERE r.id = ? AND r.deleted_at IS NULL
	LIMIT 1
	`
	if err := r.db.QueryRowContext(ctx, query, recipeID).Scan(&kitchenName, &ownerNickname); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", err
		}
		return "", "", fmt.Errorf("find kitchen and creator meta: %w", err)
	}
	return kitchenName, ownerNickname, nil
}
