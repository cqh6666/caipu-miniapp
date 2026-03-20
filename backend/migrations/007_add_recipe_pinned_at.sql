ALTER TABLE recipes ADD COLUMN pinned_at TEXT;

CREATE INDEX IF NOT EXISTS idx_recipes_pinned_at ON recipes(pinned_at);
