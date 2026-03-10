package gitutil

import (
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
