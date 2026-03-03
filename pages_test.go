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
