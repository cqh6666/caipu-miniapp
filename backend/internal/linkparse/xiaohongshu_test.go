package linkparse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDetectParsePlatform(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"https://www.bilibili.com/video/BV1aWCEYHErc", "bilibili"},
		{"https://b23.tv/abc123", "bilibili"},
		{"https://www.xiaohongshu.com/explore/68abcd1234", "xiaohongshu"},
		{"http://xhslink.com/a/xxxx", "xiaohongshu"},
		{"https://example.com/demo", ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := DetectParsePlatform(tc.input); got != tc.want {
				t.Fatalf("DetectParsePlatform(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseXiaohongshuUsesSidecar(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/parse/xiaohongshu" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sidecar-secret" {
			t.Fatalf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"platform": "xiaohongshu",
			"providerRequested": "auto",
			"providerUsed": "importer",
			"normalized": {
				"shareUrl": "http://xhslink.com/a/demo",
				"canonicalUrl": "https://www.xiaohongshu.com/explore/68abcd1234",
				"noteId": "68abcd1234"
			},
			"note": {
				"title": "番茄牛腩",
				"content": "牛腩 500克\n番茄 3个\n牛腩焯水后和番茄一起炖煮。",
				"tags": ["家常菜", "番茄牛腩"],
				"images": ["http://ci.xiaohongshu.com/1.jpg"],
				"videos": [],
				"coverUrl": "http://ci.xiaohongshu.com/cover.jpg",
				"author": {"name": "测试厨房"},
				"noteType": "image"
			},
			"warnings": []
		}`))
	}))
	defer server.Close()

	svc := NewService(Options{
		XHSSidecarEnabled:  true,
		XHSSidecarBaseURL:  server.URL,
		XHSSidecarTimeout:  3 * time.Second,
		XHSSidecarProvider: "auto",
		XHSSidecarAPIKey:   "sidecar-secret",
	})

	result, err := svc.ParseXiaohongshu(context.Background(), "https://www.xiaohongshu.com/explore/68abcd1234")
	if err != nil {
		t.Fatalf("ParseXiaohongshu returned error: %v", err)
	}
	if result.Source != "xiaohongshu" {
		t.Fatalf("Source = %q", result.Source)
	}
	if result.ProviderUsed != "importer" {
		t.Fatalf("ProviderUsed = %q", result.ProviderUsed)
	}
	if result.CanonicalURL != "https://www.xiaohongshu.com/explore/68abcd1234" {
		t.Fatalf("CanonicalURL = %q", result.CanonicalURL)
	}
	if got, want := len(result.Images), 1; got != want {
		t.Fatalf("len(Images) = %d, want %d", got, want)
	}
	if got, want := result.CoverURL, "https://ci.xiaohongshu.com/cover.jpg"; got != want {
		t.Fatalf("CoverURL = %q, want %q", got, want)
	}
	if !strings.Contains(result.RecipeDraft.Ingredient, "番茄") {
		t.Fatalf("RecipeDraft.Ingredient = %q", result.RecipeDraft.Ingredient)
	}
	if result.RecipeDraft.Summary != "" {
		t.Fatalf("RecipeDraft.Summary = %q, want empty without AI summary", result.RecipeDraft.Summary)
	}
	if len(result.RecipeDraft.ParsedContent.Steps) == 0 {
		t.Fatalf("RecipeDraft steps are empty: %#v", result.RecipeDraft)
	}
	if got, want := len(result.RecipeDraft.ImageURLs), 1; got != want {
		t.Fatalf("len(RecipeDraft.ImageURLs) = %d, want %d", got, want)
	}
	if got, want := result.Images[0], "https://ci.xiaohongshu.com/1.jpg"; got != want {
		t.Fatalf("Images[0] = %q, want %q", got, want)
	}
	if got, want := result.RecipeDraft.ImageURLs[0], "https://ci.xiaohongshu.com/1.jpg"; got != want {
		t.Fatalf("RecipeDraft.ImageURLs[0] = %q, want %q", got, want)
	}
}

func TestPreviewXiaohongshuUsesSidecar(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"platform": "xiaohongshu",
			"providerRequested": "auto",
			"providerUsed": "importer",
			"normalized": {
				"shareUrl": "http://xhslink.com/o/demo123",
				"canonicalUrl": "https://www.xiaohongshu.com/explore/68abcd1234",
				"noteId": "68abcd1234"
			},
			"note": {
				"title": "番茄土豆炖牛腩教程来咯～",
				"content": "正文",
				"tags": ["家常菜"],
				"images": ["http://ci.xiaohongshu.com/1.jpg", "https://ci.xiaohongshu.com/2.jpg"],
				"videos": [],
				"coverUrl": "http://ci.xiaohongshu.com/cover.jpg",
				"author": {"name": "测试厨房"},
				"noteType": "image"
			},
			"warnings": ["demo"]
		}`))
	}))
	defer server.Close()

	svc := NewService(Options{
		XHSSidecarEnabled:  true,
		XHSSidecarBaseURL:  server.URL,
		XHSSidecarTimeout:  3 * time.Second,
		XHSSidecarProvider: "auto",
	})

	result, err := svc.PreviewXiaohongshu(context.Background(), "http://xhslink.com/o/demo123")
	if err != nil {
		t.Fatalf("PreviewXiaohongshu returned error: %v", err)
	}
	if got, want := result.Platform, "xiaohongshu"; got != want {
		t.Fatalf("Platform = %q, want %q", got, want)
	}
	if got, want := result.Title, "番茄土豆炖牛腩"; got != want {
		t.Fatalf("Title = %q, want %q", got, want)
	}
	if got, want := result.CoverURL, "https://ci.xiaohongshu.com/cover.jpg"; got != want {
		t.Fatalf("CoverURL = %q, want %q", got, want)
	}
	if got, want := len(result.ImageURLs), 2; got != want {
		t.Fatalf("len(ImageURLs) = %d, want %d", got, want)
	}
}
