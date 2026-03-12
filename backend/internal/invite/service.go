package invite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
	"github.com/cqh6666/caipu-miniapp/backend/internal/kitchen"
	"github.com/cqh6666/caipu-miniapp/backend/internal/profile"
)

const (
	maxInviteExpireHours = 720
	maxInviteUses        = 20
	inviteCodeLength     = 8
)

const inviteCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type Service struct {
	repo               *Repository
	kitchen            *kitchen.Service
	defaultExpireHours int
	defaultMaxUses     int
}

func NewService(repo *Repository, kitchenService *kitchen.Service, defaultExpireHours, defaultMaxUses int) *Service {
	if defaultExpireHours <= 0 {
		defaultExpireHours = 72
	}
	if defaultMaxUses <= 0 {
		defaultMaxUses = 10
	}

	return &Service{
		repo:               repo,
		kitchen:            kitchenService,
		defaultExpireHours: defaultExpireHours,
		defaultMaxUses:     defaultMaxUses,
	}
}

func (s *Service) Create(ctx context.Context, userID, kitchenID int64, req createInviteRequest) (Invite, error) {
	if err := s.kitchen.EnsureMember(ctx, userID, kitchenID); err != nil {
		return Invite{}, err
	}

	maxUses, expireHours, err := s.normalizeCreateRequest(req)
	if err != nil {
		return Invite{}, err
	}

	token, err := common.NewPrefixedID("inv")
	if err != nil {
		return Invite{}, fmt.Errorf("generate invite token: %w", err)
	}

	code, err := s.generateInviteCode(ctx)
	if err != nil {
		return Invite{}, err
	}

	now := time.Now()
	record, err := s.repo.Create(ctx, createInviteParams{
		KitchenID:     kitchenID,
		InviterUserID: userID,
		Token:         token,
		Code:          code,
		Status:        statusActive,
		MaxUses:       maxUses,
		ExpiresAt:     now.Add(time.Duration(expireHours) * time.Hour).Format(time.RFC3339),
		CreatedAt:     now.Format(time.RFC3339),
	})
	if err != nil {
		return Invite{}, err
	}

	return toInvite(record), nil
}

func (s *Service) Preview(ctx context.Context, token string) (Invite, error) {
	record, err := s.findByToken(ctx, token)
	if err != nil {
		return Invite{}, err
	}

	return toInvite(record), nil
}

func (s *Service) PreviewByCode(ctx context.Context, code string) (Invite, error) {
	record, err := s.findByCode(ctx, code)
	if err != nil {
		return Invite{}, err
	}

	return toInvite(record), nil
}

func (s *Service) Accept(ctx context.Context, userID int64, token string) (AcceptResult, error) {
	record, err := s.findByToken(ctx, token)
	if err != nil {
		return AcceptResult{}, err
	}

	return s.acceptRecord(ctx, userID, record)
}

func (s *Service) AcceptByCode(ctx context.Context, userID int64, code string) (AcceptResult, error) {
	record, err := s.findByCode(ctx, code)
	if err != nil {
		return AcceptResult{}, err
	}

	return s.acceptRecord(ctx, userID, record)
}

func (s *Service) acceptRecord(ctx context.Context, userID int64, record inviteRecord) (AcceptResult, error) {
	status, err := effectiveStatus(record)
	if err != nil {
		return AcceptResult{}, err
	}

	switch status {
	case statusExpired:
		return AcceptResult{}, common.NewAppError(common.CodeConflict, "invite has expired", http.StatusConflict)
	case statusUsedUp:
		return AcceptResult{}, common.NewAppError(common.CodeConflict, "invite has reached its usage limit", http.StatusConflict)
	case statusRevoked:
		return AcceptResult{}, common.NewAppError(common.CodeConflict, "invite is no longer available", http.StatusConflict)
	}

	result, err := s.repo.Accept(ctx, userID, record)
	if err != nil {
		return AcceptResult{}, err
	}

	if !result.AlreadyMember {
		record.UsedCount++
		if record.UsedCount >= record.MaxUses {
			record.Status = statusUsedUp
		}
	}

	kitchens, err := s.kitchen.ListByUserID(ctx, userID)
	if err != nil {
		return AcceptResult{}, fmt.Errorf("list kitchens after accept invite: %w", err)
	}

	return AcceptResult{
		Invite: toInvite(record),
		Kitchen: kitchen.Summary{
			ID:   result.KitchenID,
			Name: result.KitchenName,
			Role: result.Role,
		},
		Kitchens:         kitchens,
		CurrentKitchenID: result.KitchenID,
		AlreadyMember:    result.AlreadyMember,
	}, nil
}

func (s *Service) findByToken(ctx context.Context, token string) (inviteRecord, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return inviteRecord{}, common.NewAppError(common.CodeBadRequest, "invite token is required", http.StatusBadRequest)
	}

	record, err := s.repo.FindByToken(ctx, token)
	if errors.Is(err, sql.ErrNoRows) {
		return inviteRecord{}, common.ErrNotFound
	}
	if err != nil {
		return inviteRecord{}, err
	}

	return record, nil
}

func (s *Service) findByCode(ctx context.Context, code string) (inviteRecord, error) {
	code = normalizeInviteCode(code)
	if code == "" {
		return inviteRecord{}, common.NewAppError(common.CodeBadRequest, "invite code is required", http.StatusBadRequest)
	}

	record, err := s.repo.FindByCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return inviteRecord{}, common.ErrNotFound
	}
	if err != nil {
		return inviteRecord{}, err
	}

	return record, nil
}

func (s *Service) normalizeCreateRequest(req createInviteRequest) (maxUses int, expireHours int, err error) {
	maxUses = req.MaxUses
	if maxUses <= 0 {
		maxUses = s.defaultMaxUses
	}
	if maxUses <= 0 || maxUses > maxInviteUses {
		return 0, 0, common.NewAppError(common.CodeBadRequest, "maxUses must be between 1 and 20", http.StatusBadRequest)
	}

	expireHours = req.ExpiresInHours
	if expireHours <= 0 {
		expireHours = s.defaultExpireHours
	}
	if expireHours <= 0 || expireHours > maxInviteExpireHours {
		return 0, 0, common.NewAppError(common.CodeBadRequest, "expiresInHours must be between 1 and 720", http.StatusBadRequest)
	}

	return maxUses, expireHours, nil
}

func toInvite(record inviteRecord) Invite {
	status, _ := effectiveStatus(record)
	remainingUses := record.MaxUses - record.UsedCount
	if remainingUses < 0 {
		remainingUses = 0
	}

	return Invite{
		ID:            record.ID,
		KitchenID:     record.KitchenID,
		KitchenName:   record.KitchenName,
		Token:         record.Token,
		Code:          record.Code,
		Status:        status,
		MaxUses:       record.MaxUses,
		UsedCount:     record.UsedCount,
		RemainingUses: remainingUses,
		ExpiresAt:     record.ExpiresAt,
		CreatedAt:     record.CreatedAt,
		SharePath:     "/pages/invite/index?token=" + url.QueryEscape(record.Token),
		Inviter: Inviter{
			ID:       record.InviterUserID,
			Nickname: profile.DisplayName(record.InviterNickname, record.InviterUserID, ""),
		},
	}
}

func effectiveStatus(record inviteRecord) (string, error) {
	if record.Status == statusRevoked {
		return statusRevoked, nil
	}
	if record.UsedCount >= record.MaxUses {
		return statusUsedUp, nil
	}

	expiresAt, err := time.Parse(time.RFC3339, record.ExpiresAt)
	if err != nil {
		return "", common.ErrInternal.WithErr(fmt.Errorf("parse invite expiresAt: %w", err))
	}
	if time.Now().After(expiresAt) {
		return statusExpired, nil
	}

	if record.Status == "" {
		return statusActive, nil
	}

	return record.Status, nil
}

func normalizeInviteCode(value string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func (s *Service) generateInviteCode(ctx context.Context) (string, error) {
	for attempt := 0; attempt < 12; attempt++ {
		code, err := newInviteCode()
		if err != nil {
			return "", fmt.Errorf("generate invite code: %w", err)
		}

		_, err = s.repo.FindByCode(ctx, code)
		if errors.Is(err, sql.ErrNoRows) {
			return code, nil
		}
		if err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf("generate invite code: too many collisions")
}

func newInviteCode() (string, error) {
	buffer := make([]byte, inviteCodeLength)
	randomBytes := make([]byte, inviteCodeLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	for index, value := range randomBytes {
		buffer[index] = inviteCodeAlphabet[int(value)%len(inviteCodeAlphabet)]
	}

	return string(buffer), nil
}
