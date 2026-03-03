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
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

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
