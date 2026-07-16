package ratelimit

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cqh6666/caipu-miniapp/backend/internal/common"
)

type Policy struct {
	MaxAttempts int
	Window      time.Duration
	BanDuration time.Duration
}

type Guard struct {
	scope   string
	ip      *limiter
	subject *limiter
}

func NewGuard(scope string, ipPolicy, subjectPolicy Policy) *Guard {
	return newGuardWithClock(scope, ipPolicy, subjectPolicy, time.Now)
}

func newGuardWithClock(scope string, ipPolicy, subjectPolicy Policy, now func() time.Time) *Guard {
	return &Guard{
		scope:   strings.TrimSpace(scope),
		ip:      newLimiter(ipPolicy, now),
		subject: newLimiter(subjectPolicy, now),
	}
}

func (guard *Guard) Check(request *http.Request, subject string) error {
	if err := guard.CheckIP(request); err != nil {
		return err
	}
	return guard.CheckSubject(subject)
}

func (guard *Guard) CheckIP(request *http.Request) error {
	if guard == nil {
		return nil
	}
	if guard.ip != nil && !guard.ip.allow(guard.scope+":ip:"+clientIP(request)) {
		return tooManyRequestsError()
	}
	return nil
}

func (guard *Guard) CheckSubject(subject string) error {
	if guard == nil {
		return nil
	}
	if value := strings.TrimSpace(strings.ToLower(subject)); value != "" && guard.subject != nil {
		if !guard.subject.allow(guard.scope + ":subject:" + fingerprint(value)) {
			return tooManyRequestsError()
		}
	}
	return nil
}

func tooManyRequestsError() error {
	return common.NewAppError(
		common.CodeTooManyRequests,
		"too many requests, please try again later",
		http.StatusTooManyRequests,
	)
}

type limiter struct {
	mu          sync.Mutex
	policy      Policy
	now         func() time.Time
	entries     map[string]attemptState
	lastCleanup time.Time
}

type attemptState struct {
	windowStarted time.Time
	blockedUntil  time.Time
	lastSeen      time.Time
	attempts      int
}

func newLimiter(policy Policy, now func() time.Time) *limiter {
	if policy.MaxAttempts <= 0 || policy.Window <= 0 || policy.BanDuration <= 0 {
		return nil
	}
	if now == nil {
		now = time.Now
	}
	return &limiter{policy: policy, now: now, entries: make(map[string]attemptState)}
}

func (limiter *limiter) allow(key string) bool {
	if limiter == nil {
		return true
	}
	now := limiter.now()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	limiter.cleanup(now)
	state := limiter.entries[key]
	state.lastSeen = now
	if now.Before(state.blockedUntil) {
		limiter.entries[key] = state
		return false
	}
	if state.windowStarted.IsZero() || now.Sub(state.windowStarted) >= limiter.policy.Window {
		state.windowStarted = now
		state.attempts = 0
		state.blockedUntil = time.Time{}
	}
	state.attempts++
	if state.attempts > limiter.policy.MaxAttempts {
		state.blockedUntil = now.Add(limiter.policy.BanDuration)
		limiter.entries[key] = state
		return false
	}
	limiter.entries[key] = state
	return true
}

func (limiter *limiter) cleanup(now time.Time) {
	if len(limiter.entries) < 1024 || (!limiter.lastCleanup.IsZero() && now.Sub(limiter.lastCleanup) < time.Minute) {
		return
	}
	maxIdle := limiter.policy.Window + limiter.policy.BanDuration
	for key, state := range limiter.entries {
		if now.Sub(state.lastSeen) >= maxIdle && !now.Before(state.blockedUntil) {
			delete(limiter.entries, key)
		}
	}
	limiter.lastCleanup = now
}

func clientIP(request *http.Request) string {
	if request == nil {
		return "unknown"
	}
	value := strings.TrimSpace(request.RemoteAddr)
	if host, _, err := net.SplitHostPort(value); err == nil {
		return strings.Trim(strings.ToLower(host), "[]")
	}
	value = strings.Trim(value, "[]")
	if value == "" {
		return "unknown"
	}
	return strings.ToLower(value)
}

func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
