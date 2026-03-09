package templates

import (
	"strings"
	"testing"
)

func TestContributing(t *testing.T) {
	content := Contributing("my-project")
	if !strings.Contains(content, "my-project") {
		t.Error("Contributing() should contain repo name")
	}
	if !strings.Contains(content, "How to Contribute") {
		t.Error("Contributing() should contain contribution instructions")
	}
}

func TestCodeOfConduct(t *testing.T) {
	content := CodeOfConduct()
	if !strings.Contains(content, "Contributor Covenant") {
		t.Error("CodeOfConduct() should reference Contributor Covenant")
	}
	if !strings.Contains(content, "Our Pledge") {
		t.Error("CodeOfConduct() should contain Our Pledge section")
	}
}

func TestSecurityPolicy(t *testing.T) {
	content := SecurityPolicy("my-project")
	if !strings.Contains(content, "my-project") {
		t.Error("SecurityPolicy() should contain repo name")
	}
	if !strings.Contains(content, "Reporting a Vulnerability") {
		t.Error("SecurityPolicy() should contain reporting instructions")
	}
}
