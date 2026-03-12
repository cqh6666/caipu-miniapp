package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"github.com/cqh6666/caipu-miniapp/backend/internal/db"
)

type demoRecipe struct {
	ID          string
	KitchenID   int64
	Title       string
	Ingredient  string
	Link        string
	ImageURL    string
	MealType    string
	Status      string
	Note        string
	Ingredients []string
	Steps       []string
	CreatedBy   int64
	UpdatedBy   int64
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(log.Writer(), &slog.HandlerOptions{}))

	dbConn, err := db.Open(cfg, logger)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer dbConn.Close()

	if err := bootstrap.RunMigrations(context.Background(), dbConn, logger, cfg.MigrationDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	if err := seedDemoData(context.Background(), dbConn); err != nil {
		log.Fatalf("seed demo data: %v", err)
	}

	fmt.Println("demo seed complete")
}

func seedDemoData(ctx context.Context, dbConn *sql.DB) error {
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	aliceID, err := ensureUser(ctx, tx, "dev:alice", "alice")
	if err != nil {
		return err
	}

	recipeUserID, err := ensureUser(ctx, tx, "dev:recipe-user", "recipe-user")
	if err != nil {
		return err
	}

	aliceKitchenID, err := ensureKitchen(ctx, tx, aliceID, "我们的厨房")
	if err != nil {
		return err
	}

	weekendKitchenID, err := ensureKitchen(ctx, tx, aliceID, "周末厨房")
	if err != nil {
		return err
	}

	recipeUserKitchenID, err := ensureKitchen(ctx, tx, recipeUserID, "我们的厨房")
	if err != nil {
		return err
	}

	sharedKitchenID, err := ensureKitchen(ctx, tx, aliceID, "联调试吃厨房")
	if err != nil {
		return err
	}

	if err := ensureMembership(ctx, tx, sharedKitchenID, aliceID, "owner"); err != nil {
		return err
	}
	if err := ensureMembership(ctx, tx, sharedKitchenID, recipeUserID, "member"); err != nil {
		return err
	}

	recipes := []demoRecipe{
		{
			ID:          "demo-alice-breakfast-1",
			KitchenID:   aliceKitchenID,
			Title:       "番茄鸡蛋三明治",
			Ingredient:  "吐司、鸡蛋、番茄",
			Link:        "https://example.com/recipes/tomato-egg-sandwich",
			MealType:    "breakfast",
			Status:      "wishlist",
			Note:        "适合工作日早上，十分钟内能完成。",
			Ingredients: []string{"吐司 2片", "鸡蛋 2个", "番茄 1个", "黑胡椒 少许"},
			Steps:       []string{"番茄切丁，鸡蛋打散。", "先炒番茄鸡蛋，再夹进烤过的吐司里。", "吃之前撒一点黑胡椒提味。"},
			CreatedBy:   aliceID,
			UpdatedBy:   aliceID,
		},
		{
			ID:          "demo-alice-main-1",
			KitchenID:   aliceKitchenID,
			Title:       "黑椒土豆牛肉粒",
			Ingredient:  "牛肉、土豆、黑胡椒",
			Link:        "https://example.com/recipes/black-pepper-beef-potato",
			MealType:    "main",
			Status:      "done",
			Note:        "已经做过一版，配米饭很稳。",
			Ingredients: []string{"牛肉粒 250g", "土豆 2个", "黑胡椒碎 适量", "生抽 1勺"},
			Steps:       []string{"牛肉提前腌 15 分钟。", "土豆煎到表面微焦后盛出。", "牛肉回锅和土豆一起翻炒，最后加黑胡椒。"},
			CreatedBy:   aliceID,
			UpdatedBy:   aliceID,
		},
		{
			ID:          "demo-weekend-main-1",
			KitchenID:   weekendKitchenID,
			Title:       "葱油鸡腿焖饭",
			Ingredient:  "鸡腿、香菇、米饭",
			Link:        "https://example.com/recipes/scallion-chicken-rice",
			MealType:    "main",
			Status:      "wishlist",
			Note:        "周末版本，想试试电饭煲一锅出。",
			Ingredients: []string{"鸡腿 2只", "大米 2杯", "香菇 5朵", "小葱 1把"},
			Steps:       []string{"鸡腿去骨切块，先煎出香味。", "和泡好的米、香菇一起进电饭煲。", "出锅前淋葱油再焖 5 分钟。"},
			CreatedBy:   aliceID,
			UpdatedBy:   aliceID,
		},
		{
			ID:          "demo-recipe-user-breakfast-1",
			KitchenID:   recipeUserKitchenID,
			Title:       "蓝莓酸奶碗",
			Ingredient:  "酸奶、蓝莓、燕麦",
			Link:        "https://example.com/recipes/blueberry-yogurt-bowl",
			MealType:    "breakfast",
			Status:      "done",
			Note:        "清爽型早餐，已经吃过一次。",
			Ingredients: []string{"无糖酸奶 1盒", "蓝莓 1把", "燕麦 30g", "坚果 少许"},
			Steps:       []string{"酸奶倒进碗里。", "铺上蓝莓、燕麦和坚果。", "冷藏 10 分钟口感更好。"},
			CreatedBy:   recipeUserID,
			UpdatedBy:   recipeUserID,
		},
		{
			ID:          "demo-recipe-user-main-1",
			KitchenID:   recipeUserKitchenID,
			Title:       "蒜香黄油虾",
			Ingredient:  "虾、黄油、蒜末",
			Link:        "https://example.com/recipes/garlic-butter-shrimp",
			MealType:    "main",
			Status:      "wishlist",
			Note:        "想找一个简单但看起来很像餐厅菜的版本。",
			Ingredients: []string{"鲜虾 400g", "黄油 20g", "蒜末 2勺", "欧芹 少许"},
			Steps:       []string{"虾开背去虾线。", "黄油融化后炒香蒜末。", "下虾快速翻炒到变色，最后撒欧芹。"},
			CreatedBy:   recipeUserID,
			UpdatedBy:   recipeUserID,
		},
		{
			ID:          "demo-shared-breakfast-1",
			KitchenID:   sharedKitchenID,
			Title:       "牛油果煎蛋吐司",
			Ingredient:  "牛油果、鸡蛋、吐司",
			Link:        "https://example.com/recipes/avocado-egg-toast",
			MealType:    "breakfast",
			Status:      "wishlist",
			Note:        "联调用早餐样例，适合测试想吃状态。",
			Ingredients: []string{"吐司 2片", "牛油果 1个", "鸡蛋 1个", "海盐 少许"},
			Steps:       []string{"吐司烤脆。", "牛油果压成泥后抹在吐司上。", "煎一个溏心蛋盖上去。"},
			CreatedBy:   aliceID,
			UpdatedBy:   aliceID,
		},
		{
			ID:          "demo-shared-breakfast-2",
			KitchenID:   sharedKitchenID,
			Title:       "火腿芝士可颂",
			Ingredient:  "可颂、火腿、芝士",
			Link:        "https://example.com/recipes/ham-cheese-croissant",
			MealType:    "breakfast",
			Status:      "done",
			Note:        "联调用早餐样例，适合测试吃过状态。",
			Ingredients: []string{"可颂 1个", "火腿 2片", "芝士 2片", "黄芥末 少许"},
			Steps:       []string{"可颂中间切开。", "夹入火腿和芝士。", "空气炸锅 3 分钟让芝士融化。"},
			CreatedBy:   recipeUserID,
			UpdatedBy:   recipeUserID,
		},
		{
			ID:          "demo-shared-main-1",
			KitchenID:   sharedKitchenID,
			Title:       "番茄滑蛋牛肉",
			Ingredient:  "牛肉、番茄、鸡蛋",
			Link:        "https://example.com/recipes/tomato-egg-beef",
			MealType:    "main",
			Status:      "wishlist",
			Note:        "共享厨房主菜样例之一。",
			Ingredients: []string{"牛里脊 250g", "番茄 2个", "鸡蛋 3个", "生抽 1勺"},
			Steps:       []string{"牛肉切片后简单腌制。", "番茄炒软出汁，鸡蛋单独炒到半熟。", "三者回锅快速翻炒。"},
			CreatedBy:   aliceID,
			UpdatedBy:   recipeUserID,
		},
		{
			ID:          "demo-shared-main-2",
			KitchenID:   sharedKitchenID,
			Title:       "照烧三文鱼饭",
			Ingredient:  "三文鱼、照烧汁、西兰花",
			Link:        "https://example.com/recipes/teriyaki-salmon-bowl",
			MealType:    "main",
			Status:      "done",
			Note:        "共享厨房主菜样例之二。",
			Ingredients: []string{"三文鱼 2块", "照烧汁 2勺", "西兰花 半颗", "米饭 1碗"},
			Steps:       []string{"三文鱼煎到两面上色。", "淋照烧汁收浓。", "搭配焯好的西兰花和热米饭。"},
			CreatedBy:   recipeUserID,
			UpdatedBy:   aliceID,
		},
	}

	for _, item := range recipes {
		if err := upsertRecipe(ctx, tx, item); err != nil {
			return err
		}
	}

	if err := touchKitchen(ctx, tx, weekendKitchenID); err != nil {
		return err
	}
	if err := touchKitchen(ctx, tx, recipeUserKitchenID); err != nil {
		return err
	}
	if err := touchKitchen(ctx, tx, sharedKitchenID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func ensureUser(ctx context.Context, tx *sql.Tx, openID, nickname string) (int64, error) {
	var userID int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE openid = ? LIMIT 1`, openID).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("find user by openid: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO users (openid, nickname, avatar_url, created_at, updated_at) VALUES (?, ?, NULL, ?, ?)`,
		openID,
		nickname,
		now,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}

	userID, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read user id: %w", err)
	}

	return userID, nil
}

func ensureKitchen(ctx context.Context, tx *sql.Tx, ownerUserID int64, name string) (int64, error) {
	var kitchenID int64
	err := tx.QueryRowContext(
		ctx,
		`SELECT id FROM kitchens WHERE owner_user_id = ? AND name = ? ORDER BY id LIMIT 1`,
		ownerUserID,
		name,
	).Scan(&kitchenID)
	if err == nil {
		if err := ensureMembership(ctx, tx, kitchenID, ownerUserID, "owner"); err != nil {
			return 0, err
		}
		return kitchenID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("find kitchen: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO kitchens (name, owner_user_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		name,
		ownerUserID,
		now,
		now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert kitchen: %w", err)
	}

	kitchenID, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read kitchen id: %w", err)
	}

	if err := ensureMembership(ctx, tx, kitchenID, ownerUserID, "owner"); err != nil {
		return 0, err
	}

	return kitchenID, nil
}

func ensureMembership(ctx context.Context, tx *sql.Tx, kitchenID, userID int64, role string) error {
	now := time.Now().Format(time.RFC3339)
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO kitchen_members (kitchen_id, user_id, role, joined_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(kitchen_id, user_id) DO UPDATE SET role = excluded.role`,
		kitchenID,
		userID,
		role,
		now,
	); err != nil {
		return fmt.Errorf("ensure membership: %w", err)
	}

	return nil
}

func upsertRecipe(ctx context.Context, tx *sql.Tx, item demoRecipe) error {
	ingredientsJSON, err := json.Marshal(item.Ingredients)
	if err != nil {
		return fmt.Errorf("marshal recipe ingredients: %w", err)
	}

	stepsJSON, err := json.Marshal(item.Steps)
	if err != nil {
		return fmt.Errorf("marshal recipe steps: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO recipes (
			id, kitchen_id, title, ingredient, link, image_url, meal_type, status, note,
			ingredients_json, steps_json, created_by, updated_by, created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
		ON CONFLICT(id) DO UPDATE SET
			kitchen_id = excluded.kitchen_id,
			title = excluded.title,
			ingredient = excluded.ingredient,
			link = excluded.link,
			image_url = excluded.image_url,
			meal_type = excluded.meal_type,
			status = excluded.status,
			note = excluded.note,
			ingredients_json = excluded.ingredients_json,
			steps_json = excluded.steps_json,
			updated_by = excluded.updated_by,
			updated_at = excluded.updated_at,
			deleted_at = NULL`,
		item.ID,
		item.KitchenID,
		item.Title,
		item.Ingredient,
		nullIfEmpty(item.Link),
		nullIfEmpty(item.ImageURL),
		item.MealType,
		item.Status,
		nullIfEmpty(item.Note),
		string(ingredientsJSON),
		string(stepsJSON),
		item.CreatedBy,
		item.UpdatedBy,
		now,
		now,
	); err != nil {
		return fmt.Errorf("upsert recipe %s: %w", item.ID, err)
	}

	return nil
}

func touchKitchen(ctx context.Context, tx *sql.Tx, kitchenID int64) error {
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE kitchens SET updated_at = ? WHERE id = ?`,
		time.Now().Format(time.RFC3339),
		kitchenID,
	); err != nil {
		return fmt.Errorf("touch kitchen %d: %w", kitchenID, err)
	}

	return nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
