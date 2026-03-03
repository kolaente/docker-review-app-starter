---
# docker-review-app-starter-4tt9
title: Implement Docker registry client
status: completed
type: task
priority: normal
created_at: 2026-03-03T13:59:41Z
updated_at: 2026-03-03T14:11:22Z
parent: docker-review-app-starter-014k
blocked_by:
    - docker-review-app-starter-u6es
---

## Implement Docker registry client

Queries the Docker Registry HTTP API v2 to check if a tag exists and fetch its digest.

### Files
- Create: `registry.go`
- Create: `registry_test.go`

### Step 1: Write failing test for registry image reference parsing

The proxy needs to parse image references like `registry.example.com/myapp:${SUBDOMAIN}` from the compose template to know which registry/repo to query.

Create `registry_test.go`:

```go
package main

import (
	"testing"
)

func TestParseImageRef(t *testing.T) {
	tests := []struct {
		image    string
		registry string
		repo     string
		tag      string
	}{
		{"registry.example.com/myapp:pr-42", "registry.example.com", "myapp", "pr-42"},
		{"ghcr.io/org/app:main", "ghcr.io", "org/app", "main"},
		{"docker.io/library/nginx:latest", "docker.io", "library/nginx", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.image, func(t *testing.T) {
			ref, err := ParseImageRef(tt.image)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ref.Registry != tt.registry {
				t.Errorf("registry: expected %q, got %q", tt.registry, ref.Registry)
			}
			if ref.Repo != tt.repo {
				t.Errorf("repo: expected %q, got %q", tt.repo, ref.Repo)
			}
			if ref.Tag != tt.tag {
				t.Errorf("tag: expected %q, got %q", tt.tag, ref.Tag)
			}
		})
	}
}
```

Run: `go test ./... -run TestParseImageRef -v`
Expected: FAIL.

### Step 2: Implement image reference parsing

Create `registry.go`:

```go
package main

import (
	"fmt"
	"strings"
)

type ImageRef struct {
	Registry string
	Repo     string
	Tag      string
}

func ParseImageRef(image string) (*ImageRef, error) {
	// Split tag
	tag := "latest"
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		// Make sure this colon isn't part of the registry (port)
		rest := image[idx+1:]
		if !strings.Contains(rest, "/") {
			tag = rest
			image = image[:idx]
		}
	}

	// Split registry from repo
	parts := strings.SplitN(image, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("cannot parse image reference %q: no registry prefix", image)
	}

	return &ImageRef{
		Registry: parts[0],
		Repo:     parts[1],
		Tag:      tag,
	}, nil
}
```

Run: `go test ./... -run TestParseImageRef -v`
Expected: PASS.

### Step 3: Implement registry tag check and digest fetch

Add to `registry.go`:

```go
import (
	"encoding/json"
	"net/http"
)

type RegistryClient struct {
	HTTPClient *http.Client
}

// CheckTag queries the registry for a tag. Returns the digest if it exists, empty string if not.
func (rc *RegistryClient) CheckTag(ref *ImageRef) (string, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", ref.Registry, ref.Repo, ref.Tag)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json, application/vnd.oci.image.manifest.v1+json")

	resp, err := rc.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("registry request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	return digest, nil
}
```

Note: Testing registry calls against a real registry is an integration test. Unit tests cover parsing. Integration testing happens in the end-to-end task.

### Step 4: Add template image parsing

Add a function to extract the image reference pattern from the compose template:

```go
func ParseTemplateImageRef(templatePath string) (string, error) {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("reading template: %w", err)
	}

	// Find image lines containing ${SUBDOMAIN}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "image:") && strings.Contains(line, "${SUBDOMAIN}") {
			ref := strings.TrimPrefix(line, "image:")
			ref = strings.TrimSpace(ref)
			// Remove quotes if present
			ref = strings.Trim(ref, "\"'")
			return ref, nil
		}
	}

	return "", fmt.Errorf("no image with ${SUBDOMAIN} placeholder found in template")
}
```

### Step 5: Write test for template image parsing

Add to `registry_test.go`:

```go
func TestParseTemplateImageRef(t *testing.T) {
	content := []byte("services:\n  app:\n    image: registry.example.com/myapp:${SUBDOMAIN}\n    networks:\n      - review-proxy\n")
	tmpfile, _ := os.CreateTemp("", "template-*.yml")
	defer os.Remove(tmpfile.Name())
	tmpfile.Write(content)
	tmpfile.Close()

	ref, err := ParseTemplateImageRef(tmpfile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref != "registry.example.com/myapp:${SUBDOMAIN}" {
		t.Errorf("expected registry.example.com/myapp:${SUBDOMAIN}, got %s", ref)
	}
}
```

Run: `go test ./... -run TestParseTemplateImageRef -v`
Expected: PASS.

### Step 6: Commit

```bash
git add registry.go registry_test.go
git commit -m "feat: add Docker registry client and image reference parsing"
```
