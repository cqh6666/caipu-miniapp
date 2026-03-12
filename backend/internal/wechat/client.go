package wechat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

const code2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"

type Client interface {
	Code2Session(ctx context.Context, code string) (Code2SessionResult, error)
}

type HTTPClient struct {
	appID     string
	appSecret string
	client    *http.Client
}

type Code2SessionResult struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid,omitempty"`
}

type code2SessionResponse struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func NewClient(appID, appSecret string) *HTTPClient {
	return &HTTPClient{
		appID:     appID,
		appSecret: appSecret,
		client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c *HTTPClient) Code2Session(ctx context.Context, code string) (Code2SessionResult, error) {
	if c.appID == "" || c.appSecret == "" {
		return Code2SessionResult{}, common.NewAppError(common.CodeInternalServer, "wechat login is not configured", http.StatusServiceUnavailable)
	}

	query := url.Values{
		"appid":      []string{c.appID},
		"secret":     []string{c.appSecret},
		"js_code":    []string{code},
		"grant_type": []string{"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, code2SessionURL+"?"+query.Encode(), nil)
	if err != nil {
		return Code2SessionResult{}, fmt.Errorf("create wechat request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return Code2SessionResult{}, fmt.Errorf("call wechat code2session: %w", err)
	}
	defer resp.Body.Close()

	var body code2SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Code2SessionResult{}, fmt.Errorf("decode wechat response: %w", err)
	}

	if body.ErrCode != 0 {
		return Code2SessionResult{}, common.NewAppError(common.CodeUnauthorized, "wechat login failed", http.StatusUnauthorized).
			WithErr(fmt.Errorf("wechat errcode=%d errmsg=%s", body.ErrCode, body.ErrMsg))
	}

	if body.OpenID == "" {
		return Code2SessionResult{}, common.NewAppError(common.CodeUnauthorized, "wechat login failed", http.StatusUnauthorized)
	}

	return Code2SessionResult{
		OpenID:     body.OpenID,
		SessionKey: body.SessionKey,
		UnionID:    body.UnionID,
	}, nil
}
