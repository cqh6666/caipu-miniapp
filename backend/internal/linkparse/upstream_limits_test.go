package linkparse

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upstream"
)

func TestSidecarRejectsOversizedJSON(t *testing.T) {
	server := oversizedJSONServer(t, maxSidecarResponseBytes+1)
	defer server.Close()

	client := &sidecarClient{baseURL: server.URL, client: server.Client()}
	_, err := client.parse(context.Background(), "/v1/parse/bilibili", sidecarParseRequest{Input: "https://b23.tv/demo"}, nil)
	assertUpstreamLimitError(t, err, "linkparse sidecar response exceeded size limit")
}

func TestBilibiliRejectsOversizedJSON(t *testing.T) {
	server := oversizedJSONServer(t, maxBilibiliResponseBytes+1)
	defer server.Close()

	service := &Service{httpClient: server.Client()}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	var target map[string]any
	err = service.doJSON(req, &target)
	assertUpstreamLimitError(t, err, "bilibili upstream response exceeded size limit")
}

func oversizedJSONServer(t *testing.T, size int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		chunk := strings.Repeat("x", 32*1024)
		for remaining := size; remaining > 0; {
			part := int64(len(chunk))
			if part > remaining {
				part = remaining
			}
			if _, err := w.Write([]byte(chunk[:part])); err != nil {
				return
			}
			remaining -= part
		}
	}))
}

func assertUpstreamLimitError(t *testing.T, err error, message string) {
	t.Helper()
	var appErr *common.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadGateway || appErr.Message != message {
		t.Fatalf("unexpected error: %T %v", err, err)
	}
	if !errors.Is(err, upstream.ErrResponseTooLarge) {
		t.Fatalf("error does not retain size cause: %v", err)
	}
}
