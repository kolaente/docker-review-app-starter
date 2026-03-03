---
# docker-review-app-starter-lhep
title: Implement HTTP handler and reverse proxy
status: todo
type: task
created_at: 2026-03-03T14:01:28Z
updated_at: 2026-03-03T14:01:28Z
parent: docker-review-app-starter-014k
blocked_by:
    - docker-review-app-starter-zaw5
    - docker-review-app-starter-tfp9
    - docker-review-app-starter-2kmq
---

## Implement HTTP handler and reverse proxy

The main HTTP handler that ties everything together: extracts subdomain, checks state, serves status pages or proxies to running containers.

### Files
- Create: `handler.go`
- Create: `handler_test.go`

### Step 1: Write failing test for the handler

Create `handler_test.go`:

```go
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
```

Run: `go test ./... -run TestHandler -v`
Expected: FAIL.

### Step 2: Implement the handler

Create `handler.go`:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// RegistryCheckFunc checks if an image exists for a subdomain.
// Returns the digest if found, empty string if not.
type RegistryCheckFunc func(subdomain string) (string, error)

// StartStackFunc starts a stack for the given subdomain in the background.
type StartStackFunc func(subdomain string, digest string)

type Handler struct {
	domain        string
	state         *StateManager
	registryCheck RegistryCheckFunc
	startStack    StartStackFunc
	targetService string
	targetPort    int
}

func NewHandler(domain string, state *StateManager, registryCheck RegistryCheckFunc, startStack StartStackFunc) *Handler {
	return &Handler{
		domain:        domain,
		state:         state,
		registryCheck: registryCheck,
		startStack:    startStack,
		targetService: "app",
		targetPort:    8080,
	}
}

func (h *Handler) SetTarget(service string, port int) {
	h.targetService = service
	h.targetPort = port
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	subdomain, err := ExtractSubdomain(r.Host, h.domain)
	if err != nil {
		http.Error(w, "Invalid host", http.StatusBadRequest)
		return
	}

	state := h.state.GetState(subdomain)

	switch state.Status {
	case StatusRunning:
		h.handleRunning(w, r, subdomain, state)
	case StatusStarting:
		h.handleStarting(w, subdomain)
	case StatusNotFound:
		h.handleNotFound(w, subdomain)
	case StatusStopping:
		// Treat like starting — will restart after teardown
		h.handleStarting(w, subdomain)
	case StatusUnknown:
		h.handleUnknown(w, r, subdomain)
	}
}

func (h *Handler) handleRunning(w http.ResponseWriter, r *http.Request, subdomain string, state StackState) {
	h.state.Touch(subdomain)

	// Background digest check if stale
	if h.state.NeedsDigestCheck(subdomain) {
		go h.checkAndUpdate(subdomain, state.Digest)
	}

	// Reverse proxy to the container
	target := fmt.Sprintf("http://%s-%s-1:%d", projectName(subdomain), h.targetService, h.targetPort)
	u, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error for %s: %v", subdomain, err)
		// Container might not be ready yet despite state being "running"
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, RenderPreparingPage(subdomain))
	}
	proxy.ServeHTTP(w, r)
}

func (h *Handler) handleStarting(w http.ResponseWriter, subdomain string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, RenderPreparingPage(subdomain))
}

func (h *Handler) handleNotFound(w http.ResponseWriter, subdomain string) {
	// Re-check periodically in case image was pushed
	if h.state.NeedsNotFoundRecheck(subdomain) {
		go h.recheckNotFound(subdomain)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, RenderNotFoundPage(subdomain))
}

func (h *Handler) handleUnknown(w http.ResponseWriter, r *http.Request, subdomain string) {
	digest, err := h.registryCheck(subdomain)
	if err != nil {
		log.Printf("registry check failed for %s: %v", subdomain, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if digest == "" {
		h.state.SetNotFound(subdomain)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, RenderNotFoundPage(subdomain))
		return
	}

	h.state.SetStarting(subdomain)
	h.startStack(subdomain, digest)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, RenderPreparingPage(subdomain))
}

func (h *Handler) checkAndUpdate(subdomain, currentDigest string) {
	digest, err := h.registryCheck(subdomain)
	if err != nil {
		log.Printf("background digest check failed for %s: %v", subdomain, err)
		return
	}
	if digest != "" && digest != currentDigest {
		log.Printf("image updated for %s: %s -> %s", subdomain, currentDigest, digest)
		h.state.UpdateDigest(subdomain, digest)
		// Trigger pull+restart via compose manager (caller wires this up)
	} else {
		h.state.UpdateDigest(subdomain, currentDigest)
	}
}

func (h *Handler) recheckNotFound(subdomain string) {
	digest, err := h.registryCheck(subdomain)
	if err != nil {
		log.Printf("recheck failed for %s: %v", subdomain, err)
		return
	}
	if digest != "" {
		log.Printf("image now available for %s", subdomain)
		h.state.SetStarting(subdomain)
		h.startStack(subdomain, digest)
	} else {
		h.state.SetNotFound(subdomain) // reset the timer
	}
}
```

Run: `go test ./... -run TestHandler -v`
Expected: PASS.

### Step 3: Commit

```bash
git add handler.go handler_test.go
git commit -m "feat: add HTTP handler with reverse proxy and status pages"
```
