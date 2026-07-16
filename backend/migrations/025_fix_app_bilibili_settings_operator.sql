CREATE TABLE app_bilibili_settings_v2 (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  sessdata_ciphertext TEXT NOT NULL DEFAULT '',
  masked_sessdata TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'unconfigured',
  last_checked_at TEXT,
  last_success_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
  updated_by INTEGER,
  updated_by_subject TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (updated_by) REFERENCES users(id)
);

INSERT INTO app_bilibili_settings_v2 (
  id,
  sessdata_ciphertext,
  masked_sessdata,
  status,
  last_checked_at,
  last_success_at,
  last_error,
  updated_by,
  updated_by_subject,
  updated_at
)
SELECT
  id,
  sessdata_ciphertext,
  masked_sessdata,
  status,
  last_checked_at,
  last_success_at,
  last_error,
  NULLIF(updated_by, 0),
  CASE
    WHEN updated_by > 0 THEN 'user:' || updated_by
    ELSE 'legacy'
  END,
  updated_at
FROM app_bilibili_settings;

DROP TABLE app_bilibili_settings;
ALTER TABLE app_bilibili_settings_v2 RENAME TO app_bilibili_settings;
