package appsettings

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/bootstrap"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	appdb "github.com/cqh6666/caipu-miniapp/backend/internal/db"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
)

func TestBilibiliSessionOperatorContractWithFullMigrations(t *testing.T) {
	database := openAppSettingsMigrationTestDB(t)
	defer database.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := database.Exec(
		`INSERT INTO users (id, openid, nickname, created_at, updated_at) VALUES (7, 'settings-user', 'Settings User', ?, ?)`,
		now, now,
	); err != nil {
		t.Fatal(err)
	}

	sidecar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/verify/bilibili-session" {
			t.Fatalf("unexpected sidecar path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Bilibili-SESSDATA") == "" {
			t.Fatal("missing bilibili session header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"platform":"bilibili","providerUsed":"openapi","valid":true}`))
	}))
	defer sidecar.Close()
	parser := linkparse.NewService(linkparse.Options{
		LinkparseSidecarEnabled: true,
		LinkparseSidecarBaseURL: sidecar.URL,
	})
	service := NewService(NewRepository(database), testCredentialBox(t, "migration-contract-secret"), parser, func(context.Context, int64) error {
		return nil
	})

	if _, err := service.UpdateBilibiliSession(context.Background(), 7, "user-sessdata"); err != nil {
		t.Fatalf("user update error=%v", err)
	}
	assertBilibiliOperator(t, database, sql.NullInt64{Int64: 7, Valid: true}, "user:7")

	if _, err := service.ClearBilibiliSessionBySubject(context.Background(), "admin:root", nil); err != nil {
		t.Fatalf("admin clear error=%v", err)
	}
	assertBilibiliOperator(t, database, sql.NullInt64{}, "admin:root")

	if _, err := service.UpdateBilibiliSessionBySubject(context.Background(), "admin:root", nil, "admin-sessdata"); err != nil {
		t.Fatalf("admin update error=%v", err)
	}
	assertBilibiliOperator(t, database, sql.NullInt64{}, "admin:root")

	if _, err := service.ClearBilibiliSession(context.Background(), 7); err != nil {
		t.Fatalf("user clear error=%v", err)
	}
	assertBilibiliOperator(t, database, sql.NullInt64{Int64: 7, Valid: true}, "user:7")

	var auditCount int
	if err := database.QueryRow(
		`SELECT COUNT(1) FROM app_setting_audits WHERE setting_key = 'bilibili.session.sessdata'`,
	).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 4 {
		t.Fatalf("audit count=%d, want 4", auditCount)
	}
	foreignKeyRows, err := database.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer foreignKeyRows.Close()
	if foreignKeyRows.Next() {
		t.Fatal("foreign_key_check returned a violation")
	}
	if err := foreignKeyRows.Err(); err != nil {
		t.Fatal(err)
	}
}

func assertBilibiliOperator(t *testing.T, database *sql.DB, wantUserID sql.NullInt64, wantSubject string) {
	t.Helper()
	var userID sql.NullInt64
	var subject string
	if err := database.QueryRow(
		`SELECT updated_by, updated_by_subject FROM app_bilibili_settings WHERE id = 1`,
	).Scan(&userID, &subject); err != nil {
		t.Fatal(err)
	}
	if userID != wantUserID || subject != wantSubject {
		t.Fatalf("operator user=%#v subject=%q, want user=%#v subject=%q", userID, subject, wantUserID, wantSubject)
	}
}

func openAppSettingsMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	dir := t.TempDir()
	database, err := appdb.Open(config.Config{
		SQLitePath:          filepath.Join(dir, "settings.db"),
		SQLiteBusyTimeoutMS: 5000,
		UploadDir:           filepath.Join(dir, "uploads"),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	migrations := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations"))
	if err := bootstrap.RunMigrations(
		context.Background(),
		database,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		migrations,
	); err != nil {
		database.Close()
		t.Fatal(err)
	}
	return database
}
