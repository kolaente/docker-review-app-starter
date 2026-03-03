package main

import (
	"os"
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	template := "services:\n  app:\n    image: registry.example.com/myapp:${SUBDOMAIN}\n"
	tmpfile, err := os.CreateTemp("", "template-*.yml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	if _, err := tmpfile.Write([]byte(template)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	outPath, cleanup, err := RenderTemplate(tmpfile.Name(), "pr-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	data, _ := os.ReadFile(outPath)
	content := string(data)
	if !strings.Contains(content, "registry.example.com/myapp:pr-42") {
		t.Errorf("expected substituted image tag, got:\n%s", content)
	}
	if strings.Contains(content, "${SUBDOMAIN}") {
		t.Errorf("template placeholder was not replaced")
	}
}
