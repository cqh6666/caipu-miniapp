CREATE TABLE IF NOT EXISTS app_bilibili_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  sessdata_ciphertext TEXT NOT NULL DEFAULT '',
  masked_sessdata TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'unconfigured',
  last_checked_at TEXT,
  last_success_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
  updated_by INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (updated_by) REFERENCES users(id)
);
