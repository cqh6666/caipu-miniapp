ALTER TABLE recipes ADD COLUMN flowchart_status TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN flowchart_error TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN flowchart_requested_at TEXT;
ALTER TABLE recipes ADD COLUMN flowchart_finished_at TEXT;
