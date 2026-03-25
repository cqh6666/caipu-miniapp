package mealplan

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByKitchenID(ctx context.Context, kitchenID int64) ([]Plan, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, kitchen_id, plan_date, status, COALESCE(note, ''), created_by, updated_by,
		        COALESCE(submitted_by, 0), created_at, updated_at, COALESCE(submitted_at, '')
		   FROM meal_plans
		  WHERE kitchen_id = ?
		  ORDER BY plan_date ASC, CASE status WHEN 'draft' THEN 0 ELSE 1 END ASC, submitted_at DESC, id DESC`,
		kitchenID,
	)
	if err != nil {
		return nil, fmt.Errorf("list meal plans by kitchen: %w", err)
	}
	defer rows.Close()

	plans := make([]Plan, 0)
	planIDs := make([]int64, 0)
	for rows.Next() {
		var item Plan
		if err := rows.Scan(
			&item.ID,
			&item.KitchenID,
			&item.PlanDate,
			&item.Status,
			&item.Note,
			&item.CreatedBy,
			&item.UpdatedBy,
			&item.SubmittedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.SubmittedAt,
		); err != nil {
			return nil, fmt.Errorf("scan meal plan: %w", err)
		}
		plans = append(plans, item)
		planIDs = append(planIDs, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate meal plans: %w", err)
	}
	if len(plans) == 0 {
		return nil, nil
	}

	itemsByPlanID, err := r.listItemsByPlanIDs(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	for index := range plans {
		plans[index].Items = itemsByPlanID[plans[index].ID]
	}

	return plans, nil
}

func (r *Repository) CountRecipesByKitchenID(ctx context.Context, kitchenID int64, recipeIDs []string) (int, error) {
	if len(recipeIDs) == 0 {
		return 0, nil
	}

	placeholders := make([]string, 0, len(recipeIDs))
	args := make([]any, 0, len(recipeIDs)+1)
	args = append(args, kitchenID)
	for _, recipeID := range recipeIDs {
		placeholders = append(placeholders, "?")
		args = append(args, recipeID)
	}

	query := fmt.Sprintf(
		`SELECT COUNT(1)
		   FROM recipes
		  WHERE kitchen_id = ? AND deleted_at IS NULL AND id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count recipes by kitchen: %w", err)
	}

	return count, nil
}

func (r *Repository) ReplaceDraft(ctx context.Context, plan Plan, touchedAt string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace meal draft tx: %w", err)
	}

	if err := deletePlanByKitchenDateStatusTx(ctx, tx, plan.KitchenID, plan.PlanDate, StatusDraft); err != nil {
		_ = tx.Rollback()
		return err
	}

	if hasPlanContent(plan) {
		if _, err := insertPlanTx(ctx, tx, plan); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := bumpKitchenUpdatedAt(ctx, tx, plan.KitchenID, touchedAt); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace meal draft: %w", err)
	}

	return nil
}

func (r *Repository) ReplaceSubmitted(ctx context.Context, plan Plan, touchedAt string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace submitted meal plan tx: %w", err)
	}

	if err := deletePlanByKitchenDateStatusTx(ctx, tx, plan.KitchenID, plan.PlanDate, StatusSubmitted); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := insertPlanTx(ctx, tx, plan); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := deletePlanByKitchenDateStatusTx(ctx, tx, plan.KitchenID, plan.PlanDate, StatusDraft); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := bumpKitchenUpdatedAt(ctx, tx, plan.KitchenID, touchedAt); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace submitted meal plan: %w", err)
	}

	return nil
}

func (r *Repository) listItemsByPlanIDs(ctx context.Context, planIDs []int64) (map[int64][]Item, error) {
	if len(planIDs) == 0 {
		return map[int64][]Item{}, nil
	}

	placeholders := make([]string, 0, len(planIDs))
	args := make([]any, 0, len(planIDs))
	for _, id := range planIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`SELECT plan_id, recipe_id, quantity, meal_type_snapshot, COALESCE(title_snapshot, ''), COALESCE(image_snapshot, ''), sort_index
		   FROM meal_plan_items
		  WHERE plan_id IN (%s)
		  ORDER BY plan_id ASC, sort_index ASC, id ASC`,
		strings.Join(placeholders, ", "),
	)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list meal plan items: %w", err)
	}
	defer rows.Close()

	itemsByPlanID := make(map[int64][]Item, len(planIDs))
	for rows.Next() {
		var planID int64
		var item Item
		if err := rows.Scan(
			&planID,
			&item.RecipeID,
			&item.Quantity,
			&item.MealTypeSnapshot,
			&item.TitleSnapshot,
			&item.ImageSnapshot,
			&item.Sort,
		); err != nil {
			return nil, fmt.Errorf("scan meal plan item: %w", err)
		}
		itemsByPlanID[planID] = append(itemsByPlanID[planID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate meal plan items: %w", err)
	}

	return itemsByPlanID, nil
}

func insertPlanTx(ctx context.Context, tx *sql.Tx, plan Plan) (int64, error) {
	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO meal_plans (
		     kitchen_id, plan_date, status, note, created_by, updated_by, submitted_by, created_at, updated_at, submitted_at
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		plan.KitchenID,
		plan.PlanDate,
		plan.Status,
		plan.Note,
		plan.CreatedBy,
		plan.UpdatedBy,
		plan.SubmittedBy,
		plan.CreatedAt,
		plan.UpdatedAt,
		nonNullableTrimmedString(plan.SubmittedAt),
	)
	if err != nil {
		return 0, fmt.Errorf("insert meal plan: %w", err)
	}

	planID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read meal plan id: %w", err)
	}

	for index, item := range plan.Items {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO meal_plan_items (
			     plan_id, recipe_id, quantity, meal_type_snapshot, title_snapshot, image_snapshot, sort_index, created_at, updated_at
			 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			planID,
			item.RecipeID,
			item.Quantity,
			item.MealTypeSnapshot,
			item.TitleSnapshot,
			nonNullableTrimmedString(item.ImageSnapshot),
			index,
			plan.UpdatedAt,
			plan.UpdatedAt,
		); err != nil {
			return 0, fmt.Errorf("insert meal plan item: %w", err)
		}
	}

	return planID, nil
}

func deletePlanByKitchenDateStatusTx(ctx context.Context, tx *sql.Tx, kitchenID int64, planDate, status string) error {
	rows, err := tx.QueryContext(
		ctx,
		`SELECT id FROM meal_plans WHERE kitchen_id = ? AND plan_date = ? AND status = ?`,
		kitchenID,
		planDate,
		status,
	)
	if err != nil {
		return fmt.Errorf("query existing meal plans: %w", err)
	}
	defer rows.Close()

	planIDs := make([]int64, 0, 2)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan existing meal plan id: %w", err)
		}
		planIDs = append(planIDs, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate existing meal plan ids: %w", err)
	}

	for _, id := range planIDs {
		if _, err := tx.ExecContext(ctx, `DELETE FROM meal_plan_items WHERE plan_id = ?`, id); err != nil {
			return fmt.Errorf("delete meal plan items: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM meal_plans WHERE id = ?`, id); err != nil {
			return fmt.Errorf("delete meal plan: %w", err)
		}
	}

	return nil
}

func bumpKitchenUpdatedAt(ctx context.Context, tx *sql.Tx, kitchenID int64, updatedAt string) error {
	if _, err := tx.ExecContext(ctx, `UPDATE kitchens SET updated_at = ? WHERE id = ?`, updatedAt, kitchenID); err != nil {
		return fmt.Errorf("touch kitchen updated_at: %w", err)
	}
	return nil
}

func hasPlanContent(plan Plan) bool {
	return len(plan.Items) > 0 || strings.TrimSpace(plan.Note) != ""
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func nonNullableTrimmedString(value string) string {
	return strings.TrimSpace(value)
}
