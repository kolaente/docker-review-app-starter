package main

import (
	"os"
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
