package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func TestAppRouterAuthenticationBoundaries(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler

	for _, path := range []string{"/healthz", "/api/healthz", "/api/public/app-config"} {
		response := serveIntegrationRequest(handler, http.MethodGet, path, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}

	for _, path := range []string{"/api/auth/me", "/api/kitchens", "/api/admin/dashboard/overview"} {
		response := serveIntegrationRequest(handler, http.MethodGet, path, nil)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous GET %s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}

	devLogin := serveIntegrationRequest(
		handler,
		http.MethodPost,
		"/api/auth/dev-login",
		[]byte(`{"identity":"integration-user"}`),
	)
	if devLogin.Code != http.StatusOK {
		t.Fatalf("dev login status=%d body=%s", devLogin.Code, devLogin.Body.String())
	}
	var loginPayload struct {
		Data struct {
			Token            string `json:"token"`
			CurrentKitchenID int64  `json:"currentKitchenId"`
		} `json:"data"`
	}
	if err := json.NewDecoder(devLogin.Body).Decode(&loginPayload); err != nil {
		t.Fatal(err)
	}
	if loginPayload.Data.Token == "" || loginPayload.Data.CurrentKitchenID <= 0 {
		t.Fatalf("invalid dev login response: %#v", loginPayload)
	}

	for _, path := range []string{"/api/auth/me", "/api/kitchens"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+loginPayload.Data.Token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("authenticated GET %s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}

	adminLogin := serveIntegrationRequest(
		handler,
		http.MethodPost,
		"/api/admin/auth/login",
		[]byte(`{"username":"admin","password":"admin-secret"}`),
	)
	if adminLogin.Code != http.StatusOK {
		t.Fatalf("admin login status=%d body=%s", adminLogin.Code, adminLogin.Body.String())
	}
	cookies := adminLogin.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Value == "" {
		t.Fatal("admin login did not issue a session cookie")
	}
	request := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/overview", nil)
	request.AddCookie(cookies[0])
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("admin dashboard status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAppRouterDoesNotExposeDevLoginOutsideLocal(t *testing.T) {
	application := newIntegrationApp(t, "production")
	response := serveIntegrationRequest(
		application.Server.Handler,
		http.MethodPost,
		"/api/auth/dev-login",
		[]byte(`{"identity":"integration-user"}`),
	)
	if response.Code != http.StatusNotFound {
		t.Fatalf("production dev-login status=%d body=%s", response.Code, response.Body.String())
	}
}

func newIntegrationApp(t *testing.T, appEnv string) *App {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test path")
	}
	migrationDir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations"))
	dir := t.TempDir()
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin-secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		AppName:                        "caipu-integration-test",
		AppEnv:                         appEnv,
		AppAddr:                        ":0",
		LogLevel:                       "error",
		AdminUsername:                  "admin",
		AdminPasswordHash:              string(passwordHash),
		AdminJWTSecret:                 "admin-jwt-secret",
		AppSettingsAccessMode:          "all",
		CredentialsSecret:              "credentials-secret",
		JWTSecret:                      "user-jwt-secret",
		JWTExpireHours:                 1,
		AITimeoutSeconds:               1,
		AIFlowchartTimeoutSeconds:      1,
		AITitleMaxTokens:               64,
		AITitleTimeoutSeconds:          1,
		AIAlertFailureThreshold:        3,
		AIAlertSMTPPort:                587,
		DietAssistantAITimeoutSec:      1,
		LinkparseSidecarTimeoutSec:     1,
		AMapPlacePreviewTimeoutSeconds: 1,
		AMapPlacePreviewMaxAttempts:    1,
		SQLitePath:                     filepath.Join(dir, "app.db"),
		SQLiteBusyTimeoutMS:            1000,
		MigrationDir:                   migrationDir,
		UploadDir:                      filepath.Join(dir, "uploads"),
		UploadMaxImageMB:               1,
		InviteDefaultExpireHours:       1,
		InviteDefaultMaxUses:           1,
		RecipeAutoParseInterval:        1,
		RecipeAutoParseBatchSize:       1,
		RecipeAutoParseMaxAttempts:     1,
		RecipeAutoParseRetryBaseSec:    1,
		RecipeAutoParseStaleSec:        1,
		RecipeFlowchartInterval:        1,
		RecipeFlowchartBatchSize:       1,
		RecipeImageMirrorInterval:      1,
		RecipeImageMirrorBatchSize:     1,
	}
	application, err := New(cfg)
	if err != nil {
		t.Fatalf("create integration app: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := application.Shutdown(ctx); err != nil {
			t.Errorf("shutdown integration app: %v", err)
		}
	})
	return application
}

func serveIntegrationRequest(handler http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
