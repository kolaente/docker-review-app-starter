---
# docker-review-app-starter-2kmq
title: Implement subdomain extraction
status: completed
type: task
priority: normal
created_at: 2026-03-03T13:59:16Z
updated_at: 2026-03-03T14:11:22Z
parent: docker-review-app-starter-014k
blocked_by:
    - docker-review-app-starter-u6es
---

## Implement subdomain extraction

### Files
- Create: `subdomain.go`
- Create: `subdomain_test.go`

### Step 1: Write failing test for subdomain extraction

Create `subdomain_test.go`:

```go
package main

import (
	"testing"
)

func TestExtractSubdomain(t *testing.T) {
	tests := []struct {
		host     string
		domain   string
		expected string
		wantErr  bool
	}{
		{"pr-42.review.example.com", "review.example.com", "pr-42", false},
		{"my-feature.review.example.com", "review.example.com", "my-feature", false},
		{"review.example.com", "review.example.com", "", true},
		{"other.example.com", "review.example.com", "", true},
		{"pr-42.review.example.com:8080", "review.example.com", "pr-42", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result, err := ExtractSubdomain(tt.host, tt.domain)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
```

Run: `go test ./... -run TestExtractSubdomain -v`
Expected: FAIL — `ExtractSubdomain` undefined.

### Step 2: Implement subdomain extraction

Create `subdomain.go`:

```go
package main

import (
	"fmt"
	"strings"
)

func ExtractSubdomain(host, domain string) (string, error) {
	// Strip port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		h := host[:idx]
		// Only strip if what's after : looks like a port (not part of IPv6)
		if !strings.Contains(h, ":") {
			host = h
		}
	}

	suffix := "." + domain
	if !strings.HasSuffix(host, suffix) {
		return "", fmt.Errorf("host %q does not match domain %q", host, domain)
	}

	subdomain := strings.TrimSuffix(host, suffix)
	if subdomain == "" {
		return "", fmt.Errorf("no subdomain in host %q", host)
	}

	return subdomain, nil
}
```

Run: `go test ./... -run TestExtractSubdomain -v`
Expected: PASS.

### Step 3: Commit

```bash
git add subdomain.go subdomain_test.go
git commit -m "feat: add subdomain extraction from request host"
```
