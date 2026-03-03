package main

import (
	"os"
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	template := "services:\n  app:\n    image: registry.example.com/myapp:${SUBDOMAIN}\n"
	tmpfile, _ := os.CreateTemp("", "template-*.yml")
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(template))
	tmpfile.Close()

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
