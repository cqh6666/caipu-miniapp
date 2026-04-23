ALTER TABLE recipes ADD COLUMN share_token TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN share_token_created_at TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_recipes_share_token ON recipes(share_token) WHERE share_token != '';
