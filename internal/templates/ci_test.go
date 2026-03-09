package templates

import (
	"strings"
	"testing"
)

func TestCIWorkflow(t *testing.T) {
	for _, name := range CITemplateNames() {
		t.Run(name, func(t *testing.T) {
			content, err := CIWorkflow(name)
			if err != nil {
				t.Fatalf("CIWorkflow(%q) error: %v", name, err)
			}
			if len(content) == 0 {
				t.Fatalf("CIWorkflow(%q) returned empty content", name)
			}
			s := string(content)
			if !strings.Contains(s, "name: CI") {
				t.Errorf("CIWorkflow(%q) missing 'name: CI' header", name)
			}
			if !strings.Contains(s, "actions/checkout@v4") {
				t.Errorf("CIWorkflow(%q) missing checkout action", name)
			}
		})
	}
}

func TestCIWorkflowNotFound(t *testing.T) {
	_, err := CIWorkflow("nonexistent")
	if err == nil {
		t.Fatal("CIWorkflow(nonexistent) expected error")
	}
}

func TestCITemplateNames(t *testing.T) {
	names := CITemplateNames()
	if len(names) != 4 {
		t.Fatalf("CITemplateNames() returned %d names, want 4", len(names))
	}
	expected := map[string]bool{"go": true, "rust": true, "node": true, "python": true}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected template name: %q", name)
		}
	}
}
