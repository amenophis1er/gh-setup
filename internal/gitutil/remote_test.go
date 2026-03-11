package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRemoteRegex(t *testing.T) {
	tests := []struct {
		url           string
		owner, repo   string
		shouldMatch   bool
	}{
		{"git@github.com:myorg/my-repo.git", "myorg", "my-repo", true},
		{"git@github.com:myorg/my-repo", "myorg", "my-repo", true},
		{"https://github.com/myorg/my-repo.git", "myorg", "my-repo", true},
		{"https://github.com/myorg/my-repo", "myorg", "my-repo", true},
		{"ssh://git@github.com/owner/repo.git", "owner", "repo", true},
		{"https://github.com/user/repo-name.git", "user", "repo-name", true},
		{"https://github.com/user/repo_name.git", "user", "repo_name", true},
		{"https://gitlab.com/user/repo.git", "", "", false},
		{"not-a-url", "", "", false},
	}

	for _, tt := range tests {
		m := remoteRe.FindStringSubmatch(tt.url)
		if tt.shouldMatch {
			if m == nil {
				t.Errorf("expected %q to match", tt.url)
				continue
			}
			if m[1] != tt.owner || m[2] != tt.repo {
				t.Errorf("%q: expected %s/%s, got %s/%s", tt.url, tt.owner, tt.repo, m[1], m[2])
			}
		} else {
			if m != nil {
				t.Errorf("expected %q not to match, got %v", tt.url, m)
			}
		}
	}
}

func TestIsInsideGitRepo(t *testing.T) {
	// We're running inside the gh-setup repo, so this should be true.
	if !IsInsideGitRepo() {
		t.Error("expected IsInsideGitRepo() to return true inside a git repo")
	}
}

func TestIsInsideGitRepoOutside(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()
	_ = os.Chdir(dir)

	if IsInsideGitRepo() {
		t.Error("expected IsInsideGitRepo() to return false outside a git repo")
	}
}

func TestHasRemote(t *testing.T) {
	// The gh-setup repo has origin.
	if !HasRemote("origin") {
		t.Error("expected HasRemote(\"origin\") to return true")
	}
	if HasRemote("nonexistent-remote") {
		t.Error("expected HasRemote(\"nonexistent-remote\") to return false")
	}
}

func TestAddRemote(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer func() { _ = os.Chdir(orig) }()

	// Init a bare git repo in the temp dir.
	_ = os.Chdir(dir)
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	if HasRemote("origin") {
		t.Fatal("expected no origin remote in fresh repo")
	}

	url := "git@github.com:testuser/testrepo.git"
	if err := AddRemote("origin", url); err != nil {
		t.Fatalf("AddRemote: %v", err)
	}

	if !HasRemote("origin") {
		t.Error("expected origin remote after AddRemote")
	}

	// Verify the URL matches.
	out, _ := exec.Command("git", "remote", "get-url", "origin").Output()
	got := filepath.Clean(string(out[:len(out)-1])) // trim newline
	if got != url {
		t.Errorf("expected remote URL %q, got %q", url, got)
	}
}
