package mock

import (
	"github.com/amenophis1er/gh-setup/internal/config"
	gh "github.com/google/go-github/v68/github"
)

// Client is a mock implementation of github.GitHubClient for testing.
type Client struct {
	// Organization
	IsOrganizationFn func(name string) (bool, error)

	// Repositories
	ListReposFn          func(owner string, isOrg bool) ([]*gh.Repository, error)
	GetRepoFn            func(owner, name string) (*gh.Repository, error)
	CreateRepoFn         func(owner string, isOrg bool, repo *gh.Repository) (*gh.Repository, error)
	UpdateRepoFn         func(owner, name string, repo *gh.Repository) (*gh.Repository, error)
	SetTopicsFn          func(owner, name string, topics []string) error
	GetFileContentFn     func(owner, repo, path string) (string, *string, error)
	CreateOrUpdateFileFn func(owner, repo, path, message string, content []byte, sha *string) error

	// Labels
	ListLabelsFn  func(owner, repo string) ([]*gh.Label, error)
	CreateLabelFn func(owner, repo string, label config.Label) error
	UpdateLabelFn func(owner, repo, currentName string, label config.Label) error
	DeleteLabelFn func(owner, repo, name string) error

	// Branch protection
	GetBranchProtectionFn   func(owner, repo, branch string) (*gh.Protection, error)
	ApplyBranchProtectionFn func(owner, repo, branch string, bp config.BranchProtection) error

	// Teams
	GetTeamFn         func(org, slug string) (*gh.Team, error)
	CreateTeamFn      func(org, name, description, permission string) (*gh.Team, error)
	UpdateTeamFn      func(org, slug, name, description, permission string) (*gh.Team, error)
	ListTeamMembersFn func(org, slug string) ([]*gh.User, error)
	ListOrgTeamsFn    func(org string) ([]*gh.Team, error)
	AddTeamMemberFn   func(org, slug, username string) error
	RemoveTeamMemberFn func(org, slug, username string) error

	// Security
	EnableVulnerabilityAlertsFn func(owner, repo string) error
	GetVulnerabilityAlertsFn    func(owner, repo string) (bool, error)
	UpdateSecurityAndAnalysisFn func(owner, repoName string, secretScanning, codeScanningEnabled bool) error

	// Secrets
	SetOrgSecretFn  func(org, name, value string) error
	SetRepoSecretFn func(owner, repo, name, value string) error
}

func (m *Client) IsOrganization(name string) (bool, error) {
	if m.IsOrganizationFn != nil {
		return m.IsOrganizationFn(name)
	}
	return false, nil
}

func (m *Client) ListRepos(owner string, isOrg bool) ([]*gh.Repository, error) {
	if m.ListReposFn != nil {
		return m.ListReposFn(owner, isOrg)
	}
	return nil, nil
}

func (m *Client) GetRepo(owner, name string) (*gh.Repository, error) {
	if m.GetRepoFn != nil {
		return m.GetRepoFn(owner, name)
	}
	return nil, nil
}

func (m *Client) CreateRepo(owner string, isOrg bool, repo *gh.Repository) (*gh.Repository, error) {
	if m.CreateRepoFn != nil {
		return m.CreateRepoFn(owner, isOrg, repo)
	}
	return repo, nil
}

func (m *Client) UpdateRepo(owner, name string, repo *gh.Repository) (*gh.Repository, error) {
	if m.UpdateRepoFn != nil {
		return m.UpdateRepoFn(owner, name, repo)
	}
	return repo, nil
}

func (m *Client) SetTopics(owner, name string, topics []string) error {
	if m.SetTopicsFn != nil {
		return m.SetTopicsFn(owner, name, topics)
	}
	return nil
}

func (m *Client) GetFileContent(owner, repo, path string) (string, *string, error) {
	if m.GetFileContentFn != nil {
		return m.GetFileContentFn(owner, repo, path)
	}
	return "", nil, nil
}

func (m *Client) CreateOrUpdateFile(owner, repo, path, message string, content []byte, sha *string) error {
	if m.CreateOrUpdateFileFn != nil {
		return m.CreateOrUpdateFileFn(owner, repo, path, message, content, sha)
	}
	return nil
}

func (m *Client) ListLabels(owner, repo string) ([]*gh.Label, error) {
	if m.ListLabelsFn != nil {
		return m.ListLabelsFn(owner, repo)
	}
	return nil, nil
}

func (m *Client) CreateLabel(owner, repo string, label config.Label) error {
	if m.CreateLabelFn != nil {
		return m.CreateLabelFn(owner, repo, label)
	}
	return nil
}

func (m *Client) UpdateLabel(owner, repo, currentName string, label config.Label) error {
	if m.UpdateLabelFn != nil {
		return m.UpdateLabelFn(owner, repo, currentName, label)
	}
	return nil
}

func (m *Client) DeleteLabel(owner, repo, name string) error {
	if m.DeleteLabelFn != nil {
		return m.DeleteLabelFn(owner, repo, name)
	}
	return nil
}

func (m *Client) GetBranchProtection(owner, repo, branch string) (*gh.Protection, error) {
	if m.GetBranchProtectionFn != nil {
		return m.GetBranchProtectionFn(owner, repo, branch)
	}
	return nil, nil
}

func (m *Client) ApplyBranchProtection(owner, repo, branch string, bp config.BranchProtection) error {
	if m.ApplyBranchProtectionFn != nil {
		return m.ApplyBranchProtectionFn(owner, repo, branch, bp)
	}
	return nil
}

func (m *Client) GetTeam(org, slug string) (*gh.Team, error) {
	if m.GetTeamFn != nil {
		return m.GetTeamFn(org, slug)
	}
	return nil, nil
}

func (m *Client) CreateTeam(org, name, description, permission string) (*gh.Team, error) {
	if m.CreateTeamFn != nil {
		return m.CreateTeamFn(org, name, description, permission)
	}
	return nil, nil
}

func (m *Client) UpdateTeam(org, slug, name, description, permission string) (*gh.Team, error) {
	if m.UpdateTeamFn != nil {
		return m.UpdateTeamFn(org, slug, name, description, permission)
	}
	return nil, nil
}

func (m *Client) ListTeamMembers(org, slug string) ([]*gh.User, error) {
	if m.ListTeamMembersFn != nil {
		return m.ListTeamMembersFn(org, slug)
	}
	return nil, nil
}

func (m *Client) ListOrgTeams(org string) ([]*gh.Team, error) {
	if m.ListOrgTeamsFn != nil {
		return m.ListOrgTeamsFn(org)
	}
	return nil, nil
}

func (m *Client) AddTeamMember(org, slug, username string) error {
	if m.AddTeamMemberFn != nil {
		return m.AddTeamMemberFn(org, slug, username)
	}
	return nil
}

func (m *Client) RemoveTeamMember(org, slug, username string) error {
	if m.RemoveTeamMemberFn != nil {
		return m.RemoveTeamMemberFn(org, slug, username)
	}
	return nil
}

func (m *Client) EnableVulnerabilityAlerts(owner, repo string) error {
	if m.EnableVulnerabilityAlertsFn != nil {
		return m.EnableVulnerabilityAlertsFn(owner, repo)
	}
	return nil
}

func (m *Client) GetVulnerabilityAlerts(owner, repo string) (bool, error) {
	if m.GetVulnerabilityAlertsFn != nil {
		return m.GetVulnerabilityAlertsFn(owner, repo)
	}
	return false, nil
}

func (m *Client) UpdateSecurityAndAnalysis(owner, repoName string, secretScanning, codeScanningEnabled bool) error {
	if m.UpdateSecurityAndAnalysisFn != nil {
		return m.UpdateSecurityAndAnalysisFn(owner, repoName, secretScanning, codeScanningEnabled)
	}
	return nil
}

func (m *Client) SetOrgSecret(org, name, value string) error {
	if m.SetOrgSecretFn != nil {
		return m.SetOrgSecretFn(org, name, value)
	}
	return nil
}

func (m *Client) SetRepoSecret(owner, repo, name, value string) error {
	if m.SetRepoSecretFn != nil {
		return m.SetRepoSecretFn(owner, repo, name, value)
	}
	return nil
}
