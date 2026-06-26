ALTER TABLE recipes ADD COLUMN done_at TEXT NOT NULL DEFAULT '';

ALTER TABLE places ADD COLUMN price_amount_cents INTEGER NOT NULL DEFAULT 0;
ALTER TABLE places ADD COLUMN price_currency TEXT NOT NULL DEFAULT 'CNY';
ALTER TABLE places ADD COLUMN price_type TEXT NOT NULL DEFAULT '';

UPDATE places
   SET price_amount_cents = CAST(ROUND(CAST(TRIM(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(price, '￥', ''), '¥', ''), '人民币', ''), '人均', ''), '/人', ''), '每人', ''), '元', ''), '约', ''), '左右', '')) AS REAL) * 100) AS INTEGER),
       price_currency = 'CNY',
       price_type = CASE
         WHEN price LIKE '%/人%' OR price LIKE '%人均%' OR price LIKE '%每人%' THEN 'per_person'
         ELSE 'amount'
       END
 WHERE COALESCE(TRIM(price), '') <> ''
   AND CAST(TRIM(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(price, '￥', ''), '¥', ''), '人民币', ''), '人均', ''), '/人', ''), '每人', ''), '元', ''), '约', ''), '左右', '')) AS REAL) > 0;

UPDATE recipes
   SET done_at = COALESCE(NULLIF(updated_at, ''), created_at)
 WHERE status = 'done'
   AND COALESCE(done_at, '') = '';

CREATE TABLE IF NOT EXISTS recipe_status_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kitchen_id INTEGER NOT NULL,
  recipe_id TEXT NOT NULL,
  from_status TEXT NOT NULL DEFAULT '',
  to_status TEXT NOT NULL,
  changed_by INTEGER NOT NULL DEFAULT 0,
  changed_at TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'api',
  FOREIGN KEY (kitchen_id) REFERENCES kitchens(id),
  FOREIGN KEY (recipe_id) REFERENCES recipes(id),
  FOREIGN KEY (changed_by) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS place_status_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  kitchen_id INTEGER NOT NULL,
  place_id TEXT NOT NULL,
  from_status TEXT NOT NULL DEFAULT '',
  to_status TEXT NOT NULL,
  changed_by INTEGER NOT NULL DEFAULT 0,
  changed_at TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'api',
  FOREIGN KEY (kitchen_id) REFERENCES kitchens(id),
  FOREIGN KEY (place_id) REFERENCES places(id),
  FOREIGN KEY (changed_by) REFERENCES users(id)
);

INSERT INTO recipe_status_events (
  kitchen_id, recipe_id, from_status, to_status, changed_by, changed_at, source
)
SELECT
  kitchen_id,
  id,
  '',
  status,
  updated_by,
  COALESCE(NULLIF(done_at, ''), NULLIF(updated_at, ''), created_at),
  'migration'
FROM recipes
WHERE deleted_at IS NULL
  AND COALESCE(status, '') <> '';

INSERT INTO place_status_events (
  kitchen_id, place_id, from_status, to_status, changed_by, changed_at, source
)
SELECT
  kitchen_id,
  id,
  '',
  status,
  updated_by,
  COALESCE(NULLIF(visited_at, ''), NULLIF(updated_at, ''), created_at),
  'migration'
FROM places
WHERE deleted_at IS NULL
  AND COALESCE(status, '') <> '';

CREATE INDEX IF NOT EXISTS idx_recipes_kitchen_status_created
  ON recipes(kitchen_id, status, created_at);

CREATE INDEX IF NOT EXISTS idx_recipes_kitchen_done_at
  ON recipes(kitchen_id, done_at);

CREATE INDEX IF NOT EXISTS idx_places_kitchen_created
  ON places(kitchen_id, created_at);

CREATE INDEX IF NOT EXISTS idx_places_kitchen_visited_at
  ON places(kitchen_id, visited_at);

CREATE INDEX IF NOT EXISTS idx_places_kitchen_price_amount
  ON places(kitchen_id, price_amount_cents);

CREATE INDEX IF NOT EXISTS idx_recipe_status_events_kitchen_changed
  ON recipe_status_events(kitchen_id, changed_at);

CREATE INDEX IF NOT EXISTS idx_recipe_status_events_kitchen_to_changed
  ON recipe_status_events(kitchen_id, to_status, changed_at);

CREATE INDEX IF NOT EXISTS idx_recipe_status_events_recipe_changed
  ON recipe_status_events(recipe_id, changed_at);

CREATE INDEX IF NOT EXISTS idx_place_status_events_kitchen_changed
  ON place_status_events(kitchen_id, changed_at);

CREATE INDEX IF NOT EXISTS idx_place_status_events_kitchen_to_changed
  ON place_status_events(kitchen_id, to_status, changed_at);

CREATE INDEX IF NOT EXISTS idx_place_status_events_place_changed
  ON place_status_events(place_id, changed_at);
