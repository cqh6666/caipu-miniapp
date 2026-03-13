ALTER TABLE recipes ADD COLUMN parse_status TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN parse_source TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN parse_error TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN parse_requested_at TEXT;
ALTER TABLE recipes ADD COLUMN parse_finished_at TEXT;

CREATE INDEX IF NOT EXISTS idx_recipes_parse_status_requested_at ON recipes(parse_status, parse_requested_at);
