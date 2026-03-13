package appsettings

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/linkparse"
)

type AccessAuthorizer func(context.Context, int64) error

type Service struct {
	repo      *Repository
	cipherBox *cipherBox
	parser    *linkparse.Service
	authorize AccessAuthorizer
}

func NewService(repo *Repository, secret string, parser *linkparse.Service, authorize AccessAuthorizer) *Service {
	return &Service{
		repo:      repo,
		cipherBox: newCipherBox(secret),
		parser:    parser,
		authorize: authorize,
	}
}

func (s *Service) GetBilibiliSession(ctx context.Context, userID int64) (BilibiliSessionSetting, error) {
	if err := s.ensureAuthorized(ctx, userID); err != nil {
		return BilibiliSessionSetting{}, err
	}

	record, err := s.repo.GetBilibiliSession(ctx)
	if err != nil {
		return BilibiliSessionSetting{}, err
	}
	return buildBilibiliSessionSetting(record), nil
}

func (s *Service) UpdateBilibiliSession(ctx context.Context, userID int64, rawSessdata string) (BilibiliSessionSetting, error) {
	if err := s.ensureAuthorized(ctx, userID); err != nil {
		return BilibiliSessionSetting{}, err
	}

	sessdata, err := normalizeSessdata(rawSessdata)
	if err != nil {
		return BilibiliSessionSetting{}, err
	}

	if s.parser == nil {
		return BilibiliSessionSetting{}, common.ErrInternal
	}
	if err := s.parser.VerifyBilibiliSessdata(ctx, sessdata); err != nil {
		return BilibiliSessionSetting{}, err
	}

	ciphertext, err := s.cipherBox.Encrypt(sessdata)
	if err != nil {
		return BilibiliSessionSetting{}, common.ErrInternal.WithErr(err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	record := bilibiliSessionRecord{
		SessdataCiphertext: ciphertext,
		MaskedSessdata:     maskSessdata(sessdata),
		Status:             BilibiliSessionStatusValid,
		LastCheckedAt:      now,
		LastSuccessAt:      now,
		LastError:          "",
		UpdatedBy:          userID,
		UpdatedAt:          now,
	}
	if err := s.repo.UpsertBilibiliSession(ctx, record); err != nil {
		return BilibiliSessionSetting{}, err
	}

	return buildBilibiliSessionSetting(record), nil
}

func (s *Service) ClearBilibiliSession(ctx context.Context, userID int64) (BilibiliSessionSetting, error) {
	if err := s.ensureAuthorized(ctx, userID); err != nil {
		return BilibiliSessionSetting{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	record := bilibiliSessionRecord{
		Status:    BilibiliSessionStatusUnconfigured,
		UpdatedBy: userID,
		UpdatedAt: now,
	}
	if err := s.repo.UpsertBilibiliSession(ctx, record); err != nil {
		return BilibiliSessionSetting{}, err
	}
	return buildBilibiliSessionSetting(record), nil
}

func (s *Service) CurrentBilibiliSessdata(ctx context.Context) (string, error) {
	record, err := s.repo.GetBilibiliSession(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(record.SessdataCiphertext) == "" {
		return "", nil
	}

	sessdata, err := s.cipherBox.Decrypt(record.SessdataCiphertext)
	if err != nil {
		return "", common.ErrInternal.WithErr(err)
	}
	return strings.TrimSpace(sessdata), nil
}

func normalizeSessdata(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", common.NewAppError(common.CodeBadRequest, "SESSDATA is required", http.StatusBadRequest)
	}

	if strings.Contains(value, "SESSDATA=") || strings.Contains(value, ";") {
		parts := strings.Split(value, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			key, val, ok := strings.Cut(part, "=")
			if ok && strings.EqualFold(strings.TrimSpace(key), "SESSDATA") {
				value = strings.TrimSpace(val)
				break
			}
		}
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", common.NewAppError(common.CodeBadRequest, "SESSDATA is required", http.StatusBadRequest)
	}

	return value, nil
}

func maskSessdata(sessdata string) string {
	if len(sessdata) <= 8 {
		return "****"
	}
	return sessdata[:4] + "..." + sessdata[len(sessdata)-4:]
}

func buildBilibiliSessionSetting(record bilibiliSessionRecord) BilibiliSessionSetting {
	configured := strings.TrimSpace(record.SessdataCiphertext) != ""
	status := strings.TrimSpace(record.Status)
	if status == "" {
		status = BilibiliSessionStatusUnconfigured
	}

	return BilibiliSessionSetting{
		Configured:     configured,
		Status:         status,
		MaskedSessdata: strings.TrimSpace(record.MaskedSessdata),
		LastCheckedAt:  strings.TrimSpace(record.LastCheckedAt),
		LastSuccessAt:  strings.TrimSpace(record.LastSuccessAt),
		LastError:      strings.TrimSpace(record.LastError),
		UpdatedAt:      strings.TrimSpace(record.UpdatedAt),
	}
}

func (s *Service) ensureAuthorized(ctx context.Context, userID int64) error {
	if s.authorize == nil {
		return nil
	}
	return s.authorize(ctx, userID)
}
