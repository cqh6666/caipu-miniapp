ALTER TABLE recipes ADD COLUMN content_version INTEGER NOT NULL DEFAULT 0;

ALTER TABLE recipes ADD COLUMN parse_claim_token TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN parse_claim_content_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE recipes ADD COLUMN parse_lease_expires_at TEXT NOT NULL DEFAULT '';

ALTER TABLE recipes ADD COLUMN flowchart_claim_token TEXT NOT NULL DEFAULT '';
ALTER TABLE recipes ADD COLUMN flowchart_claim_content_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE recipes ADD COLUMN flowchart_lease_expires_at TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_recipes_parse_claim
  ON recipes(parse_status, parse_lease_expires_at);

CREATE INDEX IF NOT EXISTS idx_recipes_flowchart_claim
  ON recipes(flowchart_status, flowchart_lease_expires_at);
