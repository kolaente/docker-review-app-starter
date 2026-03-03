package main

import (
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
