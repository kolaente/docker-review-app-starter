---
# docker-review-app-starter-er0o
title: Wire up main.go with all components
status: todo
type: task
created_at: 2026-03-03T14:01:48Z
updated_at: 2026-03-03T14:01:48Z
parent: docker-review-app-starter-014k
blocked_by:
    - docker-review-app-starter-lhep
---

## Wire up main.go with all components

Connect config, state manager, compose manager, registry client, and HTTP handler in main.go. Add startup recovery for existing stacks.

### Files
- Modify: `main.go`

### Step 1: Implement main.go

Update `main.go` to wire everything together:

```go
package main

import (
	"flag"
	"log"
	"net/http"
	"strings"
)

func main() {
	configPath := flag.String("config", "/etc/review-proxy/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Parse image reference pattern from template
	imagePattern, err := ParseTemplateImageRef(cfg.ComposeTemplate)
	if err != nil {
		log.Fatalf("failed to parse compose template: %v", err)
	}
	log.Printf("Image pattern: %s", imagePattern)

	// Initialize components
	composeMgr := &ComposeManager{TemplatePath: cfg.ComposeTemplate}
	registryClient := &RegistryClient{HTTPClient: http.DefaultClient}
	stateMgr := NewStateManager(cfg.IdleTimeout)

	// Registry check function: substitute subdomain into image pattern, parse, check
	registryCheck := func(subdomain string) (string, error) {
		imageStr := strings.ReplaceAll(imagePattern, "${SUBDOMAIN}", subdomain)
		ref, err := ParseImageRef(imageStr)
		if err != nil {
			return "", err
		}
		return registryClient.CheckTag(ref)
	}

	// Start stack function: runs compose up in background, updates state when done
	startStack := func(subdomain string, digest string) {
		go func() {
			log.Printf("Starting stack for %s", subdomain)
			if err := composeMgr.StartStack(subdomain); err != nil {
				log.Printf("Failed to start stack for %s: %v", subdomain, err)
				stateMgr.Remove(subdomain)
				return
			}
			log.Printf("Stack running for %s", subdomain)
			stateMgr.SetRunning(subdomain, digest)
		}()
	}

	// Idle callback: stop and clean up the stack
	stateMgr.SetOnIdle(func(subdomain string) {
		log.Printf("Idle timeout for %s, stopping stack", subdomain)
		stateMgr.SetStopping(subdomain)
		if err := composeMgr.StopStack(subdomain); err != nil {
			log.Printf("Failed to stop stack for %s: %v", subdomain, err)
		}
		stateMgr.Remove(subdomain)
		log.Printf("Stack removed for %s", subdomain)
	})

	// Startup recovery: re-adopt existing stacks
	existing, err := ListRunningStacks()
	if err != nil {
		log.Printf("Warning: could not list existing stacks: %v", err)
	} else {
		for _, sub := range existing {
			log.Printf("Re-adopting existing stack: %s", sub)
			stateMgr.SetRunning(sub, "") // unknown digest, will be checked on next request
		}
	}

	// Create handler and start server
	handler := NewHandler(cfg.Domain, stateMgr, registryCheck, startStack)
	handler.SetTarget(cfg.TargetService, cfg.TargetPort)

	log.Printf("Review proxy listening on :80 for *.%s", cfg.Domain)
	if err := http.ListenAndServe(":80", handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

### Step 2: Verify it compiles

Run: `go build -o review-proxy .`
Expected: Builds successfully.

### Step 3: Commit

```bash
git add main.go
git commit -m "feat: wire up all components in main.go with startup recovery"
```
