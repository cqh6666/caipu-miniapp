CREATE TABLE IF NOT EXISTS meal_plans (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kitchen_id INTEGER NOT NULL,
  plan_date TEXT NOT NULL,
  status TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  created_by INTEGER NOT NULL,
  updated_by INTEGER NOT NULL,
  submitted_by INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  submitted_at TEXT NOT NULL DEFAULT '',
  UNIQUE(kitchen_id, plan_date, status),
  FOREIGN KEY (kitchen_id) REFERENCES kitchens(id)
);

CREATE TABLE IF NOT EXISTS meal_plan_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  plan_id INTEGER NOT NULL,
  recipe_id TEXT NOT NULL,
  quantity INTEGER NOT NULL DEFAULT 1,
  meal_type_snapshot TEXT NOT NULL DEFAULT 'main',
  title_snapshot TEXT NOT NULL DEFAULT '',
  image_snapshot TEXT NOT NULL DEFAULT '',
  sort_index INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(plan_id, recipe_id),
  FOREIGN KEY (plan_id) REFERENCES meal_plans(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_meal_plans_kitchen_status_plan_date
  ON meal_plans(kitchen_id, status, plan_date);

CREATE INDEX IF NOT EXISTS idx_meal_plan_items_plan_sort
  ON meal_plan_items(plan_id, sort_index, id);
