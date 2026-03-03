---
# docker-review-app-starter-tfp9
title: Implement status pages (preparing and error)
status: completed
type: task
priority: normal
created_at: 2026-03-03T14:00:51Z
updated_at: 2026-03-03T14:11:22Z
parent: docker-review-app-starter-014k
blocked_by:
    - docker-review-app-starter-u6es
---

## Implement status pages

HTML pages served during stack startup and when an image is not found.

### Files
- Create: `pages.go`
- Create: `pages_test.go`

### Step 1: Write failing test for page rendering

Create `pages_test.go`:

```go
package main

import (
	"strings"
	"testing"
)

func TestPreparingPage(t *testing.T) {
	html := RenderPreparingPage("pr-42")
	if !strings.Contains(html, "pr-42") {
		t.Error("preparing page should contain subdomain")
	}
	if !strings.Contains(html, "refresh") {
		t.Error("preparing page should contain auto-refresh meta tag")
	}
}

func TestNotFoundPage(t *testing.T) {
	html := RenderNotFoundPage("pr-99")
	if !strings.Contains(html, "pr-99") {
		t.Error("not found page should contain subdomain")
	}
	if !strings.Contains(html, "not found") && !strings.Contains(html, "Not Found") && !strings.Contains(html, "does not exist") {
		t.Error("not found page should indicate the image was not found")
	}
}
```

Run: `go test ./... -run TestPreparingPage -v && go test ./... -run TestNotFoundPage -v`
Expected: FAIL.

### Step 2: Implement status pages

Create `pages.go`:

```go
package main

import "fmt"

func RenderPreparingPage(subdomain string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta http-equiv="refresh" content="3">
    <title>Preparing %s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: #f5f5f5;
            color: #333;
        }
        .container {
            text-align: center;
            padding: 2rem;
        }
        .spinner {
            width: 40px;
            height: 40px;
            border: 4px solid #e0e0e0;
            border-top-color: #333;
            border-radius: 50%%;
            animation: spin 0.8s linear infinite;
            margin: 0 auto 1.5rem;
        }
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        h1 { font-size: 1.5rem; font-weight: 500; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="spinner"></div>
        <h1>Preparing environment</h1>
        <p>Setting up <strong>%s</strong>. This page will refresh automatically.</p>
    </div>
</body>
</html>`, subdomain, subdomain)
}

func RenderNotFoundPage(subdomain string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Not Found - %s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: #f5f5f5;
            color: #333;
        }
        .container {
            text-align: center;
            padding: 2rem;
        }
        h1 { font-size: 1.5rem; font-weight: 500; }
        p { color: #666; }
        code {
            background: #e8e8e8;
            padding: 0.2rem 0.5rem;
            border-radius: 3px;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Image does not exist</h1>
        <p>No Docker image was found for <code>%s</code>.</p>
        <p>Make sure your CI pipeline has built and pushed the image for this branch.</p>
    </div>
</body>
</html>`, subdomain, subdomain)
}
```

Run: `go test ./... -run "TestPreparingPage|TestNotFoundPage" -v`
Expected: PASS.

### Step 3: Commit

```bash
git add pages.go pages_test.go
git commit -m "feat: add preparing and not-found HTML status pages"
```
