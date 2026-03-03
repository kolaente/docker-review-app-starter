---
# docker-review-app-starter-u6es
title: Set up Go project and config loading
status: completed
type: task
priority: normal
created_at: 2026-03-03T13:59:05Z
updated_at: 2026-03-03T14:09:02Z
parent: docker-review-app-starter-014k
---

## Set up Go project and config loading

### Files
- Create: `go.mod`
- Create: `main.go`
- Create: `config.go`
- Create: `config_test.go`
- Create: `config.example.yaml`

### Step 1: Initialize Go module

Run: `go mod init github.com/kolaente/docker-review-app-starter`

### Step 2: Write failing test for config loading

Create `config_test.go`:

```go
package main

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	content := []byte("domain: review.example.com\ncompose_template: docker-compose.template.yml\ntarget_service: app\ntarget_port: 8080\nidle_timeout: 5m\n")
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Write(content)
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Domain != "review.example.com" {
		t.Errorf("expected domain review.example.com, got %s", cfg.Domain)
	}
	if cfg.ComposeTemplate != "docker-compose.template.yml" {
		t.Errorf("expected compose_template docker-compose.template.yml, got %s", cfg.ComposeTemplate)
	}
	if cfg.TargetService != "app" {
		t.Errorf("expected target_service app, got %s", cfg.TargetService)
	}
	if cfg.TargetPort != 8080 {
		t.Errorf("expected target_port 8080, got %d", cfg.TargetPort)
	}
	if cfg.IdleTimeout != 5*time.Minute {
		t.Errorf("expected idle_timeout 5m, got %v", cfg.IdleTimeout)
	}
}
```

Run: `go test ./... -run TestLoadConfig -v`
Expected: FAIL — `LoadConfig` undefined.

### Step 3: Implement config loading

Create `config.go`:

```go
package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Domain          string        `yaml:"domain"`
	ComposeTemplate string        `yaml:"compose_template"`
	TargetService   string        `yaml:"target_service"`
	TargetPort      int           `yaml:"target_port"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}
```

Run: `go mod tidy && go test ./... -run TestLoadConfig -v`
Expected: PASS.

### Step 4: Create main.go skeleton

Create `main.go`:

```go
package main

import (
	"flag"
	"log"
)

func main() {
	configPath := flag.String("config", "/etc/review-proxy/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log.Printf("Review proxy starting for domain %s", cfg.Domain)
}
```

### Step 5: Create example config

Create `config.example.yaml`:

```yaml
domain: review.example.com
compose_template: docker-compose.template.yml
target_service: app
target_port: 8080
idle_timeout: 5m
```

### Step 6: Commit

```bash
git add go.mod go.sum main.go config.go config_test.go config.example.yaml
git commit -m "feat: add Go project skeleton with config loading"
```
