package linkparse

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type bilibiliFetchOptions struct {
	IncludeTranscript bool
}

func (s *Service) fetchBilibiliViaSidecar(ctx context.Context, rawInput string, opts bilibiliFetchOptions) (BilibiliParseResult, error) {
	sidecar := s.sidecarFor(ctx)
	if s == nil || sidecar == nil {
		return BilibiliParseResult{}, common.NewAppError(common.CodeInternalServer, "linkparse sidecar is not configured", http.StatusInternalServerError)
	}

	inputURL, err := extractInputURL(rawInput)
	if err != nil {
		return BilibiliParseResult{}, err
	}

	parsed, err := sidecar.parse(ctx, "/v1/parse/bilibili", sidecarParseRequest{
		Input:             rawInput,
		IncludeDebug:      false,
		IncludeTranscript: opts.IncludeTranscript,
	}, map[string]string{
		"X-Bilibili-SESSDATA": s.currentSessdata(ctx),
	})
	if err != nil {
		if isLinkparseSidecarTimeout(err) {
			return BilibiliParseResult{}, common.NewAppError(common.CodeBadRequest, "bilibili sidecar timed out", http.StatusBadRequest).WithErr(err)
		}
		var appErr *common.AppError
		if errors.As(err, &appErr) {
			return BilibiliParseResult{}, err
		}
		return BilibiliParseResult{}, common.NewAppError(common.CodeBadRequest, "request to bilibili sidecar failed", http.StatusBadRequest).WithErr(err)
	}

	result := BilibiliParseResult{
		Source:            "bilibili",
		Link:              firstNonEmpty(parsed.Normalized.CanonicalURL, parsed.Normalized.ShareURL, inputURL),
		Title:             strings.TrimSpace(parsed.Content.Title),
		Description:       strings.TrimSpace(parsed.Content.Description),
		Part:              strings.TrimSpace(parsed.Content.Part),
		Author:            strings.TrimSpace(parsed.Content.Author.Name),
		CoverURL:          strings.TrimSpace(parsed.Content.CoverURL),
		BVID:              strings.TrimSpace(parsed.Normalized.BVID),
		AID:               parsed.Normalized.AID,
		CID:               parsed.Normalized.CID,
		Page:              parsed.Normalized.Page,
		SubtitleAvailable: strings.TrimSpace(parsed.Content.Transcript) != "",
		SubtitleLanguage:  strings.TrimSpace(parsed.Content.SubtitleLanguage),
		SubtitleSegments:  parsed.Content.SubtitleSegments,
		SubtitleText:      strings.TrimSpace(parsed.Content.Transcript),
		Warnings:          parsed.Warnings,
	}

	return result, nil
}
