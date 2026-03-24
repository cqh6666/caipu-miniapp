package linkparse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

	result, err := svc.ParseBilibili(context.Background(), "https://b23.tv/demo123")
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
