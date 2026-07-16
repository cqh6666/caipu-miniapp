ALTER TABLE ai_route_scenes
  ADD COLUMN version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1);

CREATE TABLE IF NOT EXISTS app_runtime_setting_groups (
  group_name TEXT PRIMARY KEY,
  version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
  updated_by_subject TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);

INSERT OR IGNORE INTO app_runtime_setting_groups (
  group_name,
  version,
  updated_by_subject,
  updated_at
)
SELECT
  group_name,
  1,
  MAX(COALESCE(updated_by_subject, '')),
  MAX(COALESCE(updated_at, ''))
FROM app_runtime_settings
GROUP BY group_name;

INSERT OR IGNORE INTO app_runtime_setting_groups (
  group_name,
  version,
  updated_by_subject,
  updated_at
)
SELECT
  'bilibili.session',
  1,
  COALESCE(updated_by_subject, ''),
  COALESCE(updated_at, '')
FROM app_bilibili_settings
WHERE id = 1;

ALTER TABLE ai_provider_alert_states
  ADD COLUMN failure_streak_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS ai_provider_alert_deliveries (
  event_id TEXT PRIMARY KEY,
  failure_streak_id TEXT NOT NULL UNIQUE,
  provider_id TEXT NOT NULL,
  scene TEXT NOT NULL DEFAULT '',
  trigger_source TEXT NOT NULL DEFAULT '',
  target_type TEXT NOT NULL DEFAULT '',
  target_id TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'sending', 'sent', 'cancelled')),
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  claim_token TEXT NOT NULL DEFAULT '',
  claim_expires_at TEXT NOT NULL DEFAULT '',
  available_at TEXT NOT NULL,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  sent_at TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (provider_id) REFERENCES ai_provider_alert_states(provider_id)
);

CREATE INDEX IF NOT EXISTS idx_ai_provider_alert_deliveries_claim
  ON ai_provider_alert_deliveries(status, available_at, claim_expires_at, created_at);

CREATE INDEX IF NOT EXISTS idx_ai_provider_alert_deliveries_provider
  ON ai_provider_alert_deliveries(provider_id, created_at DESC);
