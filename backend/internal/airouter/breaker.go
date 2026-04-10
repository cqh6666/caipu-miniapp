package airouter

import (
	"fmt"
	"sync"
	"time"
)

type breakerState struct {
	consecutiveRetryableFailures int
	openUntil                    time.Time
}

type breakerStore struct {
	mu     sync.Mutex
	states map[string]breakerState
}

func newBreakerStore() *breakerStore {
	return &breakerStore{
		states: make(map[string]breakerState),
	}
}

func (s *breakerStore) key(scene Scene, providerID string) string {
	return fmt.Sprintf("%s:%s", scene, providerID)
}

func (s *breakerStore) state(scene Scene, providerID string) breakerState {
	return s.states[s.key(scene, providerID)]
}

func (s *breakerStore) isOpen(scene Scene, providerID string, now time.Time) (bool, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.state(scene, providerID)
	if state.openUntil.IsZero() || !state.openUntil.After(now) {
		if !state.openUntil.IsZero() {
			state.openUntil = time.Time{}
			s.states[s.key(scene, providerID)] = state
		}
		return false, time.Time{}
	}
	return true, state.openUntil
}

func (s *breakerStore) markFailure(scene Scene, providerID string, cfg BreakerConfig, now time.Time) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.key(scene, providerID)
	state := s.states[key]
	state.consecutiveRetryableFailures++
	if cfg.FailureThreshold > 0 && state.consecutiveRetryableFailures >= cfg.FailureThreshold {
		cooldown := time.Duration(cfg.CooldownSeconds) * time.Second
		if cooldown <= 0 {
			cooldown = time.Minute
		}
		state.openUntil = now.Add(cooldown)
	}
	s.states[key] = state
	return state.openUntil
}

func (s *breakerStore) markSuccess(scene Scene, providerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.key(scene, providerID)
	state := s.states[key]
	state.consecutiveRetryableFailures = 0
	state.openUntil = time.Time{}
	s.states[key] = state
}
