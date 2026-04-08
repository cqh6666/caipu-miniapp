CREATE TABLE IF NOT EXISTS ai_job_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scene TEXT NOT NULL,
  target_type TEXT NOT NULL DEFAULT '',
  target_id TEXT NOT NULL DEFAULT '',
  trigger_source TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '',
  final_provider TEXT NOT NULL DEFAULT '',
  final_model TEXT NOT NULL DEFAULT '',
  fallback_used INTEGER NOT NULL DEFAULT 0,
  error_message TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL DEFAULT '',
  duration_ms INTEGER NOT NULL DEFAULT 0,
  meta_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_ai_job_runs_scene_status_started_at
  ON ai_job_runs(scene, status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_job_runs_target_started_at
  ON ai_job_runs(target_type, target_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_job_runs_request_id
  ON ai_job_runs(request_id);

CREATE TABLE IF NOT EXISTS ai_call_logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_run_id INTEGER,
  scene TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  endpoint TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '',
  http_status INTEGER NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  error_type TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  meta_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  FOREIGN KEY (job_run_id) REFERENCES ai_job_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_ai_call_logs_scene_status_created_at
  ON ai_call_logs(scene, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_call_logs_job_run_id
  ON ai_call_logs(job_run_id);
CREATE INDEX IF NOT EXISTS idx_ai_call_logs_provider_model_created_at
  ON ai_call_logs(provider, model, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_call_logs_request_id
  ON ai_call_logs(request_id);

CREATE TABLE IF NOT EXISTS app_runtime_settings (
  key TEXT PRIMARY KEY,
  group_name TEXT NOT NULL,
  value_text TEXT NOT NULL DEFAULT '',
  value_ciphertext TEXT NOT NULL DEFAULT '',
  value_type TEXT NOT NULL DEFAULT 'string',
  is_secret INTEGER NOT NULL DEFAULT 0,
  is_restart_required INTEGER NOT NULL DEFAULT 0,
  description TEXT NOT NULL DEFAULT '',
  updated_by_subject TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_app_runtime_settings_group_name
  ON app_runtime_settings(group_name);

CREATE TABLE IF NOT EXISTS app_setting_audits (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  group_name TEXT NOT NULL DEFAULT '',
  setting_key TEXT NOT NULL,
  action TEXT NOT NULL,
  old_value_masked TEXT NOT NULL DEFAULT '',
  new_value_masked TEXT NOT NULL DEFAULT '',
  operator_subject TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_app_setting_audits_group_created_at
  ON app_setting_audits(group_name, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_app_setting_audits_key_created_at
  ON app_setting_audits(setting_key, created_at DESC);
