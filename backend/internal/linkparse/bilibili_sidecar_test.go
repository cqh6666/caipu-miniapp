package linkparse

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func TestParseBilibiliUsesSidecar(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/parse/bilibili" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sidecar-secret" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("X-Bilibili-SESSDATA"); got != "sess-123" {
			t.Fatalf("X-Bilibili-SESSDATA = %q", got)
		}

		var req sidecarParseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if !req.IncludeTranscript {
			t.Fatal("ParseBilibili should request transcript")
		}
		if got, want := req.Input, "https://b23.tv/demo123"; got != want {
			t.Fatalf("Input = %q, want normalized %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"platform": "bilibili",
			"providerRequested": "auto",
			"providerUsed": "openapi",
			"normalized": {
				"shareUrl": "https://b23.tv/demo123",
				"canonicalUrl": "https://www.bilibili.com/video/BV1xx411c7mD?p=2",
				"id": "BV1xx411c7mD",
				"bvid": "BV1xx411c7mD",
				"aid": 10086,
				"cid": 20086,
				"page": 2
			},
			"content": {
				"title": "番茄牛腩",
				"description": "牛腩 500克\n番茄 3个",
				"body": "",
				"part": "第二集",
				"transcript": "牛腩 500克\n番茄 3个\n先焯水，再和番茄一起慢炖。",
				"transcriptStatus": "success",
				"transcriptError": "",
				"tags": [],
				"images": [],
				"videos": [],
				"coverUrl": "https://i0.hdslb.com/demo.jpg",
				"author": {"name": "厨房UP"},
				"contentType": "video",
				"likes": 0,
				"comments": 0,
				"favorites": 0,
				"subtitleLanguage": "中文",
				"subtitleSegments": 3
			},
			"warnings": ["已自动展开 B 站短链接。"],
			"quality": "full"
		}`))
	}))
	defer server.Close()

	svc := NewService(Options{
		LinkparseSidecarEnabled: true,
		LinkparseSidecarBaseURL: server.URL,
		LinkparseSidecarTimeout: 3 * time.Second,
		LinkparseSidecarAPIKey:  "sidecar-secret",
		BilibiliSessdataProvider: func(context.Context) string {
			return "sess-123"
		},
	})

	result, err := svc.ParseBilibili(context.Background(), "看看 https://b23.tv/demo123。")
	if err != nil {
		t.Fatalf("ParseBilibili returned error: %v", err)
	}
	if got, want := result.Source, "bilibili"; got != want {
		t.Fatalf("Source = %q, want %q", got, want)
	}
	if got, want := result.BVID, "BV1xx411c7mD"; got != want {
		t.Fatalf("BVID = %q, want %q", got, want)
	}
	if !result.SubtitleAvailable {
		t.Fatal("SubtitleAvailable should be true")
	}
	if got, want := result.SubtitleLanguage, "中文"; got != want {
		t.Fatalf("SubtitleLanguage = %q, want %q", got, want)
	}
	if !strings.Contains(result.RecipeDraft.Ingredient, "番茄") {
		t.Fatalf("RecipeDraft.Ingredient = %q", result.RecipeDraft.Ingredient)
	}
	if len(result.RecipeDraft.ParsedContent.Steps) == 0 {
		t.Fatalf("RecipeDraft steps are empty: %#v", result.RecipeDraft)
	}
}

func TestParseBilibiliFallsBackWithoutTranscriptViaSidecar(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req sidecarParseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if !req.IncludeTranscript {
			t.Fatal("ParseBilibili should request transcript")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"platform": "bilibili",
			"providerRequested": "auto",
			"providerUsed": "openapi",
			"normalized": {
				"shareUrl": "https://www.bilibili.com/video/BV1xx411c7mD",
				"canonicalUrl": "https://www.bilibili.com/video/BV1xx411c7mD",
				"id": "BV1xx411c7mD",
				"bvid": "BV1xx411c7mD",
				"aid": 10086,
				"cid": 20086,
				"page": 1
			},
			"content": {
				"title": "番茄牛腩",
				"description": "牛腩 500克\n番茄 3个\n炖到软烂即可。",
				"body": "",
				"part": "",
				"transcript": "",
				"transcriptStatus": "skipped",
				"transcriptError": "",
				"tags": [],
				"images": [],
				"videos": [],
				"coverUrl": "https://i0.hdslb.com/demo.jpg",
				"author": {"name": "厨房UP"},
				"contentType": "video",
				"likes": 0,
				"comments": 0,
				"favorites": 0,
				"subtitleLanguage": "",
				"subtitleSegments": 0
			},
			"warnings": ["当前视频没有可直接访问的字幕。"],
			"quality": "degraded"
		}`))
	}))
	defer server.Close()

	svc := NewService(Options{
		LinkparseSidecarEnabled: true,
		LinkparseSidecarBaseURL: server.URL,
		LinkparseSidecarTimeout: 3 * time.Second,
	})

	result, err := svc.ParseBilibili(context.Background(), "https://www.bilibili.com/video/BV1xx411c7mD")
	if err != nil {
		t.Fatalf("ParseBilibili returned error: %v", err)
	}
	if got, want := result.SummaryMode, "heuristic"; got != want {
		t.Fatalf("SummaryMode = %q, want %q", got, want)
	}
	if len(result.RecipeDraft.ParsedContent.Steps) == 0 {
		t.Fatalf("RecipeDraft steps should not be empty: %#v", result.RecipeDraft)
	}
	if !strings.Contains(strings.Join(result.Warnings, "\n"), "字幕") {
		t.Fatalf("Warnings should mention transcript fallback: %#v", result.Warnings)
	}
}

func TestBilibiliOperationsFailWhenSidecarIsNotConfigured(t *testing.T) {
	t.Parallel()

	service := NewService(Options{})
	operations := []struct {
		name string
		run  func() error
	}{
		{name: "preview", run: func() error {
			_, err := service.PreviewBilibili(context.Background(), "https://www.bilibili.com/video/BV1xx411c7mD")
			return err
		}},
		{name: "parse", run: func() error {
			_, err := service.ParseBilibili(context.Background(), "https://www.bilibili.com/video/BV1xx411c7mD")
			return err
		}},
		{name: "verify", run: func() error {
			return service.VerifyBilibiliSessdata(context.Background(), "session-secret")
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			var appErr *common.AppError
			if err := operation.run(); !errors.As(err, &appErr) {
				t.Fatalf("error = %v, want *common.AppError", err)
			}
			if appErr.HTTPStatus != http.StatusServiceUnavailable || appErr.Message != "linkparse sidecar is not configured" {
				t.Fatalf("AppError = %#v", appErr)
			}
		})
	}
}

func TestVerifyBilibiliSessdataUsesSidecarEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/verify/bilibili-session" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sidecar-secret" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := r.Header.Get("X-Bilibili-SESSDATA"); got != "session-secret" {
			t.Fatalf("X-Bilibili-SESSDATA = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"platform":"bilibili","providerUsed":"openapi","valid":true}`))
	}))
	defer server.Close()

	service := NewService(Options{
		LinkparseSidecarEnabled: true,
		LinkparseSidecarBaseURL: server.URL,
		LinkparseSidecarAPIKey:  "sidecar-secret",
	})
	if err := service.VerifyBilibiliSessdata(context.Background(), "session-secret"); err != nil {
		t.Fatalf("VerifyBilibiliSessdata() error = %v", err)
	}
}

func TestVerifyBilibiliSessdataMapsSidecarErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		body       string
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "invalid credentials",
			status:     http.StatusBadRequest,
			body:       `{"ok":false,"error":{"code":"invalid_credentials","message":"TOP_SECRET upstream details"}}`,
			wantStatus: http.StatusBadRequest,
			wantMsg:    "当前 SESSDATA 无法获取 B 站字幕，请更新后重试",
		},
		{
			name:       "provider unavailable",
			status:     http.StatusServiceUnavailable,
			body:       `{"ok":false,"error":{"code":"provider_unavailable","message":"private sidecar details"}}`,
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "bilibili session verification is unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			service := NewService(Options{
				LinkparseSidecarEnabled: true,
				LinkparseSidecarBaseURL: server.URL,
			})
			var appErr *common.AppError
			err := service.VerifyBilibiliSessdata(context.Background(), "session-secret")
			if !errors.As(err, &appErr) {
				t.Fatalf("error = %v, want *common.AppError", err)
			}
			if appErr.HTTPStatus != test.wantStatus || appErr.Message != test.wantMsg {
				t.Fatalf("AppError = %#v", appErr)
			}
			if strings.Contains(appErr.Error(), "private") || strings.Contains(appErr.Error(), "TOP_SECRET") {
				t.Fatalf("error leaked sidecar details: %v", appErr)
			}
		})
	}
}

func TestBilibiliSidecarMapsFailureClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		body       string
		wantStatus int
		wantCode   int
		wantMsg    string
	}{
		{
			name:       "invalid request",
			status:     http.StatusBadRequest,
			body:       `{"ok":false,"error":{"code":"unsupported_url","message":"private input details"}}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   common.CodeBadRequest,
			wantMsg:    "linkparse sidecar rejected the request",
		},
		{
			name:       "upstream failure",
			status:     http.StatusBadGateway,
			body:       `{"ok":false,"error":{"code":"upstream_failure","message":"private upstream details"}}`,
			wantStatus: http.StatusBadGateway,
			wantCode:   common.CodeInternalServer,
			wantMsg:    "linkparse sidecar upstream failed",
		},
		{
			name:       "provider unavailable",
			status:     http.StatusServiceUnavailable,
			body:       `{"ok":false,"error":{"code":"provider_unavailable","message":"private provider details"}}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   common.CodeServiceUnavailable,
			wantMsg:    "linkparse sidecar provider is unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			service := NewService(Options{
				LinkparseSidecarEnabled: true,
				LinkparseSidecarBaseURL: server.URL,
			})
			_, err := service.PreviewBilibili(context.Background(), "https://www.bilibili.com/video/BV1xx411c7mD")
			var appErr *common.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("error = %v, want *common.AppError", err)
			}
			if appErr.HTTPStatus != test.wantStatus || appErr.Code != test.wantCode || appErr.Message != test.wantMsg {
				t.Fatalf("AppError = %#v", appErr)
			}
			if strings.Contains(appErr.Error(), "private") {
				t.Fatalf("public error leaked private details: %v", appErr)
			}
		})
	}
}

func TestBilibiliSidecarTransportFailureIsBadGateway(t *testing.T) {
	t.Parallel()

	service := NewService(Options{
		LinkparseSidecarEnabled: true,
		LinkparseSidecarBaseURL: "http://sidecar.test",
	})
	service.sidecar.client.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed")
	})

	_, err := service.PreviewBilibili(context.Background(), "https://www.bilibili.com/video/BV1xx411c7mD")
	var appErr *common.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error = %v, want *common.AppError", err)
	}
	if appErr.HTTPStatus != http.StatusBadGateway || appErr.Code != common.CodeInternalServer || appErr.Message != "request to bilibili sidecar failed" {
		t.Fatalf("AppError = %#v", appErr)
	}
}

func TestBilibiliSidecarRejectsInvalidURLBeforeRequest(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "bilibili pseudo suffix", input: "https://bilibili.com.attacker.example/video/BV1xx411c7mD"},
		{name: "short link pseudo suffix", input: "https://b23.tv.attacker.example/demo"},
		{name: "userinfo", input: "https://user@www.bilibili.com/video/BV1xx411c7mD"},
		{name: "loopback", input: "http://127.0.0.1/video/BV1xx411c7mD"},
		{name: "invalid port", input: "https://b23.tv:bad/demo"},
	}
	operations := []struct {
		name string
		run  func(*Service, string) error
	}{
		{
			name: "parse",
			run: func(service *Service, input string) error {
				_, err := service.ParseBilibili(context.Background(), input)
				return err
			},
		},
		{
			name: "preview",
			run: func(service *Service, input string) error {
				_, err := service.PreviewBilibili(context.Background(), input)
				return err
			},
		},
	}

	for _, operation := range operations {
		for _, tt := range tests {
			t.Run(operation.name+"/"+tt.name, func(t *testing.T) {
				sidecarCalls := 0
				sessdataCalls := 0
				service := NewService(Options{
					LinkparseSidecarEnabled: true,
					LinkparseSidecarBaseURL: "http://sidecar.test",
					LinkparseSidecarTimeout: time.Second,
					BilibiliSessdataProvider: func(context.Context) string {
						sessdataCalls++
						return "TOP_SECRET"
					},
				})
				service.sidecar.client.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
					sidecarCalls++
					return nil, errors.New("unexpected sidecar request")
				})

				err := operation.run(service, tt.input)
				var appErr *common.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("error = %v, want *common.AppError", err)
				}
				if appErr.Code != common.CodeBadRequest || appErr.HTTPStatus != http.StatusBadRequest || appErr.Message != "invalid bilibili url" {
					t.Fatalf("AppError = %#v", appErr)
				}
				if sidecarCalls != 0 {
					t.Fatalf("sidecar calls = %d, want 0", sidecarCalls)
				}
				if sessdataCalls != 0 {
					t.Fatalf("SESSDATA provider calls = %d, want 0", sessdataCalls)
				}
			})
		}
	}
}
