package appsettings

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	_ "modernc.org/sqlite"
)

func TestRepositorySaveBilibiliSessionRejectsStaleVersion(t *testing.T) {
	provider := newRuntimeProviderForTest(t)
	repo := provider.repo
	if _, err := repo.db.Exec(`
CREATE TABLE app_bilibili_settings (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	sessdata_ciphertext TEXT NOT NULL DEFAULT '',
	masked_sessdata TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'unconfigured',
	last_checked_at TEXT,
	last_success_at TEXT,
	last_error TEXT NOT NULL DEFAULT '',
	updated_by INTEGER,
	updated_by_subject TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL DEFAULT ''
);`); err != nil {
		t.Fatal(err)
	}
	expectedVersion := 0
	first := bilibiliSessionRecord{
		SessdataCiphertext: "cipher-a",
		MaskedSessdata:     "mask-a",
		Status:             BilibiliSessionStatusValid,
		UpdatedBySubject:   "admin:a",
		UpdatedAt:          "2026-07-16T01:00:00Z",
	}
	version, err := repo.SaveBilibiliSessionWithAudit(context.Background(), first, settingAuditRecord{
		GroupName:       "bilibili.session",
		SettingKey:      "bilibili.session.sessdata",
		Action:          "update",
		OperatorSubject: "admin:a",
		CreatedAt:       first.UpdatedAt,
	}, &expectedVersion)
	if err != nil {
		t.Fatalf("first SaveBilibiliSessionWithAudit() error = %v", err)
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}

	second := first
	second.SessdataCiphertext = "cipher-b"
	second.MaskedSessdata = "mask-b"
	second.UpdatedBySubject = "admin:b"
	second.UpdatedAt = "2026-07-16T01:01:00Z"
	_, err = repo.SaveBilibiliSessionWithAudit(context.Background(), second, settingAuditRecord{
		GroupName:       "bilibili.session",
		SettingKey:      "bilibili.session.sessdata",
		Action:          "update",
		OperatorSubject: "admin:b",
		CreatedAt:       second.UpdatedAt,
	}, &expectedVersion)
	var appErr *common.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusConflict {
		t.Fatalf("stale SaveBilibiliSessionWithAudit() error = %v, want 409", err)
	}
	record, err := repo.GetBilibiliSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if record.SessdataCiphertext != "cipher-a" || record.Version != 1 {
		t.Fatalf("record after conflict = %#v", record)
	}
}

func TestNormalizeSessdata(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "raw sessdata",
			input: "abc123",
			want:  "abc123",
		},
		{
			name:  "cookie header",
			input: "foo=1; SESSDATA=abc123%2C456; bar=2",
			want:  "abc123%2C456",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeSessdata(test.input)
			if err != nil {
				t.Fatalf("normalizeSessdata() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("normalizeSessdata() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRepositorySaveBilibiliSessionRollsBackWhenAuditFails(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`
CREATE TABLE app_bilibili_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  sessdata_ciphertext TEXT NOT NULL DEFAULT '',
  masked_sessdata TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'unconfigured',
  last_checked_at TEXT,
  last_success_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
  updated_by INTEGER,
  updated_by_subject TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE app_setting_audits (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  group_name TEXT NOT NULL,
  setting_key TEXT NOT NULL,
  action TEXT NOT NULL,
  old_value_masked TEXT NOT NULL DEFAULT '',
  new_value_masked TEXT NOT NULL DEFAULT '',
  operator_subject TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE app_runtime_setting_groups (
  group_name TEXT PRIMARY KEY,
  version INTEGER NOT NULL DEFAULT 1,
  updated_by_subject TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);
INSERT INTO app_bilibili_settings (
  id, sessdata_ciphertext, masked_sessdata, status, updated_by_subject, updated_at
) VALUES (1, 'old-cipher', 'old-mask', 'valid', 'legacy', '2026-07-16T00:00:00Z');
CREATE TRIGGER reject_bilibili_audit
BEFORE INSERT ON app_setting_audits
BEGIN
  SELECT RAISE(ABORT, 'audit insert rejected');
END;
`); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(db)
	_, err = repo.SaveBilibiliSessionWithAudit(context.Background(), bilibiliSessionRecord{
		SessdataCiphertext: "new-cipher",
		MaskedSessdata:     "new-mask",
		Status:             BilibiliSessionStatusValid,
		UpdatedBySubject:   "admin:root",
		UpdatedAt:          "2026-07-16T00:01:00Z",
	}, settingAuditRecord{
		GroupName:       "bilibili.session",
		SettingKey:      "bilibili.session.sessdata",
		Action:          "update",
		NewValueMasked:  "new-mask",
		OperatorSubject: "admin:root",
		CreatedAt:       "2026-07-16T00:01:00Z",
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "audit insert rejected") {
		t.Fatalf("SaveBilibiliSessionWithAudit() error=%v", err)
	}
	record, err := repo.GetBilibiliSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if record.SessdataCiphertext != "old-cipher" || record.MaskedSessdata != "old-mask" {
		t.Fatalf("configuration was not rolled back: %#v", record)
	}
	var auditCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM app_setting_audits`).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 0 {
		t.Fatalf("auditCount=%d, want=0", auditCount)
	}
}

func TestCipherBoxRoundTrip(t *testing.T) {
	box := newCipherBox("test-secret")
	plain := "abc123%2C456"

	ciphertext, err := box.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if ciphertext == plain {
		t.Fatalf("Encrypt() returned plaintext")
	}

	got, err := box.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if got != plain {
		t.Fatalf("Decrypt() = %q, want %q", got, plain)
	}
}

func TestRepositoryGetBilibiliSessionHandlesNulls(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
CREATE TABLE app_bilibili_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  sessdata_ciphertext TEXT NOT NULL DEFAULT '',
  masked_sessdata TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'unconfigured',
  last_checked_at TEXT,
  last_success_at TEXT,
  last_error TEXT NOT NULL DEFAULT '',
  updated_by INTEGER,
  updated_by_subject TEXT NOT NULL,
  updated_at TEXT NOT NULL DEFAULT ''
);
CREATE TABLE app_runtime_setting_groups (
  group_name TEXT PRIMARY KEY,
  version INTEGER NOT NULL DEFAULT 1,
  updated_by_subject TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);
INSERT INTO app_bilibili_settings (id, sessdata_ciphertext, masked_sessdata, status, last_checked_at, last_success_at, last_error, updated_by, updated_by_subject, updated_at)
VALUES (1, '', '', 'unconfigured', NULL, NULL, '', NULL, 'legacy', '2026-03-13T19:07:43Z');
`); err != nil {
		t.Fatalf("setup db error = %v", err)
	}

	repo := NewRepository(db)
	record, err := repo.GetBilibiliSession(context.Background())
	if err != nil {
		t.Fatalf("GetBilibiliSession() error = %v", err)
	}
	if record.LastCheckedAt != "" || record.LastSuccessAt != "" {
		t.Fatalf("expected empty nullable fields, got lastCheckedAt=%q lastSuccessAt=%q", record.LastCheckedAt, record.LastSuccessAt)
	}
}
