package appsettings

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	_ "modernc.org/sqlite"
)

func TestRuntimeProviderUpdateRuntimeGroupPrefersExplicitSecretValueOverClear(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	ctx := context.Background()

	if _, err := provider.UpdateRuntimeGroup(ctx, "tester", "req-1", "ai.summary", map[string]any{
		"api_key": "old-secret",
	}, nil); err != nil {
		t.Fatalf("initial UpdateRuntimeGroup() error = %v", err)
	}

	if _, err := provider.UpdateRuntimeGroup(ctx, "tester", "req-2", "ai.summary", map[string]any{
		"api_key": "new-secret",
	}, []string{"api_key"}); err != nil {
		t.Fatalf("conflicting UpdateRuntimeGroup() error = %v", err)
	}

	if got := provider.SummaryAI(ctx).APIKey; got != "new-secret" {
		t.Fatalf("provider.SummaryAI().APIKey = %q, want %q", got, "new-secret")
	}
}

func TestRuntimeProviderTestRuntimeGroupPrefersExplicitValueOverClear(t *testing.T) {
	t.Parallel()

	provider := newRuntimeProviderForTest(t)
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path = %q", r.URL.Path)
		}
		if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "Bearer live-secret" {
			t.Fatalf("Authorization header = %q, want %q", got, "Bearer live-secret")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"test"}`))
	}))
	defer server.Close()

	result, err := provider.TestRuntimeGroup(ctx, "tester", "req-3", "ai.summary", map[string]any{
		"base_url": server.URL,
		"model":    "gpt-test",
		"api_key":  "live-secret",
	}, []string{"base_url", "model", "api_key"})
	if err != nil {
		t.Fatalf("TestRuntimeGroup() error = %v", err)
	}
	if !result.OK {
		t.Fatalf("TestRuntimeGroup().OK = false, message = %q", result.Message)
	}
}

func newRuntimeProviderForTest(t *testing.T) *RuntimeProvider {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	statements := []string{
		`CREATE TABLE app_runtime_settings (
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
		);`,
		`CREATE TABLE app_setting_audits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			group_name TEXT NOT NULL DEFAULT '',
			setting_key TEXT NOT NULL,
			action TEXT NOT NULL,
			old_value_masked TEXT NOT NULL DEFAULT '',
			new_value_masked TEXT NOT NULL DEFAULT '',
			operator_subject TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("db.Exec(%q) error = %v", statement, err)
		}
	}

	repo := NewRepository(db)
	return NewRuntimeProvider(repo, "test-secret", config.Config{
		AIBaseURL:                  "https://default.example.com/v1",
		AIModel:                    "default-model",
		AITimeoutSeconds:           30,
		AIFlowchartTimeoutSeconds:  45,
		AITitleTimeoutSeconds:      3,
		AITitleMaxTokens:           64,
		LinkparseSidecarTimeoutSec: 150,
	})
}
