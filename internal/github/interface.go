package github

import (
	"github.com/amenophis1er/gh-setup/internal/config"
	gh "github.com/google/go-github/v68/github"
)

// GitHubClient defines the interface for all GitHub operations used by gh-setup.
type GitHubClient interface {
	// Organization
	IsOrganization(name string) (bool, error)

	// Repositories
	ListRepos(owner string, isOrg bool) ([]*gh.Repository, error)
	GetRepo(owner, name string) (*gh.Repository, error)
	CreateRepo(owner string, isOrg bool, repo *gh.Repository) (*gh.Repository, error)
	UpdateRepo(owner, name string, repo *gh.Repository) (*gh.Repository, error)
	SetTopics(owner, name string, topics []string) error
	GetFileContent(owner, repo, path string) (string, *string, error)
	CreateOrUpdateFile(owner, repo, path, message string, content []byte, sha *string) error

	// Labels
	ListLabels(owner, repo string) ([]*gh.Label, error)
	CreateLabel(owner, repo string, label config.Label) error
	UpdateLabel(owner, repo, currentName string, label config.Label) error
	DeleteLabel(owner, repo, name string) error

	// Branch protection
	GetBranchProtection(owner, repo, branch string) (*gh.Protection, error)
	ApplyBranchProtection(owner, repo, branch string, bp config.BranchProtection) error

	// Teams
	GetTeam(org, slug string) (*gh.Team, error)
	CreateTeam(org, name, description, permission string) (*gh.Team, error)
	UpdateTeam(org, slug, name, description, permission string) (*gh.Team, error)
	ListTeamMembers(org, slug string) ([]*gh.User, error)
	ListOrgTeams(org string) ([]*gh.Team, error)
	AddTeamMember(org, slug, username string) error
	RemoveTeamMember(org, slug, username string) error

	// Security
	EnableVulnerabilityAlerts(owner, repo string) error
	GetVulnerabilityAlerts(owner, repo string) (bool, error)
	UpdateSecurityAndAnalysis(owner, repoName string, secretScanning, codeScanningEnabled bool) error

	// Secrets
	SetOrgSecret(org, name, value string) error
	SetRepoSecret(owner, repo, name, value string) error
}
