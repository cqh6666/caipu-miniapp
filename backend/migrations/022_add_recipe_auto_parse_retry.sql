ALTER TABLE recipes ADD COLUMN title_source TEXT NOT NULL DEFAULT 'manual';
ALTER TABLE recipes ADD COLUMN parse_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE recipes ADD COLUMN parse_next_attempt_at TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN parse_last_error_type TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN parse_processing_started_at TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_recipes_parse_status_next_attempt
  ON recipes(parse_status, parse_next_attempt_at, parse_requested_at);
