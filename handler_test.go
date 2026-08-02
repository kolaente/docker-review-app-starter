package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerUnknownSubdomain_ImageNotFound(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)

	// Mock registry that always returns "not found"
	registryCheck := func(subdomain string) (string, error) {
		return "", nil // empty digest = not found
	}

	h := NewHandler("review.example.com", sm, registryCheck, nil, nil)

	req := httptest.NewRequest("GET", "http://pr-99.review.example.com/", nil)
	req.Host = "pr-99.review.example.com"
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "pr-99") {
		t.Error("response should contain subdomain")
	}
}

func TestHandlerUnknownSubdomain_ImageExists(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)

	registryCheck := func(subdomain string) (string, error) {
		return "sha256:abc123", nil
	}

	startCalled := false
	startStack := func(subdomain string, digest string) {
		startCalled = true
	}

	h := NewHandler("review.example.com", sm, registryCheck, startStack, nil)

	req := httptest.NewRequest("GET", "http://pr-42.review.example.com/", nil)
	req.Host = "pr-42.review.example.com"
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "refresh") {
		t.Error("response should contain auto-refresh for preparing page")
	}
	if !startCalled {
		t.Error("startStack should have been called")
	}
}

func TestHandlerStarting(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)
	sm.SetStarting("pr-42")

	h := NewHandler("review.example.com", sm, nil, nil, nil)

	req := httptest.NewRequest("GET", "http://pr-42.review.example.com/", nil)
	req.Host = "pr-42.review.example.com"
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "refresh") {
		t.Error("response should contain auto-refresh for preparing page")
	}
}

func TestHandlerBadHost(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)
	h := NewHandler("review.example.com", sm, nil, nil, nil)

	req := httptest.NewRequest("GET", "http://other.example.com/", nil)
	req.Host = "other.example.com"
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerCheckAndUpdate_NewDigest(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)
	sm.SetRunning("pr-42", "sha256:old")

	registryCheck := func(subdomain string) (string, error) {
		return "sha256:new", nil
	}
	updated := make(chan string, 1)
	updateStack := func(subdomain string) error {
		updated <- subdomain
		return nil
	}

	h := NewHandler("review.example.com", sm, registryCheck, nil, updateStack)
	h.checkAndUpdate("pr-42", "sha256:old")

	select {
	case sub := <-updated:
		if sub != "pr-42" {
			t.Errorf("expected update for pr-42, got %s", sub)
		}
	default:
		t.Fatal("updateStack should have been called for a changed digest")
	}
	if got := sm.GetState("pr-42").Digest; got != "sha256:new" {
		t.Errorf("expected digest sha256:new, got %s", got)
	}
}

func TestHandlerCheckAndUpdate_SameDigest(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)
	sm.SetRunning("pr-42", "sha256:same")

	registryCheck := func(subdomain string) (string, error) {
		return "sha256:same", nil
	}
	updateStack := func(subdomain string) error {
		t.Error("updateStack should not be called when the digest is unchanged")
		return nil
	}

	h := NewHandler("review.example.com", sm, registryCheck, nil, updateStack)
	h.checkAndUpdate("pr-42", "sha256:same")
}

func TestHandlerCheckAndUpdate_UpdateFailureKeepsDigest(t *testing.T) {
	sm := NewStateManager(5*time.Minute, 30*time.Second)
	sm.SetRunning("pr-42", "sha256:old")

	registryCheck := func(subdomain string) (string, error) {
		return "sha256:new", nil
	}
	updateStack := func(subdomain string) error {
		return errors.New("pull failed")
	}

	h := NewHandler("review.example.com", sm, registryCheck, nil, updateStack)
	h.checkAndUpdate("pr-42", "sha256:old")

	// Digest must stay stale so the next check retries the update
	if got := sm.GetState("pr-42").Digest; got != "sha256:old" {
		t.Errorf("expected digest sha256:old after failed update, got %s", got)
	}
	if sm.GetState("pr-42").Updating {
		t.Error("expected updating flag cleared after failed update")
	}
}
