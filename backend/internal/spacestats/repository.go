package spacestats

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetStats(ctx context.Context, kitchenID int64, windowStart, today string) (Stats, error) {
	recipeStats, err := r.getRecipeStats(ctx, kitchenID, windowStart)
	if err != nil {
		return Stats{}, err
	}
	placeStats, err := r.getPlaceStats(ctx, kitchenID, windowStart)
	if err != nil {
		return Stats{}, err
	}
	mealPlanStats, err := r.getMealPlanStats(ctx, kitchenID, windowStart, today)
	if err != nil {
		return Stats{}, err
	}
	memberStats, err := r.getMemberStats(ctx, kitchenID, windowStart)
	if err != nil {
		return Stats{}, err
	}
	trends, err := r.getTrendStats(ctx, kitchenID, windowStart)
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{
		Source:    "remote",
		Recipes:   recipeStats,
		Places:    placeStats,
		MealPlans: mealPlanStats,
		Members:   memberStats,
		Trends:    trends,
	}
	stats.Overview = OverviewStats{
		RecipeTotal:              stats.Recipes.Total,
		PlaceTotal:               stats.Places.Total,
		SubmittedMealPlanDays:    stats.MealPlans.SubmittedDays,
		MemberTotal:              stats.Members.Total,
		WeeklyAvailableRecipes:   stats.Recipes.ByStatus["wishlist"],
		WeekendAvailablePlaces:   stats.Places.ByStatus["want"],
		TopRevisitPlaces:         r.topRevisitPlaces(ctx, kitchenID),
		RecentCreatedRecipes:     stats.Recipes.RecentCreatedTotal,
		RecentCreatedPlaces:      stats.Places.RecentCreatedTotal,
		RecentVisitedPlaces:      stats.Places.RecentVisitedTotal,
		RecentSubmittedMealPlans: stats.MealPlans.RecentSubmittedDays,
	}
	stats.Actions = buildActions(stats)
	return stats, nil
}

func (r *Repository) getRecipeStats(ctx context.Context, kitchenID int64, windowStart string) (RecipeStats, error) {
	stats := RecipeStats{
		ByMealType: map[string]int{"breakfast": 0, "main": 0},
		ByStatus:   map[string]int{"wishlist": 0, "done": 0},
	}
	var breakfastTotal, mainTotal, wishlistTotal, doneTotal int

	err := r.db.QueryRowContext(ctx, `
SELECT
  COUNT(1),
  COALESCE(SUM(CASE WHEN meal_type = 'breakfast' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN meal_type = 'main' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'wishlist' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN COALESCE(NULLIF(image_urls_json, ''), '[]') <> '[]' OR COALESCE(TRIM(image_url), '') <> '' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN parse_status = 'done' OR COALESCE(TRIM(steps_json), '') NOT IN ('', '[]') THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN flowchart_status = 'done' OR COALESCE(TRIM(flowchart_image_url), '') <> '' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN flowchart_status IN ('pending', 'processing') THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN COALESCE(TRIM(flowchart_image_url), '') = '' AND COALESCE(TRIM(steps_json), '') NOT IN ('', '[]') THEN 1 ELSE 0 END), 0)
FROM recipes
WHERE kitchen_id = ? AND deleted_at IS NULL`, kitchenID).Scan(
		&stats.Total,
		&breakfastTotal,
		&mainTotal,
		&wishlistTotal,
		&doneTotal,
		&stats.ImageCoveredTotal,
		&stats.ParsedTotal,
		&stats.FlowchartDoneTotal,
		&stats.FlowchartQueueTotal,
		&stats.FlowchartTodoTotal,
	)
	if err != nil {
		return RecipeStats{}, fmt.Errorf("aggregate recipe stats: %w", err)
	}
	stats.ByMealType["breakfast"] = breakfastTotal
	stats.ByMealType["main"] = mainTotal
	stats.ByStatus["wishlist"] = wishlistTotal
	stats.ByStatus["done"] = doneTotal
	stats.ImageCoverage = ratio(stats.ImageCoveredTotal, stats.Total)

	recentCondition, recentArgs := optionalWindow("created_at", windowStart)
	args := append([]any{kitchenID}, recentArgs...)
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM recipes
WHERE kitchen_id = ? AND deleted_at IS NULL`+recentCondition, args...).Scan(&stats.RecentCreatedTotal); err != nil {
		return RecipeStats{}, fmt.Errorf("count recent recipes: %w", err)
	}

	doneCondition, doneArgs := optionalWindow("changed_at", windowStart)
	args = append([]any{kitchenID}, doneArgs...)
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM recipe_status_events
WHERE kitchen_id = ? AND to_status = 'done'`+doneCondition, args...).Scan(&stats.DoneTrendTotal); err != nil {
		return RecipeStats{}, fmt.Errorf("count recipe done events: %w", err)
	}

	return stats, nil
}

func (r *Repository) getPlaceStats(ctx context.Context, kitchenID int64, windowStart string) (PlaceStats, error) {
	stats := PlaceStats{
		ByStatus:                  map[string]int{"want": 0, "visited": 0},
		PriceCurrency:             "CNY",
		RevisitRatingDistribution: map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0},
	}
	var wantTotal, visitedTotal, ratingSum, ratingCount int

	err := r.db.QueryRowContext(ctx, `
SELECT
  COUNT(1),
  COALESCE(SUM(CASE WHEN status = 'want' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'visited' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN latitude <> 0 AND longitude <> 0 THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN status = 'visited' AND (revisit_rating > 0 OR COALESCE(NULLIF(recommended_items_json, ''), '[]') <> '[]') THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN revisit_rating >= 4 THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN revisit_rating > 0 AND revisit_rating <= 2 THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN revisit_rating > 0 THEN revisit_rating ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN revisit_rating > 0 THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN COALESCE(TRIM(external_provider), '') <> '' AND COALESCE(TRIM(external_poi_id), '') <> '' THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN price_amount_cents > 0 THEN 1 ELSE 0 END), 0),
  COALESCE(SUM(CASE WHEN price_amount_cents > 0 THEN price_amount_cents ELSE 0 END), 0)
FROM places
WHERE kitchen_id = ? AND deleted_at IS NULL`, kitchenID).Scan(
		&stats.Total,
		&wantTotal,
		&visitedTotal,
		&stats.LocatedTotal,
		&stats.ExperienceCompletedTotal,
		&stats.HighlyRecommendedTotal,
		&stats.LowRatingTotal,
		&ratingSum,
		&ratingCount,
		&stats.POIMatchedTotal,
		&stats.PricedPlaceTotal,
		&stats.TotalPriceAmountCents,
	)
	if err != nil {
		return PlaceStats{}, fmt.Errorf("aggregate place stats: %w", err)
	}
	stats.ByStatus["want"] = wantTotal
	stats.ByStatus["visited"] = visitedTotal
	stats.LocationCoverage = ratio(stats.LocatedTotal, stats.Total)
	if ratingCount > 0 {
		stats.AverageRevisitRating = round2(float64(ratingSum) / float64(ratingCount))
	}
	if stats.PricedPlaceTotal > 0 {
		stats.AveragePriceAmountCents = stats.TotalPriceAmountCents / int64(stats.PricedPlaceTotal)
	}

	distribution, err := r.countPlaceRevisitRatings(ctx, kitchenID)
	if err != nil {
		return PlaceStats{}, err
	}
	for key, count := range distribution {
		stats.RevisitRatingDistribution[key] = count
	}

	createdCondition, createdArgs := optionalWindow("created_at", windowStart)
	args := append([]any{kitchenID}, createdArgs...)
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM places
WHERE kitchen_id = ? AND deleted_at IS NULL`+createdCondition, args...).Scan(&stats.RecentCreatedTotal); err != nil {
		return PlaceStats{}, fmt.Errorf("count recent places: %w", err)
	}

	visitedCondition, visitedArgs := optionalWindow("changed_at", windowStart)
	args = append([]any{kitchenID}, visitedArgs...)
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM place_status_events
WHERE kitchen_id = ? AND to_status = 'visited'`+visitedCondition, args...).Scan(&stats.RecentVisitedTotal); err != nil {
		return PlaceStats{}, fmt.Errorf("count visited place events: %w", err)
	}

	stats.TopRecommendedItems, err = r.topJSONLabels(ctx, kitchenID, "recommended_items_json", 5)
	if err != nil {
		return PlaceStats{}, err
	}
	stats.TopScenes, err = r.topSceneLabels(ctx, kitchenID, 5)
	if err != nil {
		return PlaceStats{}, err
	}

	return stats, nil
}

func (r *Repository) getMealPlanStats(ctx context.Context, kitchenID int64, windowStart, today string) (MealPlanStats, error) {
	stats := MealPlanStats{
		ItemsByMealType: map[string]int{"breakfast": 0, "main": 0},
	}

	if err := r.db.QueryRowContext(ctx, `
SELECT
  COUNT(DISTINCT CASE WHEN status = 'draft' AND EXISTS (SELECT 1 FROM meal_plan_items mpi WHERE mpi.plan_id = meal_plans.id) THEN plan_date END),
  COUNT(DISTINCT CASE WHEN status = 'submitted' THEN plan_date END)
FROM meal_plans
WHERE kitchen_id = ?`, kitchenID).Scan(&stats.DraftDays, &stats.SubmittedDays); err != nil {
		return MealPlanStats{}, fmt.Errorf("aggregate meal plan days: %w", err)
	}

	var avg sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, `
SELECT AVG(item_count)
FROM (
  SELECT COUNT(mpi.id) AS item_count
  FROM meal_plans mp
  LEFT JOIN meal_plan_items mpi ON mpi.plan_id = mp.id
  WHERE mp.kitchen_id = ? AND mp.status = 'submitted'
  GROUP BY mp.id
)`, kitchenID).Scan(&avg); err != nil {
		return MealPlanStats{}, fmt.Errorf("average meal plan dishes: %w", err)
	}
	if avg.Valid {
		stats.AverageDishCount = round2(avg.Float64)
	}

	condition, argsTail := optionalWindow("COALESCE(NULLIF(submitted_at, ''), updated_at, created_at)", windowStart)
	args := append([]any{kitchenID}, argsTail...)
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(DISTINCT plan_date)
FROM meal_plans
WHERE kitchen_id = ? AND status = 'submitted'`+condition, args...).Scan(&stats.RecentSubmittedDays); err != nil {
		return MealPlanStats{}, fmt.Errorf("count recent submitted meal plans: %w", err)
	}

	next, err := r.getMealPlanBrief(ctx, `
SELECT mp.id, mp.plan_date, mp.status, COUNT(mpi.id)
FROM meal_plans mp
LEFT JOIN meal_plan_items mpi ON mpi.plan_id = mp.id
WHERE mp.kitchen_id = ? AND mp.status = 'submitted' AND mp.plan_date >= ?
GROUP BY mp.id, mp.plan_date, mp.status
ORDER BY mp.plan_date ASC, mp.id DESC
LIMIT 1`, kitchenID, today)
	if err != nil {
		return MealPlanStats{}, err
	}
	stats.NextPlan = next

	latest, err := r.getMealPlanBrief(ctx, `
SELECT mp.id, mp.plan_date, mp.status, COUNT(mpi.id)
FROM meal_plans mp
LEFT JOIN meal_plan_items mpi ON mpi.plan_id = mp.id
WHERE mp.kitchen_id = ? AND mp.status = 'submitted'
GROUP BY mp.id, mp.plan_date, mp.status, mp.submitted_at, mp.updated_at, mp.created_at
ORDER BY mp.plan_date DESC, COALESCE(NULLIF(mp.submitted_at, ''), mp.updated_at, mp.created_at) DESC, mp.id DESC
LIMIT 1`, kitchenID)
	if err != nil {
		return MealPlanStats{}, err
	}
	stats.LatestPlan = latest

	rows, err := r.db.QueryContext(ctx, `
SELECT COALESCE(NULLIF(mpi.meal_type_snapshot, ''), 'main'), COUNT(1)
FROM meal_plans mp
JOIN meal_plan_items mpi ON mpi.plan_id = mp.id
WHERE mp.kitchen_id = ? AND mp.status = 'submitted'
GROUP BY COALESCE(NULLIF(mpi.meal_type_snapshot, ''), 'main')`, kitchenID)
	if err != nil {
		return MealPlanStats{}, fmt.Errorf("count meal plan items by meal type: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return MealPlanStats{}, fmt.Errorf("scan meal plan item type: %w", err)
		}
		stats.ItemsByMealType[key] = count
	}
	if err := rows.Err(); err != nil {
		return MealPlanStats{}, fmt.Errorf("iterate meal plan item types: %w", err)
	}

	return stats, nil
}

func (r *Repository) getMemberStats(ctx context.Context, kitchenID int64, windowStart string) (MemberStats, error) {
	recipeWindow, recipeArgs := optionalWindow("r.created_at", windowStart)
	placeWindow, placeArgs := optionalWindow("p.created_at", windowStart)
	mealWindow, mealArgs := optionalWindow("COALESCE(NULLIF(mp.submitted_at, ''), mp.updated_at, mp.created_at)", windowStart)
	args := []any{kitchenID}
	args = append(args, recipeArgs...)
	args = append(args, kitchenID)
	args = append(args, placeArgs...)
	args = append(args, kitchenID)
	args = append(args, mealArgs...)
	args = append(args, kitchenID)

	query := fmt.Sprintf(`
SELECT
  km.user_id,
  COALESCE(u.nickname, ''),
  COALESCE(u.avatar_url, ''),
  km.role,
  km.joined_at,
  COALESCE(rc.total, 0),
  COALESCE(pc.total, 0),
  COALESCE(mc.total, 0)
FROM kitchen_members km
JOIN users u ON u.id = km.user_id
LEFT JOIN (
  SELECT r.created_by AS user_id, COUNT(1) AS total
  FROM recipes r
  WHERE r.kitchen_id = ? AND r.deleted_at IS NULL%s
  GROUP BY r.created_by
) rc ON rc.user_id = km.user_id
LEFT JOIN (
  SELECT p.created_by AS user_id, COUNT(1) AS total
  FROM places p
  WHERE p.kitchen_id = ? AND p.deleted_at IS NULL%s
  GROUP BY p.created_by
) pc ON pc.user_id = km.user_id
LEFT JOIN (
  SELECT CASE WHEN mp.submitted_by > 0 THEN mp.submitted_by ELSE mp.updated_by END AS user_id, COUNT(1) AS total
  FROM meal_plans mp
  WHERE mp.kitchen_id = ? AND mp.status = 'submitted'%s
  GROUP BY CASE WHEN mp.submitted_by > 0 THEN mp.submitted_by ELSE mp.updated_by END
) mc ON mc.user_id = km.user_id
WHERE km.kitchen_id = ?
ORDER BY (COALESCE(rc.total, 0) + COALESCE(pc.total, 0) + COALESCE(mc.total, 0)) DESC,
  CASE km.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END,
  km.joined_at ASC,
  km.user_id ASC`, recipeWindow, placeWindow, mealWindow)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return MemberStats{}, fmt.Errorf("aggregate member contributions: %w", err)
	}
	defer rows.Close()

	stats := MemberStats{}
	for rows.Next() {
		var item MemberContribution
		if err := rows.Scan(
			&item.UserID,
			&item.Nickname,
			&item.AvatarURL,
			&item.Role,
			&item.JoinedAt,
			&item.RecipeCreatedTotal,
			&item.PlaceCreatedTotal,
			&item.MealPlanSubmittedTotal,
		); err != nil {
			return MemberStats{}, fmt.Errorf("scan member contribution: %w", err)
		}
		item.Total = item.RecipeCreatedTotal + item.PlaceCreatedTotal + item.MealPlanSubmittedTotal
		stats.Contributors = append(stats.Contributors, item)
	}
	if err := rows.Err(); err != nil {
		return MemberStats{}, fmt.Errorf("iterate member contributions: %w", err)
	}
	stats.Total = len(stats.Contributors)
	return stats, nil
}

func (r *Repository) getTrendStats(ctx context.Context, kitchenID int64, windowStart string) (TrendStats, error) {
	var stats TrendStats
	var err error
	stats.RecipeCreated, err = r.dailyPoints(ctx, `
SELECT substr(created_at, 1, 10), COUNT(1)
FROM recipes
WHERE kitchen_id = ? AND deleted_at IS NULL%s
GROUP BY substr(created_at, 1, 10)
ORDER BY substr(created_at, 1, 10) ASC`, kitchenID, "created_at", windowStart)
	if err != nil {
		return TrendStats{}, fmt.Errorf("recipe created trend: %w", err)
	}
	stats.RecipeDone, err = r.dailyPoints(ctx, `
SELECT substr(changed_at, 1, 10), COUNT(1)
FROM recipe_status_events
WHERE kitchen_id = ? AND to_status = 'done'%s
GROUP BY substr(changed_at, 1, 10)
ORDER BY substr(changed_at, 1, 10) ASC`, kitchenID, "changed_at", windowStart)
	if err != nil {
		return TrendStats{}, fmt.Errorf("recipe done trend: %w", err)
	}
	stats.PlaceCreated, err = r.dailyPoints(ctx, `
SELECT substr(created_at, 1, 10), COUNT(1)
FROM places
WHERE kitchen_id = ? AND deleted_at IS NULL%s
GROUP BY substr(created_at, 1, 10)
ORDER BY substr(created_at, 1, 10) ASC`, kitchenID, "created_at", windowStart)
	if err != nil {
		return TrendStats{}, fmt.Errorf("place created trend: %w", err)
	}
	stats.PlaceVisited, err = r.dailyPoints(ctx, `
SELECT substr(changed_at, 1, 10), COUNT(1)
FROM place_status_events
WHERE kitchen_id = ? AND to_status = 'visited'%s
GROUP BY substr(changed_at, 1, 10)
ORDER BY substr(changed_at, 1, 10) ASC`, kitchenID, "changed_at", windowStart)
	if err != nil {
		return TrendStats{}, fmt.Errorf("place visited trend: %w", err)
	}
	stats.MealPlanSubmitted, err = r.dailyPoints(ctx, `
SELECT substr(COALESCE(NULLIF(submitted_at, ''), updated_at, created_at), 1, 10), COUNT(DISTINCT plan_date)
FROM meal_plans
WHERE kitchen_id = ? AND status = 'submitted'%s
GROUP BY substr(COALESCE(NULLIF(submitted_at, ''), updated_at, created_at), 1, 10)
ORDER BY substr(COALESCE(NULLIF(submitted_at, ''), updated_at, created_at), 1, 10) ASC`, kitchenID, "COALESCE(NULLIF(submitted_at, ''), updated_at, created_at)", windowStart)
	if err != nil {
		return TrendStats{}, fmt.Errorf("meal plan submitted trend: %w", err)
	}
	return stats, nil
}

func (r *Repository) topRevisitPlaces(ctx context.Context, kitchenID int64) []TopRevisitPlace {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, name, revisit_rating, COALESCE(recommended_items_json, '[]'),
       COALESCE(json_extract(COALESCE(NULLIF(image_urls_json, ''), '[]'), '$[0]'), ''),
       COALESCE(visited_at, '')
FROM places
WHERE kitchen_id = ? AND deleted_at IS NULL AND status = 'visited' AND revisit_rating >= 4
ORDER BY revisit_rating DESC, COALESCE(NULLIF(visited_at, ''), updated_at, created_at) DESC, updated_at DESC, id DESC
LIMIT 3`, kitchenID)
	if err != nil {
		return []TopRevisitPlace{}
	}
	defer rows.Close()

	items := make([]TopRevisitPlace, 0, 3)
	for rows.Next() {
		var item TopRevisitPlace
		var recommendedItemsJSON string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.RevisitRating,
			&recommendedItemsJSON,
			&item.ImageURL,
			&item.VisitedAt,
		); err != nil {
			return []TopRevisitPlace{}
		}
		item.RecommendedItems = unmarshalStringList(recommendedItemsJSON)
		items = append(items, item)
	}
	return items
}

func (r *Repository) countPlaceRevisitRatings(ctx context.Context, kitchenID int64) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT CAST(revisit_rating AS TEXT), COUNT(1)
FROM places
WHERE kitchen_id = ? AND deleted_at IS NULL AND revisit_rating BETWEEN 1 AND 5
GROUP BY revisit_rating`, kitchenID)
	if err != nil {
		return nil, fmt.Errorf("count revisit rating distribution: %w", err)
	}
	defer rows.Close()

	result := map[string]int{}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scan revisit rating distribution: %w", err)
		}
		result[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate revisit rating distribution: %w", err)
	}
	return result, nil
}

func (r *Repository) topJSONLabels(ctx context.Context, kitchenID int64, column string, limit int) ([]CountedLabel, error) {
	if column != "recommended_items_json" && column != "scenes_json" && column != "tags_json" {
		return nil, fmt.Errorf("unsupported json label column: %s", column)
	}
	query := fmt.Sprintf(`
SELECT TRIM(json_each.value), COUNT(1)
FROM places, json_each(COALESCE(NULLIF(places.%s, ''), '[]'))
WHERE places.kitchen_id = ? AND places.deleted_at IS NULL AND TRIM(json_each.value) <> ''
GROUP BY TRIM(json_each.value)
ORDER BY COUNT(1) DESC, TRIM(json_each.value) ASC
LIMIT ?`, column)
	return queryCountedLabels(ctx, r.db, query, kitchenID, limit)
}

func (r *Repository) topSceneLabels(ctx context.Context, kitchenID int64, limit int) ([]CountedLabel, error) {
	const query = `
SELECT label, COUNT(1)
FROM (
  SELECT TRIM(json_each.value) AS label
  FROM places, json_each(COALESCE(NULLIF(places.scenes_json, ''), '[]'))
  WHERE places.kitchen_id = ? AND places.deleted_at IS NULL AND TRIM(json_each.value) <> ''
  UNION ALL
  SELECT TRIM(json_each.value) AS label
  FROM places, json_each(COALESCE(NULLIF(places.tags_json, ''), '[]'))
  WHERE places.kitchen_id = ? AND places.deleted_at IS NULL AND TRIM(json_each.value) <> ''
)
GROUP BY label
ORDER BY COUNT(1) DESC, label ASC
LIMIT ?`
	return queryCountedLabels(ctx, r.db, query, kitchenID, kitchenID, limit)
}

func (r *Repository) getMealPlanBrief(ctx context.Context, query string, args ...any) (*MealPlanBrief, error) {
	var item MealPlanBrief
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&item.ID, &item.PlanDate, &item.Status, &item.ItemCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get meal plan brief: %w", err)
	}
	return &item, nil
}

func (r *Repository) dailyPoints(ctx context.Context, queryTemplate string, kitchenID int64, column, windowStart string) ([]DailyPoint, error) {
	condition, tail := optionalWindow(column, windowStart)
	args := append([]any{kitchenID}, tail...)
	query := fmt.Sprintf(queryTemplate, condition)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]DailyPoint, 0)
	for rows.Next() {
		var item DailyPoint
		if err := rows.Scan(&item.Date, &item.Count); err != nil {
			return nil, err
		}
		if strings.TrimSpace(item.Date) != "" {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func queryCountedLabels(ctx context.Context, db *sql.DB, query string, args ...any) ([]CountedLabel, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CountedLabel, 0)
	for rows.Next() {
		var item CountedLabel
		if err := rows.Scan(&item.Label, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func optionalWindow(column, windowStart string) (string, []any) {
	if strings.TrimSpace(windowStart) == "" {
		return "", nil
	}
	return fmt.Sprintf(" AND %s >= ?", column), []any{windowStart}
}

func ratio(numerator int, denominator int) float64 {
	if denominator <= 0 {
		return 0
	}
	return round2(float64(numerator) / float64(denominator))
}

func round2(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func unmarshalStringList(raw string) []string {
	var values []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return []string{}
	}
	if values == nil {
		return []string{}
	}
	return values
}

func buildActions(stats Stats) []Action {
	actions := make([]Action, 0, 4)
	if stats.Recipes.ByStatus["wishlist"] > 0 {
		actions = append(actions, Action{
			Type:   "view_weekly_available_recipes",
			Label:  "查看本周可选",
			Count:  stats.Recipes.ByStatus["wishlist"],
			Target: "recipes:wishlist",
		})
	}
	if stats.Places.ByStatus["want"] > 0 {
		actions = append(actions, Action{
			Type:   "view_weekend_available_places",
			Label:  "查看周末可去",
			Count:  stats.Places.ByStatus["want"],
			Target: "places:want",
		})
	}
	if stats.MealPlans.DraftDays > 0 {
		actions = append(actions, Action{
			Type:   "view_meal_plan_drafts",
			Label:  "继续安排草稿",
			Count:  stats.MealPlans.DraftDays,
			Target: "mealPlans:draft",
		})
	}
	if len(stats.Overview.TopRevisitPlaces) > 0 {
		actions = append(actions, Action{
			Type:   "view_top_revisit_places",
			Label:  "查看高分复访",
			Count:  len(stats.Overview.TopRevisitPlaces),
			Target: "places:revisit",
		})
	}
	return actions
}
