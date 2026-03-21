ALTER TABLE recipes ADD COLUMN flowchart_image_url TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN flowchart_updated_at TEXT;
ALTER TABLE recipes ADD COLUMN flowchart_source_hash TEXT NOT NULL DEFAULT '';
