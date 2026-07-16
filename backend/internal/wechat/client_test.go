package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/upstream"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCode2SessionSuccess(t *testing.T) {
	client := NewClient("wx-app-id", "wx-secret")
	client.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.String() == "" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
		query := req.URL.Query()
		if query.Get("appid") != "wx-app-id" || query.Get("secret") != "wx-secret" ||
			query.Get("js_code") != "login-code" || query.Get("grant_type") != "authorization_code" {
			t.Fatalf("unexpected query: %s", req.URL.RawQuery)
		}
		return jsonResponse(req, http.StatusOK, `{"openid":"openid-1","session_key":"session-1","unionid":"union-1"}`), nil
	})

	result, err := client.Code2Session(context.Background(), "login-code")
	if err != nil {
		t.Fatalf("code2session: %v", err)
	}
	if result.OpenID != "openid-1" || result.SessionKey != "session-1" || result.UnionID != "union-1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestCode2SessionErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		appID      string
		secret     string
		response   string
		wantStatus int
		wantText   string
	}{
		{name: "missing config", wantStatus: http.StatusServiceUnavailable, wantText: "not configured"},
		{name: "wechat error", appID: "app", secret: "secret", response: `{"errcode":40029,"errmsg":"invalid code"}`, wantStatus: http.StatusUnauthorized, wantText: "wechat login failed"},
		{name: "missing openid", appID: "app", secret: "secret", response: `{"session_key":"session"}`, wantStatus: http.StatusUnauthorized, wantText: "wechat login failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.appID, tt.secret)
			client.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return jsonResponse(req, http.StatusOK, tt.response), nil
			})
			_, err := client.Code2Session(context.Background(), "code")
			var appErr *common.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected AppError, got %T %v", err, err)
			}
			if appErr.HTTPStatus != tt.wantStatus || !strings.Contains(appErr.Message, tt.wantText) {
				t.Fatalf("unexpected app error: %#v", appErr)
			}
		})
	}
}

func TestCode2SessionReportsTransportAndDecodeErrors(t *testing.T) {
	client := NewClient("app", "secret")
	client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})
	if _, err := client.Code2Session(context.Background(), "code"); err == nil || !strings.Contains(err.Error(), "call wechat code2session") {
		t.Fatalf("expected transport error, got %v", err)
	}

	client.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(req, http.StatusOK, "not-json"), nil
	})
	_, err := client.Code2Session(context.Background(), "code")
	var appErr *common.AppError
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadGateway || appErr.Message != "invalid wechat upstream response" {
		t.Fatalf("expected stable decode AppError, got %T %v", err, err)
	}
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected retained JSON cause, got %T %v", err, err)
	}
}

func TestCode2SessionRejectsOversizedResponse(t *testing.T) {
	client := NewClient("app", "secret")
	client.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(req, http.StatusOK, strings.Repeat("x", int(maxCode2SessionResponseBytes)+1)), nil
	})

	_, err := client.Code2Session(context.Background(), "code")
	var appErr *common.AppError
	if !errors.As(err, &appErr) || appErr.HTTPStatus != http.StatusBadGateway || appErr.Message != "wechat upstream response exceeded size limit" {
		t.Fatalf("unexpected error: %T %v", err, err)
	}
	if !errors.Is(err, upstream.ErrResponseTooLarge) {
		t.Fatalf("error does not retain size cause: %v", err)
	}
}

func jsonResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}
