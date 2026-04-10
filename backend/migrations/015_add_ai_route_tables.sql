CREATE TABLE IF NOT EXISTS ai_route_scenes (
  scene TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 0,
  strategy TEXT NOT NULL DEFAULT 'priority_failover',
  max_attempts INTEGER NOT NULL DEFAULT 2,
  retry_policy_json TEXT NOT NULL DEFAULT '[]',
  breaker_failure_threshold INTEGER NOT NULL DEFAULT 3,
  breaker_cooldown_seconds INTEGER NOT NULL DEFAULT 60,
  request_options_json TEXT NOT NULL DEFAULT '{}',
  updated_by_subject TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS ai_route_providers (
  id TEXT PRIMARY KEY,
  scene TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  adapter TEXT NOT NULL DEFAULT 'openai-compatible',
  enabled INTEGER NOT NULL DEFAULT 1,
  priority INTEGER NOT NULL DEFAULT 10,
  weight INTEGER NOT NULL DEFAULT 100,
  base_url TEXT NOT NULL DEFAULT '',
  api_key_ciphertext TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  timeout_seconds INTEGER NOT NULL DEFAULT 30,
  extra_json TEXT NOT NULL DEFAULT '{}',
  updated_by_subject TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(scene) REFERENCES ai_route_scenes(scene) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ai_route_providers_scene_enabled_priority
  ON ai_route_providers(scene, enabled, priority ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_ai_route_providers_scene_id
  ON ai_route_providers(scene, id);
