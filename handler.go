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
		_, _ = fmt.Fprint(w, RenderPreparingPage(subdomain))
	}
	proxy.ServeHTTP(w, r)
}

func (h *Handler) handleStarting(w http.ResponseWriter, subdomain string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, RenderPreparingPage(subdomain))
}

func (h *Handler) handleNotFound(w http.ResponseWriter, subdomain string) {
	// Re-check periodically in case image was pushed
	if h.state.NeedsNotFoundRecheck(subdomain) {
		go h.recheckNotFound(subdomain)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprint(w, RenderNotFoundPage(subdomain))
}

func (h *Handler) handleUnknown(w http.ResponseWriter, _ *http.Request, subdomain string) {
	log.Printf("[%s] first request, checking registry for image", subdomain)
	digest, err := h.registryCheck(subdomain)
	if err != nil {
		log.Printf("[%s] registry check failed: %v", subdomain, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if digest == "" {
		log.Printf("[%s] image not found in registry", subdomain)
		h.state.SetNotFound(subdomain)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, RenderNotFoundPage(subdomain))
		return
	}

	log.Printf("[%s] image found in registry (digest: %s), starting stack", subdomain, digest)
	h.state.SetStarting(subdomain)
	h.startStack(subdomain, digest)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, RenderPreparingPage(subdomain))
}

func (h *Handler) checkAndUpdate(subdomain, currentDigest string) {
	log.Printf("[%s] checking registry for image updates (current digest: %s)", subdomain, currentDigest)
	digest, err := h.registryCheck(subdomain)
	if err != nil {
		log.Printf("[%s] background digest check failed: %v", subdomain, err)
		return
	}
	if digest != "" && digest != currentDigest {
		log.Printf("[%s] image updated: %s -> %s", subdomain, currentDigest, digest)
		h.state.UpdateDigest(subdomain, digest)
		// Trigger pull+restart via compose manager (caller wires this up)
	} else {
		log.Printf("[%s] image unchanged (digest: %s)", subdomain, currentDigest)
		h.state.UpdateDigest(subdomain, currentDigest)
	}
}

func (h *Handler) recheckNotFound(subdomain string) {
	log.Printf("[%s] rechecking registry for previously missing image", subdomain)
	digest, err := h.registryCheck(subdomain)
	if err != nil {
		log.Printf("[%s] recheck failed: %v", subdomain, err)
		return
	}
	if digest != "" {
		log.Printf("[%s] image now available (digest: %s), starting stack", subdomain, digest)
		h.state.SetStarting(subdomain)
		h.startStack(subdomain, digest)
	} else {
		log.Printf("[%s] image still not found", subdomain)
		h.state.SetNotFound(subdomain) // reset the timer
	}
}
