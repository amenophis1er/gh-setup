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

// IsInsideGitRepo returns true if the current directory is inside a git working tree.
func IsInsideGitRepo() bool {
	err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run()
	return err == nil
}

// HasRemote returns true if the named remote exists.
func HasRemote(name string) bool {
	err := exec.Command("git", "remote", "get-url", name).Run()
	return err == nil
}

// AddRemote adds a new git remote with the given name and URL.
func AddRemote(name, url string) error {
	return exec.Command("git", "remote", "add", name, url).Run()
}
