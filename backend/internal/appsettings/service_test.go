package appsettings

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

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
  updated_by INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT ''
);
INSERT INTO app_bilibili_settings (id, sessdata_ciphertext, masked_sessdata, status, last_checked_at, last_success_at, last_error, updated_by, updated_at)
VALUES (1, '', '', 'unconfigured', NULL, NULL, '', 0, '2026-03-13T19:07:43Z');
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
