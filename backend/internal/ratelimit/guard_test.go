package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGuardLimitsByIPAndSubject(t *testing.T) {
	t.Parallel()

	policy := Policy{MaxAttempts: 2, Window: time.Minute, BanDuration: 5 * time.Minute}
	t.Run("IP", func(t *testing.T) {
		guard := NewGuard("login", policy, Policy{})
		for index, subject := range []string{"one", "two", "three"} {
			request := httptest.NewRequest(http.MethodPost, "/login", nil)
			request.RemoteAddr = "203.0.113.10:1234"
			err := guard.Check(request, subject)
			if index < 2 && err != nil {
				t.Fatalf("attempt %d error=%v", index, err)
			}
			if index == 2 && err == nil {
				t.Fatal("expected IP rate limit")
			}
		}
	})

	t.Run("subject", func(t *testing.T) {
		guard := NewGuard("login", Policy{}, policy)
		for index, address := range []string{"203.0.113.10:1", "203.0.113.11:2", "203.0.113.12:3"} {
			request := httptest.NewRequest(http.MethodPost, "/login", nil)
			request.RemoteAddr = address
			err := guard.Check(request, "same-account")
			if index < 2 && err != nil {
				t.Fatalf("attempt %d error=%v", index, err)
			}
			if index == 2 && err == nil {
				t.Fatal("expected subject rate limit")
			}
		}
	})
}

func TestGuardBanExpires(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 15, 20, 0, 0, 0, time.UTC)
	guard := newGuardWithClock(
		"login",
		Policy{MaxAttempts: 1, Window: time.Minute, BanDuration: 5 * time.Minute},
		Policy{},
		func() time.Time { return now },
	)
	request := httptest.NewRequest(http.MethodPost, "/login", nil)
	request.RemoteAddr = "203.0.113.10:1234"
	if err := guard.Check(request, ""); err != nil {
		t.Fatal(err)
	}
	if err := guard.Check(request, ""); err == nil {
		t.Fatal("expected temporary ban")
	}
	now = now.Add(4 * time.Minute)
	if err := guard.Check(request, ""); err == nil {
		t.Fatal("ban expired too early")
	}
	now = now.Add(2 * time.Minute)
	if err := guard.Check(request, ""); err != nil {
		t.Fatalf("ban did not expire: %v", err)
	}
}

func TestGuardIsConcurrencySafe(t *testing.T) {
	t.Parallel()

	guard := NewGuard(
		"accept",
		Policy{MaxAttempts: 5, Window: time.Minute, BanDuration: time.Minute},
		Policy{},
	)
	var allowed atomic.Int32
	var wait sync.WaitGroup
	for index := 0; index < 20; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			request := httptest.NewRequest(http.MethodPost, "/accept", nil)
			request.RemoteAddr = "203.0.113.10:1234"
			if guard.Check(request, "") == nil {
				allowed.Add(1)
			}
		}()
	}
	wait.Wait()
	if got := allowed.Load(); got != 5 {
		t.Fatalf("allowed=%d, want 5", got)
	}
}
