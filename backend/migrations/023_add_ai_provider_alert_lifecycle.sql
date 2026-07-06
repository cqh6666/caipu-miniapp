-- 告警状态生命周期扩展：归档 / 静默 / 配置变更标记，以及处置事件表。
-- 关联需求：docs/admin-ai-provider-alert-state-ux-design-2026-07-06.md

ALTER TABLE ai_provider_alert_states ADD COLUMN archived_at TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_provider_alert_states ADD COLUMN archived_by TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_provider_alert_states ADD COLUMN archive_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_provider_alert_states ADD COLUMN muted_until TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_provider_alert_states ADD COLUMN muted_by TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_provider_alert_states ADD COLUMN mute_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE ai_provider_alert_states ADD COLUMN last_config_changed_at TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS ai_provider_alert_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  provider_id TEXT NOT NULL,
  scene TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  operator_subject TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_provider_alert_events_provider
  ON ai_provider_alert_events(provider_id, created_at DESC);
