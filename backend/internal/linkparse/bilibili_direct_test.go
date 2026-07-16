package linkparse

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func TestParseBilibiliDirectModeFetchesTrustedSubtitle(t *testing.T) {
	t.Parallel()

	transport := newBilibiliDirectTransport(t)
	service := NewService(Options{
		HTTPClient: &http.Client{Transport: transport},
		BilibiliSessdataProvider: func(context.Context) string {
			return "sess-direct"
		},
	})

	result, err := service.ParseBilibili(context.Background(), "https://www.bilibili.com/video/BV1xx411c7mD?p=2")
	if err != nil {
		t.Fatalf("ParseBilibili() error = %v", err)
	}
	if result.BVID != "BV1xx411c7mD" || result.CID != 20086 || result.Page != 2 {
		t.Fatalf("unexpected video identity: %#v", result)
	}
	if !result.SubtitleAvailable || result.SubtitleSegments != 3 || result.SubtitleLanguage != "中文" {
		t.Fatalf("unexpected subtitle result: %#v", result)
	}
	if !strings.Contains(result.SubtitleText, "番茄") || result.SummaryMode != "heuristic" {
		t.Fatalf("unexpected direct parse summary: %#v", result)
	}

	transport.assertRequests(t, []string{
		"api.bilibili.com/x/web-interface/view",
		"api.bilibili.com/x/player/v2",
		"i0.hdslb.com/subtitle/demo.json",
	})
	transport.assertCookies(t, "SESSDATA=sess-direct")
}

func TestPreviewBilibiliDirectModeResolvesShortLink(t *testing.T) {
	t.Parallel()

	transport := newBilibiliDirectTransport(t)
	resolveClient := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		resolvedURL, err := url.Parse("https://www.bilibili.com/video/BV1xx411c7mD?p=2")
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    &http.Request{Method: request.Method, URL: resolvedURL, Header: request.Header.Clone()},
		}, nil
	})}
	service := NewService(Options{
		HTTPClient:       &http.Client{Transport: transport},
		ResolveURLClient: resolveClient,
	})

	result, err := service.PreviewBilibili(context.Background(), "https://b23.tv/demo123")
	if err != nil {
		t.Fatalf("PreviewBilibili() error = %v", err)
	}
	if result.CanonicalURL != "https://www.bilibili.com/video/BV1xx411c7mD?p=2" {
		t.Fatalf("CanonicalURL = %q", result.CanonicalURL)
	}
	if result.Title != "番茄牛腩" || result.CoverURL != "https://i0.hdslb.com/demo.jpg" {
		t.Fatalf("unexpected preview: %#v", result)
	}
}

type bilibiliDirectTransport struct {
	testingT *testing.T
	mu       sync.Mutex
	requests []string
	cookies  []string
}

func newBilibiliDirectTransport(t *testing.T) *bilibiliDirectTransport {
	t.Helper()
	return &bilibiliDirectTransport{testingT: t}
}

func (transport *bilibiliDirectTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.testingT.Helper()
	if got := request.Header.Get("Cookie"); got != "SESSDATA=sess-direct" && got != "" {
		transport.testingT.Fatalf("unexpected Cookie for %s: %q", request.URL, got)
	}

	key := request.URL.Hostname() + request.URL.Path
	transport.mu.Lock()
	transport.requests = append(transport.requests, key)
	transport.cookies = append(transport.cookies, request.Header.Get("Cookie"))
	transport.mu.Unlock()

	var payload string
	switch key {
	case "api.bilibili.com/x/web-interface/view":
		payload = `{
			"code": 0,
			"data": {
				"title": "番茄牛腩",
				"desc": "牛腩 500克，番茄 3个",
				"pic": "https://i0.hdslb.com/demo.jpg",
				"bvid": "BV1xx411c7mD",
				"aid": 10086,
				"owner": {"name": "厨房UP"},
				"pages": [
					{"cid": 10086, "page": 1, "part": "第一集"},
					{"cid": 20086, "page": 2, "part": "第二集"}
				]
			}
		}`
	case "api.bilibili.com/x/player/v2":
		payload = `{
			"code": 0,
			"data": {
				"subtitle": {
					"subtitles": [
						{"lan": "zh-CN", "lan_doc": "中文", "subtitle_url": "https://i0.hdslb.com/subtitle/demo.json"}
					]
				}
			}
		}`
	case "i0.hdslb.com/subtitle/demo.json":
		payload = `{
			"body": [
				{"from": 0, "to": 1, "content": "牛腩 500克"},
				{"from": 1, "to": 2, "content": "番茄 3个"},
				{"from": 2, "to": 3, "content": "先焯水，再慢炖至软烂"}
			]
		}`
	default:
		transport.testingT.Fatalf("unexpected direct Bilibili request: %s", request.URL)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(payload)),
		Request:    request,
	}, nil
}

func (transport *bilibiliDirectTransport) assertCookies(t *testing.T, expected string) {
	t.Helper()
	transport.mu.Lock()
	defer transport.mu.Unlock()
	for index, cookie := range transport.cookies {
		if cookie != expected {
			t.Fatalf("request %d Cookie = %q, want %q", index, cookie, expected)
		}
	}
}

func (transport *bilibiliDirectTransport) assertRequests(t *testing.T, expected []string) {
	t.Helper()
	transport.mu.Lock()
	defer transport.mu.Unlock()
	if len(transport.requests) != len(expected) {
		t.Fatalf("requests = %#v, want %#v", transport.requests, expected)
	}
	for index := range expected {
		if transport.requests[index] != expected[index] {
			t.Fatalf("requests = %#v, want %#v", transport.requests, expected)
		}
	}
}
