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
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = "linkparse sidecar request failed"
		}
		callErr := common.NewAppError(common.CodeBadRequest, message, http.StatusBadRequest)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr, nil)
		return sidecarParseResponse{}, callErr
	}

	var parsed sidecarParseResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		callErr := common.NewAppError(common.CodeBadRequest, "failed to decode linkparse sidecar response", http.StatusBadRequest).WithErr(err)
		logCall(audit.CallStatusFailed, resp.StatusCode, callErr, nil)
		return sidecarParseResponse{}, callErr
	}
	if !parsed.OK {
		if parsed.Error != nil && strings.TrimSpace(parsed.Error.Message) != "" {
			callErr := common.NewAppError(common.CodeBadRequest, strings.TrimSpace(parsed.Error.Message), http.StatusBadRequest)
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

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
