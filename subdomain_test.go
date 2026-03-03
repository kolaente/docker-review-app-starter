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
