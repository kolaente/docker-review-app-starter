package main

import (
	"testing"
	"time"
)

func TestStackStateTransitions(t *testing.T) {
	sm := NewStateManager(1*time.Second, 30*time.Second) // 1s timeout for testing

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
	sm := NewStateManager(5*time.Minute, 30*time.Second)

	sm.SetNotFound("pr-99")
	state := sm.GetState("pr-99")
	if state.Status != StatusNotFound {
		t.Errorf("expected not_found, got %v", state.Status)
	}
}

func backdateDigestCheck(sm *StateManager, subdomain string, d time.Duration) {
	sm.mu.Lock()
	sm.stacks[subdomain].LastDigestCheck = time.Now().Add(-d)
	sm.mu.Unlock()
}

func TestStackStateClaimDigestCheck(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)

	sm.SetRunning("pr-42", "sha256:abc123")

	if sm.ClaimDigestCheck("pr-42") {
		t.Error("should not claim a digest check immediately after start")
	}

	backdateDigestCheck(sm, "pr-42", time.Minute)

	if !sm.ClaimDigestCheck("pr-42") {
		t.Error("should claim a digest check once the interval elapsed")
	}

	// A second claim must not run while the first is in flight, even if stale
	backdateDigestCheck(sm, "pr-42", time.Minute)
	if sm.ClaimDigestCheck("pr-42") {
		t.Error("should not claim a digest check while one is in flight")
	}

	sm.ReleaseDigestCheck("pr-42", "sha256:def456")
	if got := sm.GetState("pr-42").Digest; got != "sha256:def456" {
		t.Errorf("expected digest sha256:def456, got %s", got)
	}
	if sm.GetState("pr-42").Updating {
		t.Error("expected updating flag cleared after release")
	}
}

func TestStackStateReleaseDigestCheckKeepsDigestOnFailure(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)

	sm.SetRunning("pr-42", "sha256:abc123")
	backdateDigestCheck(sm, "pr-42", time.Minute)
	if !sm.ClaimDigestCheck("pr-42") {
		t.Fatal("expected to claim digest check")
	}

	sm.ReleaseDigestCheck("pr-42", "")
	if got := sm.GetState("pr-42").Digest; got != "sha256:abc123" {
		t.Errorf("expected digest unchanged after failed check, got %s", got)
	}
}

func TestStackStateClaimDigestCheckNotRunning(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)

	sm.SetStarting("pr-42")
	if sm.ClaimDigestCheck("pr-42") {
		t.Error("should not claim a digest check for a starting stack")
	}
	if sm.ClaimDigestCheck("pr-unknown") {
		t.Error("should not claim a digest check for an unknown stack")
	}
}
