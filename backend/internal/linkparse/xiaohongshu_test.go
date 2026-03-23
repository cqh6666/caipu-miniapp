package linkparse

import (
	"context"
	"encoding/json"
	"io"
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
		var req xhsSidecarParseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.IncludeTranscript {
			t.Fatal("ParseXiaohongshu should not request transcript by default")
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
				"transcript": "先把牛腩冷水下锅焯水，再和番茄一起慢炖到软烂。",
				"transcriptStatus": "success",
				"transcriptError": "",
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
	if got, want := result.TranscriptStatus, "success"; got != want {
		t.Fatalf("TranscriptStatus = %q, want %q", got, want)
	}
	if !strings.Contains(result.Transcript, "焯水") {
		t.Fatalf("Transcript = %q", result.Transcript)
	}
	if got, want := result.CoverURL, "https://ci.xiaohongshu.com/cover.jpg"; got != want {
		t.Fatalf("CoverURL = %q, want %q", got, want)
	}
	if !strings.Contains(result.RecipeDraft.Ingredient, "番茄") {
		t.Fatalf("RecipeDraft.Ingredient = %q", result.RecipeDraft.Ingredient)
	}
	if result.RecipeDraft.Summary == "" {
		t.Fatal("RecipeDraft.Summary should not be empty without AI summary")
	}
	if len(result.RecipeDraft.ParsedContent.Steps) == 0 {
		t.Fatalf("RecipeDraft steps are empty: %#v", result.RecipeDraft)
	}
	if len(result.RecipeDraft.ParsedContent.MainIngredients) == 0 {
		t.Fatalf("RecipeDraft main ingredients are empty: %#v", result.RecipeDraft)
	}
	if result.RecipeDraft.ParsedContent.Steps[0].Title == "" || result.RecipeDraft.ParsedContent.Steps[0].Detail == "" {
		t.Fatalf("RecipeDraft first step should contain title and detail: %#v", result.RecipeDraft.ParsedContent.Steps[0])
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

func TestBuildXiaohongshuAISummaryPromptIncludesTranscript(t *testing.T) {
	t.Parallel()

	prompt := buildXiaohongshuAISummaryPrompt(XiaohongshuParseResult{
		Title:      "葱姜煎鲳鱼",
		Content:    "正文里只有标题，没有细步骤。",
		Transcript: "先把鲳鱼擦干，再下锅煎到两面金黄，最后加葱姜焖两分钟。",
		Tags:       []string{"家常菜", "鲳鱼"},
	})

	if !strings.Contains(prompt, "视频转写") {
		t.Fatalf("prompt should include transcript section: %q", prompt)
	}
	if !strings.Contains(prompt, "两面金黄") {
		t.Fatalf("prompt should contain transcript content: %q", prompt)
	}
}

func TestSummarizeXiaohongshuHeuristicallyUsesTranscript(t *testing.T) {
	t.Parallel()

	draft := summarizeXiaohongshuHeuristically(XiaohongshuParseResult{
		Title:      "葱姜煎鲳鱼",
		Content:    "只有标题，没有做法。",
		Transcript: "鲳鱼 1条\n葱 2根\n姜 6片\n先把鲳鱼擦干，再下锅煎到两面金黄，最后放葱姜焖两分钟。",
	})

	if len(draft.ParsedContent.MainIngredients) == 0 {
		t.Fatalf("main ingredients should not be empty: %#v", draft)
	}
	if len(draft.ParsedContent.Steps) == 0 {
		t.Fatalf("steps should not be empty: %#v", draft)
	}
	if !strings.Contains(draft.ParsedContent.Steps[0].Detail, "煎到两面金黄") &&
		(len(draft.ParsedContent.Steps) < 2 || !strings.Contains(draft.ParsedContent.Steps[1].Detail, "煎到两面金黄")) {
		t.Fatalf("transcript detail should be used in heuristic steps: %#v", draft.ParsedContent.Steps)
	}
}

func TestPreviewXiaohongshuUsesSidecar(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req xhsSidecarParseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.IncludeTranscript {
			t.Fatal("PreviewXiaohongshu should not request transcript")
		}
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

func TestParseRecipeLinkRequestsTranscriptForXiaohongshu(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req xhsSidecarParseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if !req.IncludeTranscript {
			t.Fatal("ParseRecipeLink should request transcript for xiaohongshu auto-parse")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"platform": "xiaohongshu",
			"providerRequested": "auto",
			"providerUsed": "rednote",
			"normalized": {
				"shareUrl": "http://xhslink.com/o/demo123",
				"canonicalUrl": "https://www.xiaohongshu.com/explore/68abcd1234",
				"noteId": "68abcd1234"
			},
			"note": {
				"title": "葱姜煎鲳鱼",
				"content": "正文很短。",
				"transcript": "鲳鱼 1条\n葱 2根\n姜 6片\n先煎到两面金黄，再加葱姜焖两分钟。",
				"transcriptStatus": "success",
				"transcriptError": "",
				"tags": ["家常菜"],
				"images": ["http://ci.xiaohongshu.com/1.jpg"],
				"videos": ["https://sns-video-hw.xhscdn.com/demo.mp4"],
				"coverUrl": "http://ci.xiaohongshu.com/cover.jpg",
				"author": {"name": "测试厨房"},
				"noteType": "video"
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
	})

	outcome, err := svc.ParseRecipeLink(context.Background(), "https://www.xiaohongshu.com/explore/68abcd1234")
	if err != nil {
		t.Fatalf("ParseRecipeLink returned error: %v", err)
	}
	if !strings.Contains(outcome.RecipeDraft.Note, "视频转写") {
		t.Fatalf("RecipeDraft.Note should mention transcript usage: %#v", outcome.RecipeDraft)
	}
	if len(outcome.RecipeDraft.ParsedContent.Steps) == 0 {
		t.Fatalf("RecipeDraft steps should not be empty: %#v", outcome.RecipeDraft)
	}
}

func TestParseRecipeLinkFallsBackWhenTranscriptTimesOut(t *testing.T) {
	t.Parallel()

	svc := NewService(Options{
		XHSSidecarEnabled:  true,
		XHSSidecarBaseURL:  "http://xhs-sidecar.test",
		XHSSidecarTimeout:  20 * time.Millisecond,
		XHSSidecarProvider: "auto",
	})
	if svc.xhs == nil {
		t.Fatal("expected xiaohongshu sidecar client")
	}

	var calls []bool
	svc.xhs.client = &http.Client{
		Timeout: 20 * time.Millisecond,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			var payload xhsSidecarParseRequest
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request body: %v", err)
			}

			calls = append(calls, payload.IncludeTranscript)
			if payload.IncludeTranscript {
				return nil, context.DeadlineExceeded
			}

			return jsonHTTPResponse(`{
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
					"title": "葱姜煎鲳鱼",
					"content": "鲳鱼 1条\n葱 2根\n姜 6片\n先把鲳鱼擦干，再下锅煎到两面金黄，最后放葱姜焖两分钟。",
					"transcript": "",
					"transcriptStatus": "",
					"transcriptError": "",
					"tags": ["家常菜"],
					"images": ["http://ci.xiaohongshu.com/1.jpg"],
					"videos": ["https://sns-video-hw.xhscdn.com/demo.mp4"],
					"coverUrl": "http://ci.xiaohongshu.com/cover.jpg",
					"author": {"name": "测试厨房"},
					"noteType": "video"
				},
				"warnings": []
			}`), nil
		}),
	}

	outcome, err := svc.ParseRecipeLink(context.Background(), "https://www.xiaohongshu.com/explore/68abcd1234")
	if err != nil {
		t.Fatalf("ParseRecipeLink returned error: %v", err)
	}
	if !strings.Contains(outcome.RecipeDraft.Note, "未成功转写") {
		t.Fatalf("RecipeDraft.Note should mention transcript fallback: %#v", outcome.RecipeDraft)
	}
	if len(outcome.RecipeDraft.ParsedContent.Steps) == 0 {
		t.Fatalf("RecipeDraft steps should not be empty after fallback: %#v", outcome.RecipeDraft)
	}
	if got, want := len(calls), 2; got != want {
		t.Fatalf("sidecar calls = %d, want %d", got, want)
	}
	if !calls[0] {
		t.Fatalf("first request should include transcript: %#v", calls)
	}
	if calls[1] {
		t.Fatalf("second request should disable transcript after timeout: %#v", calls)
	}
}

func TestParseXiaohongshuReturnsTimeoutError(t *testing.T) {
	t.Parallel()

	svc := NewService(Options{
		XHSSidecarEnabled: true,
		XHSSidecarBaseURL: "http://xhs-sidecar.test",
		XHSSidecarTimeout: 20 * time.Millisecond,
	})
	if svc.xhs == nil {
		t.Fatal("expected xiaohongshu sidecar client")
	}
	svc.xhs.client = &http.Client{
		Timeout: 20 * time.Millisecond,
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, context.DeadlineExceeded
		}),
	}

	_, err := svc.ParseXiaohongshu(context.Background(), "https://www.xiaohongshu.com/explore/68abcd1234")
	if err == nil {
		t.Fatal("ParseXiaohongshu should fail on timeout")
	}
	if got := err.Error(); got != "xiaohongshu sidecar timed out" {
		t.Fatalf("error = %q, want timeout message", got)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonHTTPResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}
