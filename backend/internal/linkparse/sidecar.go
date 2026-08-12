package linkparse

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/audit"
	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

func sidecarUnavailableError() error {
	return common.NewAppError(common.CodeServiceUnavailable, "linkparse sidecar is not configured", http.StatusServiceUnavailable)
}

type sidecarClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
	tracker audit.Tracker
}

type sidecarParseRequest struct {
	Input             string `json:"input"`
	Provider          string `json:"provider,omitempty"`
	IncludeDebug      bool   `json:"includeDebug"`
	IncludeTranscript bool   `json:"includeTranscript"`
}

type sidecarParseResponse struct {
	OK                bool   `json:"ok"`
	Platform          string `json:"platform"`
	ProviderRequested string `json:"providerRequested"`
	ProviderUsed      string `json:"providerUsed"`
	Normalized        struct {
		ShareURL     string `json:"shareUrl"`
		CanonicalURL string `json:"canonicalUrl"`
		ID           string `json:"id"`
		XSECToken    string `json:"xsecToken"`
		BVID         string `json:"bvid"`
		AID          int64  `json:"aid"`
		CID          int64  `json:"cid"`
		Page         int    `json:"page"`
	} `json:"normalized"`
	Content struct {
		Title            string   `json:"title"`
		Description      string   `json:"description"`
		Body             string   `json:"body"`
		Part             string   `json:"part"`
		Transcript       string   `json:"transcript"`
		TranscriptStatus string   `json:"transcriptStatus"`
		TranscriptError  string   `json:"transcriptError"`
		Tags             []string `json:"tags"`
		Images           []string `json:"images"`
		Videos           []string `json:"videos"`
		CoverURL         string   `json:"coverUrl"`
		Author           struct {
			Name      string `json:"name"`
			AvatarURL string `json:"avatarUrl"`
		} `json:"author"`
		ContentType      string `json:"contentType"`
		Likes            int64  `json:"likes"`
		Comments         int64  `json:"comments"`
		Favorites        int64  `json:"favorites"`
		SubtitleLanguage string `json:"subtitleLanguage"`
		SubtitleSegments int    `json:"subtitleSegments"`
	} `json:"content"`
	Quality string `json:"quality"`
	Error   *struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Retryable bool   `json:"retryable"`
	} `json:"error,omitempty"`
	Warnings []string `json:"warnings"`
}

func (c *sidecarClient) verifyBilibiliSession(ctx context.Context, sessdata string) error {
	startedAt := time.Now()
	const path = "/v1/verify/bilibili-session"
	logCall := func(status string, httpStatus int, err error) {
		if c == nil || c.tracker == nil {
			return
		}
		jobCtx, ok := audit.CurrentJobContext(ctx)
		if !ok || jobCtx.JobRunID <= 0 {
			return
		}
		_ = c.tracker.LogCall(ctx, audit.CallLogInput{
			JobRunID:     jobCtx.JobRunID,
			Scene:        jobCtx.Scene,
			Provider:     "linkparse-sidecar",
			Endpoint:     path,
			Status:       status,
			HTTPStatus:   httpStatus,
			LatencyMS:    time.Since(startedAt).Milliseconds(),
			ErrorType:    audit.ErrorTypeFromError(err),
			ErrorMessage: errorMessage(err),
			RequestID:    common.RequestID(ctx),
		})
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, nil)
	if err != nil {
		logCall(audit.CallStatusFailed, 0, err)
		return common.ErrInternal.WithErr(err)
	}
	if strings.TrimSpace(c.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.apiKey))
	}
	req.Header.Set("X-Bilibili-SESSDATA", strings.TrimSpace(sessdata))

	resp, err := c.client.Do(req)
	if err != nil {
		callErr := common.NewAppError(common.CodeInternalServer, "request to bilibili sidecar failed", http.StatusBadGateway).WithErr(err)
		logCall(audit.CallStatusFromError(err), 0, callErr)
		return callErr
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 2049))
	if readErr != nil {
		callErr := common.NewAppError(common.CodeInternalServer, "invalid linkparse sidecar response", http.StatusBadGateway).WithErr(readErr)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return callErr
	}
	if len(data) > 2048 {
		callErr := common.NewAppError(common.CodeInternalServer, "linkparse sidecar response exceeded size limit", http.StatusBadGateway)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return callErr
	}

	var payload struct {
		OK    bool `json:"ok"`
		Valid bool `json:"valid"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		callErr := common.NewAppError(common.CodeInternalServer, "invalid linkparse sidecar response", http.StatusBadGateway).WithErr(err)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return callErr
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && payload.OK && payload.Valid {
		logCall(audit.CallStatusSuccess, resp.StatusCode, nil)
		return nil
	}

	if resp.StatusCode == http.StatusBadRequest {
		message := "当前 SESSDATA 无法获取 B 站字幕，请更新后重试"
		if payload.Error != nil && payload.Error.Code == "invalid_input" {
			message = "SESSDATA is required"
		}
		callErr := common.NewAppError(common.CodeBadRequest, message, http.StatusBadRequest)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
		return callErr
	}
	callErr := common.NewAppError(common.CodeServiceUnavailable, "bilibili session verification is unavailable", http.StatusServiceUnavailable)
	logCall(audit.CallStatusFailed, resp.StatusCode, callErr)
	return callErr
}

func (c *sidecarClient) parse(ctx context.Context, path string, payload sidecarParseRequest, extraHeaders map[string]string) (sidecarParseResponse, error) {
	startedAt := time.Now()
	logCall := func(status string, httpStatus int, err error, meta map[string]any) {
		if c == nil || c.tracker == nil {
			return
		}
		jobCtx, ok := audit.CurrentJobContext(ctx)
		if !ok || jobCtx.JobRunID <= 0 {
			return
		}
		_ = c.tracker.LogCall(ctx, audit.CallLogInput{
			JobRunID:     jobCtx.JobRunID,
			Scene:        jobCtx.Scene,
			Provider:     "linkparse-sidecar",
			Endpoint:     path,
			Model:        strings.TrimSpace(payload.Provider),
			Status:       status,
			HTTPStatus:   httpStatus,
			LatencyMS:    time.Since(startedAt).Milliseconds(),
			ErrorType:    audit.ErrorTypeFromError(err),
			ErrorMessage: errorMessage(err),
			RequestID:    common.RequestID(ctx),
			Meta:         meta,
		})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logCall(audit.CallStatusFailed, 0, err, nil)
		return sidecarParseResponse{}, common.ErrInternal.WithErr(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		logCall(audit.CallStatusFailed, 0, err, nil)
		return sidecarParseResponse{}, common.ErrInternal.WithErr(err)
	}

	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(c.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.apiKey))
	}
	for key, value := range extraHeaders {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logCall(audit.CallStatusFromError(err), 0, err, nil)
		return sidecarParseResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2049))
		callErr := mapSidecarHTTPError(resp.StatusCode, data)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr, nil)
		return sidecarParseResponse{}, callErr
	}

	var parsed sidecarParseResponse
	if err := decodeBoundedUpstreamJSON(resp.Body, maxSidecarResponseBytes, "linkparse sidecar", &parsed); err != nil {
		callErr := err
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr, nil)
		return sidecarParseResponse{}, callErr
	}
	if !parsed.OK {
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			callErr := sanitizedUpstreamError(common.CodeBadRequest, "linkparse sidecar parse failed", http.StatusBadRequest, parsed.Error.Message)
			logCall(audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
				"provider_used": strings.TrimSpace(parsed.ProviderUsed),
			})
			return sidecarParseResponse{}, callErr
		}
		callErr := common.NewAppError(common.CodeBadRequest, "linkparse sidecar parse failed", http.StatusBadRequest)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr, map[string]any{
			"provider_used": strings.TrimSpace(parsed.ProviderUsed),
		})
		return sidecarParseResponse{}, callErr
	}

	logCall(audit.CallStatusSuccess, resp.StatusCode, nil, map[string]any{
		"provider_requested": strings.TrimSpace(parsed.ProviderRequested),
		"provider_used":      strings.TrimSpace(parsed.ProviderUsed),
		"quality":            strings.TrimSpace(parsed.Quality),
	})

	return parsed, nil
}

func mapSidecarHTTPError(status int, data []byte) error {
	if len(data) > 2048 {
		return common.NewAppError(common.CodeInternalServer, "linkparse sidecar response exceeded size limit", http.StatusBadGateway)
	}

	var payload sidecarParseResponse
	_ = json.Unmarshal(data, &payload)
	errorCode := ""
	errorMessage := ""
	if payload.Error != nil {
		errorCode = strings.TrimSpace(payload.Error.Code)
		errorMessage = strings.TrimSpace(payload.Error.Message)
	}

	switch {
	case status == http.StatusBadRequest || errorCode == "invalid_input" || errorCode == "unsupported_url" || errorCode == "invalid_credentials":
		return sanitizedUpstreamError(common.CodeBadRequest, "linkparse sidecar rejected the request", http.StatusBadRequest, errorMessage)
	case status == http.StatusServiceUnavailable || errorCode == "provider_unavailable":
		return sanitizedUpstreamError(common.CodeServiceUnavailable, "linkparse sidecar provider is unavailable", http.StatusServiceUnavailable, errorMessage)
	default:
		return sanitizedUpstreamError(common.CodeInternalServer, "linkparse sidecar upstream failed", http.StatusBadGateway, errorMessage)
	}
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
