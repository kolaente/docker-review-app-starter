---
# docker-review-app-starter-zaw5
title: Implement stack state manager with idle timeout
status: todo
type: task
created_at: 2026-03-03T14:00:28Z
updated_at: 2026-03-03T14:00:28Z
parent: docker-review-app-starter-014k
blocked_by:
    - docker-review-app-starter-tbgp
    - docker-review-app-starter-4tt9
---

## Implement stack state manager with idle timeout

The central state machine that tracks all known subdomains, their current state, idle timers, and image digests.

### Files
- Create: `state.go`
- Create: `state_test.go`

### Step 1: Write failing test for state transitions

Create `state_test.go`:

```go
package main

import (
	"sync"
	"testing"
	"time"
)

func TestStackStateTransitions(t *testing.T) {
	sm := NewStateManager(1 * time.Second) // 1s timeout for testing

	// Unknown subdomain returns unknown state
	state := sm.GetState("pr-42")
	if state.Status != StatusUnknown {
		t.Errorf("expected unknown, got %v", state.Status)
	}

	// Transition to starting
	sm.SetStarting("pr-42")
	state = sm.GetState("pr-42")
	if state.Status != StatusStarting {
		t.Errorf("expected starting, got %v", state.Status)
	}

	// Transition to running
	sm.SetRunning("pr-42", "sha256:abc123")
	state = sm.GetState("pr-42")
	if state.Status != StatusRunning {
		t.Errorf("expected running, got %v", state.Status)
	}
	if state.Digest != "sha256:abc123" {
		t.Errorf("expected digest sha256:abc123, got %s", state.Digest)
	}

	// Touch resets idle timer
	sm.Touch("pr-42")
}

func TestStackStateNotFound(t *testing.T) {
	sm := NewStateManager(5 * time.Minute)

	sm.SetNotFound("pr-99")
	state := sm.GetState("pr-99")
	if state.Status != StatusNotFound {
		t.Errorf("expected not_found, got %v", state.Status)
	}
}

func TestStackStateNeedsDigestCheck(t *testing.T) {
	sm := NewStateManager(5 * time.Minute)

	sm.SetRunning("pr-42", "sha256:abc123")

	// Just set to running, shouldn't need check yet
	if sm.NeedsDigestCheck("pr-42") {
		t.Error("should not need digest check immediately after start")
	}

	// Simulate time passing by backdating the last check
	sm.mu.Lock()
	sm.stacks["pr-42"].LastDigestCheck = time.Now().Add(-6 * time.Minute)
	sm.mu.Unlock()

	if !sm.NeedsDigestCheck("pr-42") {
		t.Error("should need digest check after 5+ minutes")
	}
}
```

Run: `go test ./... -run TestStackState -v`
Expected: FAIL.

### Step 2: Implement state manager

Create `state.go`:

```go
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
	Status         StackStatus
	Digest         string
	LastDigestCheck time.Time
	LastRequest    time.Time
	IdleTimer      *time.Timer
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
		Status:         StatusNotFound,
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
```

Run: `go test ./... -run TestStackState -v`
Expected: PASS.

### Step 3: Commit

```bash
git add state.go state_test.go
git commit -m "feat: add stack state manager with idle timeout tracking"
```
