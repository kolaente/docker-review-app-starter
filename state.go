package main

import (
	"sync"
	"time"
)

type StackStatus int

const (
	StatusUnknown StackStatus = iota
	StatusStarting
	StatusRunning
	StatusNotFound
	StatusStopping
)

type StackState struct {
	Status          StackStatus
	Digest          string
	LastDigestCheck time.Time
	LastRequest     time.Time
	IdleTimer       *time.Timer
}

type StateManager struct {
	mu          sync.Mutex
	stacks      map[string]*StackState
	idleTimeout time.Duration
	onIdle      func(subdomain string) // called when idle timer fires
}

func NewStateManager(idleTimeout time.Duration) *StateManager {
	return &StateManager{
		stacks:      make(map[string]*StackState),
		idleTimeout: idleTimeout,
	}
}

func (sm *StateManager) SetOnIdle(fn func(subdomain string)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.onIdle = fn
}

func (sm *StateManager) GetState(subdomain string) StackState {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.stacks[subdomain]
	if !ok {
		return StackState{Status: StatusUnknown}
	}
	return *s
}

func (sm *StateManager) SetStarting(subdomain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stacks[subdomain] = &StackState{
		Status:      StatusStarting,
		LastRequest: time.Now(),
	}
}

func (sm *StateManager) SetRunning(subdomain, digest string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.stacks[subdomain]
	if !ok {
		s = &StackState{}
		sm.stacks[subdomain] = s
	}
	s.Status = StatusRunning
	s.Digest = digest
	s.LastDigestCheck = time.Now()
	s.LastRequest = time.Now()
	sm.resetTimerLocked(subdomain, s)
}

func (sm *StateManager) SetNotFound(subdomain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.stacks[subdomain] = &StackState{
		Status:          StatusNotFound,
		LastDigestCheck: time.Now(),
	}
}

func (sm *StateManager) SetStopping(subdomain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.stacks[subdomain]; ok {
		s.Status = StatusStopping
		if s.IdleTimer != nil {
			s.IdleTimer.Stop()
		}
	}
}

func (sm *StateManager) Remove(subdomain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.stacks[subdomain]; ok {
		if s.IdleTimer != nil {
			s.IdleTimer.Stop()
		}
		delete(sm.stacks, subdomain)
	}
}

func (sm *StateManager) Touch(subdomain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.stacks[subdomain]; ok {
		s.LastRequest = time.Now()
		sm.resetTimerLocked(subdomain, s)
	}
}

func (sm *StateManager) NeedsDigestCheck(subdomain string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.stacks[subdomain]
	if !ok || s.Status != StatusRunning {
		return false
	}
	return time.Since(s.LastDigestCheck) > 5*time.Minute
}

func (sm *StateManager) UpdateDigest(subdomain, digest string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.stacks[subdomain]; ok {
		s.Digest = digest
		s.LastDigestCheck = time.Now()
	}
}

func (sm *StateManager) NeedsNotFoundRecheck(subdomain string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.stacks[subdomain]
	if !ok || s.Status != StatusNotFound {
		return false
	}
	return time.Since(s.LastDigestCheck) > 1*time.Minute
}

func (sm *StateManager) resetTimerLocked(subdomain string, s *StackState) {
	if s.IdleTimer != nil {
		s.IdleTimer.Stop()
	}
	s.IdleTimer = time.AfterFunc(sm.idleTimeout, func() {
		if sm.onIdle != nil {
			sm.onIdle(subdomain)
		}
	})
}
