package gitutil

import (
	"os/exec"
	"regexp"
	"strings"
)

// Remote holds the owner and repo name parsed from a git remote URL.
type Remote struct {
	Owner string
	Repo  string
}

var remoteRe = regexp.MustCompile(`(?:github\.com[:/])([^/]+)/([^/.]+?)(?:\.git)?$`)

// DetectRemote parses the "origin" remote URL of the current git repository
// and extracts the GitHub owner and repo name. Returns zero Remote and nil
// error if not in a git repo or origin is not a GitHub URL.
func DetectRemote() (Remote, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return Remote{}, nil // not a git repo or no origin — not an error
	}

	url := strings.TrimSpace(string(out))
	m := remoteRe.FindStringSubmatch(url)
	if m == nil {
		return Remote{}, nil // not a GitHub remote
	}

	return Remote{Owner: m[1], Repo: m[2]}, nil
}
