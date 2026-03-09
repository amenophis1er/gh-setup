package templates

import (
	"strings"
	"testing"
)

func TestDependabotConfig(t *testing.T) {
	t.Run("empty ecosystem", func(t *testing.T) {
		content := DependabotConfig("")
		if !strings.Contains(content, "github-actions") {
			t.Error("should always include github-actions")
		}
		if strings.Count(content, "package-ecosystem") != 1 {
			t.Error("empty ecosystem should only have github-actions")
		}
	})

	t.Run("with ecosystem", func(t *testing.T) {
		content := DependabotConfig("gomod")
		if !strings.Contains(content, "github-actions") {
			t.Error("should include github-actions")
		}
		if !strings.Contains(content, "gomod") {
			t.Error("should include gomod ecosystem")
		}
		if strings.Count(content, "package-ecosystem") != 2 {
			t.Error("should have two package-ecosystem entries")
		}
	})
}

func TestCIToEcosystem(t *testing.T) {
	tests := []struct {
		ci       string
		expected string
	}{
		{"go", "gomod"},
		{"rust", "cargo"},
		{"node", "npm"},
		{"python", "pip"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ci, func(t *testing.T) {
			got := CIToEcosystem(tt.ci)
			if got != tt.expected {
				t.Errorf("CIToEcosystem(%q) = %q, want %q", tt.ci, got, tt.expected)
			}
		})
	}
}
