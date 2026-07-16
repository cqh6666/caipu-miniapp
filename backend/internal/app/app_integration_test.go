package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/admin"
	"github.com/cqh6666/caipu-miniapp/backend/internal/auth"
	"github.com/cqh6666/caipu-miniapp/backend/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func TestAppRouterAuthenticationBoundaries(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler

	for _, path := range []string{"/livez", "/readyz", "/healthz", "/api/healthz", "/api/public/app-config"} {
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
	userTokenOnAdmin := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/overview", nil)
	userTokenOnAdmin.Header.Set("Authorization", "Bearer "+loginPayload.Data.Token)
	userTokenOnAdminResponse := httptest.NewRecorder()
	handler.ServeHTTP(userTokenOnAdminResponse, userTokenOnAdmin)
	if userTokenOnAdminResponse.Code != http.StatusUnauthorized {
		t.Fatalf("user token on admin API status=%d body=%s", userTokenOnAdminResponse.Code, userTokenOnAdminResponse.Body.String())
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

	adminTokenOnUser := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	adminTokenOnUser.Header.Set("Authorization", "Bearer "+cookies[0].Value)
	adminTokenOnUserResponse := httptest.NewRecorder()
	handler.ServeHTTP(adminTokenOnUserResponse, adminTokenOnUser)
	if adminTokenOnUserResponse.Code != http.StatusUnauthorized {
		t.Fatalf("admin token on user API status=%d body=%s", adminTokenOnUserResponse.Code, adminTokenOnUserResponse.Body.String())
	}

	adminBearerOnAdmin := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard/overview", nil)
	adminBearerOnAdmin.Header.Set("Authorization", "Bearer "+cookies[0].Value)
	adminBearerOnAdminResponse := httptest.NewRecorder()
	handler.ServeHTTP(adminBearerOnAdminResponse, adminBearerOnAdmin)
	if adminBearerOnAdminResponse.Code != http.StatusUnauthorized {
		t.Fatalf("admin bearer on admin API status=%d body=%s", adminBearerOnAdminResponse.Code, adminBearerOnAdminResponse.Body.String())
	}
}

func TestProfileKitchenAndInviteHTTPContractsAcrossRealSQLiteMigrations(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler
	login := func(identity string) (string, int64) {
		response := serveIntegrationRequest(
			handler,
			http.MethodPost,
			"/api/auth/dev-login",
			[]byte(fmt.Sprintf(`{"identity":%q}`, identity)),
		)
		if response.Code != http.StatusOK {
			t.Fatalf("login %s status=%d body=%s", identity, response.Code, response.Body.String())
		}
		var payload struct {
			Data struct {
				Token            string `json:"token"`
				CurrentKitchenID int64  `json:"currentKitchenId"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		return payload.Data.Token, payload.Data.CurrentKitchenID
	}

	ownerToken, defaultKitchenID := login("contract-owner")
	if ownerToken == "" || defaultKitchenID <= 0 {
		t.Fatalf("invalid owner session: token=%q kitchen=%d", ownerToken, defaultKitchenID)
	}

	profile := serveAuthenticatedIntegrationRequest(
		handler,
		http.MethodPatch,
		"/api/auth/profile",
		[]byte(`{"nickname":"契约用户","avatarUrl":"https://cdn.example.com/avatar.png"}`),
		ownerToken,
	)
	if profile.Code != http.StatusOK || !strings.Contains(profile.Body.String(), `"nickname":"契约用户"`) {
		t.Fatalf("profile status=%d body=%s", profile.Code, profile.Body.String())
	}

	createdKitchen := serveAuthenticatedIntegrationRequest(
		handler,
		http.MethodPost,
		"/api/kitchens",
		[]byte(`{"name":"契约空间"}`),
		ownerToken,
	)
	if createdKitchen.Code != http.StatusCreated {
		t.Fatalf("create kitchen status=%d body=%s", createdKitchen.Code, createdKitchen.Body.String())
	}
	var kitchenPayload struct {
		Data struct {
			Kitchen struct {
				ID int64 `json:"id"`
			} `json:"kitchen"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createdKitchen.Body).Decode(&kitchenPayload); err != nil {
		t.Fatal(err)
	}
	contractKitchenID := kitchenPayload.Data.Kitchen.ID
	if contractKitchenID <= 0 {
		t.Fatalf("invalid created kitchen payload: %#v", kitchenPayload)
	}

	updatedKitchen := serveAuthenticatedIntegrationRequest(
		handler,
		http.MethodPatch,
		fmt.Sprintf("/api/kitchens/%d", contractKitchenID),
		[]byte(`{"name":"契约空间（已改名）"}`),
		ownerToken,
	)
	if updatedKitchen.Code != http.StatusOK || !strings.Contains(updatedKitchen.Body.String(), "已改名") {
		t.Fatalf("update kitchen status=%d body=%s", updatedKitchen.Code, updatedKitchen.Body.String())
	}

	createdInvite := serveAuthenticatedIntegrationRequest(
		handler,
		http.MethodPost,
		fmt.Sprintf("/api/kitchens/%d/invites", contractKitchenID),
		[]byte(`{"maxUses":2,"expiresInHours":24}`),
		ownerToken,
	)
	if createdInvite.Code != http.StatusCreated {
		t.Fatalf("create invite status=%d body=%s", createdInvite.Code, createdInvite.Body.String())
	}
	var invitePayload struct {
		Data struct {
			Invite struct {
				Token string `json:"token"`
			} `json:"invite"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createdInvite.Body).Decode(&invitePayload); err != nil {
		t.Fatal(err)
	}
	inviteToken := invitePayload.Data.Invite.Token
	if inviteToken == "" {
		t.Fatalf("invalid invite payload: %#v", invitePayload)
	}

	preview := serveIntegrationRequest(handler, http.MethodGet, "/api/invites/"+inviteToken, nil)
	if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), "契约空间（已改名）") {
		t.Fatalf("preview invite status=%d body=%s", preview.Code, preview.Body.String())
	}

	memberToken, _ := login("contract-member")
	accepted := serveAuthenticatedIntegrationRequest(
		handler,
		http.MethodPost,
		"/api/invites/"+inviteToken+"/accept",
		nil,
		memberToken,
	)
	if accepted.Code != http.StatusOK || !strings.Contains(accepted.Body.String(), fmt.Sprintf(`"currentKitchenId":%d`, contractKitchenID)) {
		t.Fatalf("accept invite status=%d body=%s", accepted.Code, accepted.Body.String())
	}

	memberKitchens := serveAuthenticatedIntegrationRequest(
		handler,
		http.MethodGet,
		"/api/kitchens",
		nil,
		memberToken,
	)
	if memberKitchens.Code != http.StatusOK || !strings.Contains(memberKitchens.Body.String(), "契约空间（已改名）") {
		t.Fatalf("member kitchens status=%d body=%s", memberKitchens.Code, memberKitchens.Body.String())
	}
}

func TestReadinessFailsButLivenessSurvivesClosedDatabase(t *testing.T) {
	application := newIntegrationApp(t, "local")
	if err := application.DB.Close(); err != nil {
		t.Fatal(err)
	}

	ready := serveIntegrationRequest(application.Server.Handler, http.MethodGet, "/readyz", nil)
	if ready.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status=%d body=%s", ready.Code, ready.Body.String())
	}
	live := serveIntegrationRequest(application.Server.Handler, http.MethodGet, "/livez", nil)
	if live.Code != http.StatusOK {
		t.Fatalf("live status=%d body=%s", live.Code, live.Body.String())
	}
	if live.Header().Get("X-Release-ID") == "" {
		t.Fatal("liveness response missing release ID")
	}
	for _, field := range []string{`"releaseId":`, `"gitCommit":`, `"buildTime":`, `"goToolchain":`} {
		if !strings.Contains(live.Body.String(), field) {
			t.Fatalf("liveness response missing build field %s: %s", field, live.Body.String())
		}
	}
}

func TestReadinessRejectsIncompleteMigrations(t *testing.T) {
	application := newIntegrationApp(t, "local")
	if _, err := application.DB.Exec(`DELETE FROM schema_migrations WHERE filename = (SELECT MAX(filename) FROM schema_migrations)`); err != nil {
		t.Fatal(err)
	}

	response := serveIntegrationRequest(application.Server.Handler, http.MethodGet, "/healthz", nil)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("health alias status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestReadinessRejectsInvalidWritableDirectoryWithoutLeakingPath(t *testing.T) {
	application := newIntegrationApp(t, "local")
	invalidUploadDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(invalidUploadDir, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := application.Config
	cfg.UploadDir = invalidUploadDir
	health := newHealthHandler(cfg, application.DB, application.Logger)
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	health.ready(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), invalidUploadDir) {
		t.Fatalf("readiness response leaked internal path: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":50300`) {
		t.Fatalf("readiness response missing stable unavailable code: %s", response.Body.String())
	}
}

func TestAdminWritesRequireSessionBoundCSRFToken(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler
	login := serveIntegrationRequest(
		handler,
		http.MethodPost,
		"/api/admin/auth/login",
		[]byte(`{"username":"admin","password":"admin-secret"}`),
	)
	if login.Code != http.StatusOK {
		t.Fatalf("admin login status=%d body=%s", login.Code, login.Body.String())
	}
	var loginPayload struct {
		Data struct {
			CSRFToken string `json:"csrfToken"`
		} `json:"data"`
	}
	if err := json.NewDecoder(login.Body).Decode(&loginPayload); err != nil {
		t.Fatal(err)
	}
	cookies := login.Result().Cookies()
	if len(cookies) != 1 || loginPayload.Data.CSRFToken == "" {
		t.Fatalf("admin login cookie=%#v payload=%#v", cookies, loginPayload)
	}
	sessionCookie := cookies[0]
	if sessionCookie.Path != "/api/admin" || !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("admin session cookie is not narrowly scoped: %#v", sessionCookie)
	}

	tests := []struct {
		name      string
		csrfToken string
		fetchSite string
		want      int
	}{
		{name: "missing CSRF", want: http.StatusForbidden},
		{name: "cross-site", csrfToken: loginPayload.Data.CSRFToken, fetchSite: "cross-site", want: http.StatusForbidden},
		{name: "same-origin", csrfToken: loginPayload.Data.CSRFToken, fetchSite: "same-origin", want: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/admin/auth/logout", nil)
			request.AddCookie(sessionCookie)
			if test.csrfToken != "" {
				request.Header.Set(admin.AdminCSRFHeader, test.csrfToken)
			}
			if test.fetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", test.fetchSite)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status=%d body=%s, want=%d", response.Code, response.Body.String(), test.want)
			}
			if test.want == http.StatusOK {
				logoutCookies := response.Result().Cookies()
				if len(logoutCookies) != 1 || logoutCookies[0].Path != "/api/admin" || logoutCookies[0].MaxAge >= 0 {
					t.Fatalf("logout cookie=%#v", logoutCookies)
				}
			}
		})
	}
}

func TestUserLogoutAndVersionIncrementRevokeOldTokens(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler
	login := func() (string, int64) {
		response := serveIntegrationRequest(
			handler,
			http.MethodPost,
			"/api/auth/dev-login",
			[]byte(`{"identity":"token-version-user"}`),
		)
		if response.Code != http.StatusOK {
			t.Fatalf("dev login status=%d body=%s", response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "tokenVersion") {
			t.Fatalf("internal token version leaked in session response: %s", response.Body.String())
		}
		var payload struct {
			Data struct {
				Token string `json:"token"`
				User  struct {
					ID int64 `json:"id"`
				} `json:"user"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		return payload.Data.Token, payload.Data.User.ID
	}
	requestWithToken := func(method, path, token string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	firstToken, userID := login()
	if response := requestWithToken(http.MethodGet, "/api/auth/me", firstToken); response.Code != http.StatusOK {
		t.Fatalf("initial token status=%d body=%s", response.Code, response.Body.String())
	}
	logout := requestWithToken(http.MethodPost, "/api/auth/logout", firstToken)
	if logout.Code != http.StatusOK {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body.String())
	}
	if response := requestWithToken(http.MethodGet, "/api/auth/me", firstToken); response.Code != http.StatusUnauthorized {
		t.Fatalf("logged-out token status=%d body=%s", response.Code, response.Body.String())
	}

	secondToken, secondUserID := login()
	if secondUserID != userID || secondToken == firstToken {
		t.Fatalf("second login user=%d tokenChanged=%t", secondUserID, secondToken != firstToken)
	}
	if response := requestWithToken(http.MethodGet, "/api/auth/me", secondToken); response.Code != http.StatusOK {
		t.Fatalf("second token status=%d body=%s", response.Code, response.Body.String())
	}
	if _, err := application.DB.Exec(`UPDATE users SET token_version = token_version + 1 WHERE id = ?`, userID); err != nil {
		t.Fatal(err)
	}
	if response := requestWithToken(http.MethodGet, "/api/auth/me", secondToken); response.Code != http.StatusUnauthorized {
		t.Fatalf("version-incremented token status=%d body=%s", response.Code, response.Body.String())
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

func TestProductionRouterForbidsOrdinaryUserGlobalSettingsWrite(t *testing.T) {
	application := newIntegrationApp(t, "production")
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := application.DB.Exec(
		`INSERT INTO users (openid, nickname, created_at, updated_at) VALUES ('ordinary-user', 'Ordinary', ?, ?)`,
		now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	token, err := auth.NewTokenManager(application.Config.JWTSecret, 1).Issue(userID, 1)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodPut,
		"/api/app-settings/bilibili-session",
		bytes.NewReader([]byte(`{"sessdata":"must-not-be-saved"}`)),
	)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	application.Server.Handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("settings update status=%d body=%s, want 403", response.Code, response.Body.String())
	}
}

func TestAppRouterJSONRequestContractAndBodyLimit(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "valid", body: `{"identity":"json-contract-user"}`, wantStatus: http.StatusOK},
		{name: "empty", body: "", wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `{"identity":"user","extra":true}`, wantStatus: http.StatusBadRequest},
		{name: "multiple values", body: `{"identity":"user"} {"identity":"other"}`, wantStatus: http.StatusBadRequest},
		{name: "trailing garbage", body: `{"identity":"user"} trailing`, wantStatus: http.StatusBadRequest},
		{
			name:       "oversized",
			body:       `{"identity":"` + strings.Repeat("a", int(defaultRequestBodyLimit)) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serveIntegrationRequest(
				handler,
				http.MethodPost,
				"/api/auth/dev-login",
				[]byte(test.body),
			)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d body=%s, want=%d", response.Code, response.Body.String(), test.wantStatus)
			}
		})
	}
}

func TestAppRouterUsesExplicitBodyLimitOverrides(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler
	body := []byte(strings.Repeat("x", int(defaultRequestBodyLimit+1)))

	for _, path := range []string{
		"/api/diet-assistant/chat/stream",
		"/api/uploads/images",
	} {
		response := serveIntegrationRequest(handler, http.MethodPost, path, body)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("POST %s status=%d body=%s, want auth boundary after body override", path, response.Code, response.Body.String())
		}
	}
}

func TestAppRouterServesOnlyPublicImageFilesWithoutDirectoryListing(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler
	imageDir := filepath.Join(application.Config.UploadDir, "2026", "07")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	imageName := "img_1_0123456789ab.png"
	imageData := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(imageDir, imageName), imageData, 0o644); err != nil {
		t.Fatal(err)
	}

	imageResponse := serveIntegrationRequest(
		handler,
		http.MethodGet,
		"/uploads/2026/07/"+imageName,
		nil,
	)
	if imageResponse.Code != http.StatusOK || !bytes.Equal(imageResponse.Body.Bytes(), imageData) {
		t.Fatalf("image status=%d body=%q", imageResponse.Code, imageResponse.Body.Bytes())
	}
	if got := imageResponse.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("image Cache-Control=%q", got)
	}

	directoryResponse := serveIntegrationRequest(handler, http.MethodGet, "/uploads/2026/07/", nil)
	if directoryResponse.Code != http.StatusNotFound {
		t.Fatalf("directory status=%d body=%s", directoryResponse.Code, directoryResponse.Body.String())
	}
	if strings.Contains(directoryResponse.Body.String(), imageName) {
		t.Fatalf("directory response leaked file name: %s", directoryResponse.Body.String())
	}
}

func TestAppRouterRateLimitsAdminLoginByAccount(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler
	body := []byte(`{"username":"admin","password":"wrong-password"}`)

	for attempt := 1; attempt <= 6; attempt++ {
		response := serveIntegrationRequest(handler, http.MethodPost, "/api/admin/auth/login", body)
		wantStatus := http.StatusUnauthorized
		if attempt == 6 {
			wantStatus = http.StatusTooManyRequests
		}
		if response.Code != wantStatus {
			t.Fatalf("attempt=%d status=%d body=%s, want=%d", attempt, response.Code, response.Body.String(), wantStatus)
		}
	}
}

func TestAppRouterRateLimitsInvalidAdminLoginBodiesByIP(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler

	for attempt := 1; attempt <= 11; attempt++ {
		response := serveIntegrationRequest(handler, http.MethodPost, "/api/admin/auth/login", nil)
		wantStatus := http.StatusBadRequest
		if attempt == 11 {
			wantStatus = http.StatusTooManyRequests
		}
		if response.Code != wantStatus {
			t.Fatalf("attempt=%d status=%d body=%s, want=%d", attempt, response.Code, response.Body.String(), wantStatus)
		}
	}
}

func TestAppRouterRateLimitsInvitePreviewAndAcceptByTarget(t *testing.T) {
	application := newIntegrationApp(t, "local")
	handler := application.Server.Handler

	for attempt := 1; attempt <= 21; attempt++ {
		response := serveIntegrationRequest(handler, http.MethodGet, "/api/invites/repeated-missing-token", nil)
		wantStatus := http.StatusNotFound
		if attempt == 21 {
			wantStatus = http.StatusTooManyRequests
		}
		if response.Code != wantStatus {
			t.Fatalf("preview attempt=%d status=%d body=%s, want=%d", attempt, response.Code, response.Body.String(), wantStatus)
		}
	}

	login := serveIntegrationRequest(
		handler,
		http.MethodPost,
		"/api/auth/dev-login",
		[]byte(`{"identity":"invite-rate-limit-user"}`),
	)
	if login.Code != http.StatusOK {
		t.Fatalf("dev login status=%d body=%s", login.Code, login.Body.String())
	}
	var loginPayload struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(login.Body).Decode(&loginPayload); err != nil {
		t.Fatal(err)
	}

	for attempt := 1; attempt <= 6; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/invites/repeated-missing-token/accept", nil)
		request.Header.Set("Authorization", "Bearer "+loginPayload.Data.Token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		wantStatus := http.StatusNotFound
		if attempt == 6 {
			wantStatus = http.StatusTooManyRequests
		}
		if response.Code != wantStatus {
			t.Fatalf("accept attempt=%d status=%d body=%s, want=%d", attempt, response.Code, response.Body.String(), wantStatus)
		}
	}
}

func TestAppServerResourceBoundariesKeepStreamingWritesUnlimited(t *testing.T) {
	application := newIntegrationApp(t, "local")
	server := application.Server

	if server.ReadHeaderTimeout != 5*time.Second || server.ReadTimeout != 30*time.Second {
		t.Fatalf("read timeouts: header=%s body=%s", server.ReadHeaderTimeout, server.ReadTimeout)
	}
	if server.MaxHeaderBytes != 1<<20 {
		t.Fatalf("MaxHeaderBytes=%d, want=%d", server.MaxHeaderBytes, 1<<20)
	}
	if server.WriteTimeout != 0 {
		t.Fatalf("WriteTimeout=%s, want zero so SSE is not truncated", server.WriteTimeout)
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
	accessMode := "all"
	if appEnv != "local" {
		accessMode = "admin"
	}
	cfg := config.Config{
		AppName:                        "caipu-integration-test",
		AppEnv:                         appEnv,
		AppAddr:                        ":0",
		LogLevel:                       "error",
		AdminUsername:                  "admin",
		AdminPasswordHash:              string(passwordHash),
		AdminJWTSecret:                 "admin-jwt-secret",
		AdminCookiePath:                "/api/admin",
		AppSettingsAccessMode:          accessMode,
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
		UploadPublicBaseURL:            "https://static.example.com/uploads",
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

func serveAuthenticatedIntegrationRequest(
	handler http.Handler,
	method string,
	path string,
	body []byte,
	token string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
