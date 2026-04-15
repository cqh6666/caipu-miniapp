CREATE TABLE IF NOT EXISTS ai_provider_alert_states (
  provider_id TEXT PRIMARY KEY,
  scene TEXT NOT NULL DEFAULT '',
  provider_name TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  consecutive_failures INTEGER NOT NULL DEFAULT 0,
  last_status TEXT NOT NULL DEFAULT '',
  last_error_type TEXT NOT NULL DEFAULT '',
  last_error_message TEXT NOT NULL DEFAULT '',
  last_http_status INTEGER NOT NULL DEFAULT 0,
  last_request_id TEXT NOT NULL DEFAULT '',
  last_failed_at TEXT NOT NULL DEFAULT '',
  last_recovered_at TEXT NOT NULL DEFAULT '',
  last_alerted_at TEXT NOT NULL DEFAULT '',
  last_alerted_failure_count INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_ai_provider_alert_states_updated_at
  ON ai_provider_alert_states(updated_at DESC);
