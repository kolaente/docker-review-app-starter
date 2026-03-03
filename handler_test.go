package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerUnknownSubdomain_ImageNotFound(t *testing.T) {
	sm := NewStateManager(5 * time.Minute)

	// Mock registry that always returns "not found"
	registryCheck := func(subdomain string) (string, error) {
		return "", nil // empty digest = not found
	}

	h := NewHandler("review.example.com", sm, registryCheck, nil)

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
	sm := NewStateManager(5 * time.Minute)

	registryCheck := func(subdomain string) (string, error) {
		return "sha256:abc123", nil
	}

	startCalled := false
	startStack := func(subdomain string, digest string) {
		startCalled = true
	}

	h := NewHandler("review.example.com", sm, registryCheck, startStack)

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
	sm := NewStateManager(5 * time.Minute)
	sm.SetStarting("pr-42")

	h := NewHandler("review.example.com", sm, nil, nil)

	req := httptest.NewRequest("GET", "http://pr-42.review.example.com/", nil)
	req.Host = "pr-42.review.example.com"
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "refresh") {
		t.Error("response should contain auto-refresh for preparing page")
	}
}

func TestHandlerBadHost(t *testing.T) {
	sm := NewStateManager(5 * time.Minute)
	h := NewHandler("review.example.com", sm, nil, nil)

	req := httptest.NewRequest("GET", "http://other.example.com/", nil)
	req.Host = "other.example.com"
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
