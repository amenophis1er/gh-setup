package importer

import (
	"testing"

	"github.com/amenophis1er/gh-setup/internal/github/mock"
	gh "github.com/google/go-github/v68/github"
)

func TestImportSingleRepo(t *testing.T) {
	m := &mock.Client{
		IsOrganizationFn: func(name string) (bool, error) { return false, nil },
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:        gh.Ptr("my-repo"),
				Description: gh.Ptr("A test repo"),
				Homepage:    gh.Ptr("https://example.com"),
				Private:     gh.Ptr(false),
				Topics:      []string{"go", "cli"},
				DefaultBranch:       gh.Ptr("main"),
				DeleteBranchOnMerge: gh.Ptr(true),
			}, nil
		},
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
		GetVulnerabilityAlertsFn: func(owner, repo string) (bool, error) {
			return true, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return []*gh.Label{
				{Name: gh.Ptr("bug"), Color: gh.Ptr("d73a4a"), Description: gh.Ptr("Something isn't working")},
			}, nil
		},
	}

	cfg, err := RunWith(m, Options{Account: "testuser", RepoName: "my-repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Account.Type != "individual" {
		t.Errorf("expected individual, got %s", cfg.Account.Type)
	}
	if cfg.Account.Name != "testuser" {
		t.Errorf("expected testuser, got %s", cfg.Account.Name)
	}
	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Name != "my-repo" {
		t.Errorf("expected my-repo, got %s", cfg.Repos[0].Name)
	}
	if cfg.Repos[0].Visibility != "public" {
		t.Errorf("expected public, got %s", cfg.Repos[0].Visibility)
	}
	if cfg.Repos[0].Description != "A test repo" {
		t.Errorf("expected 'A test repo', got %s", cfg.Repos[0].Description)
	}
	if len(cfg.Labels.Items) != 1 {
		t.Fatalf("expected 1 label, got %d", len(cfg.Labels.Items))
	}
	if cfg.Labels.Items[0].Name != "bug" {
		t.Errorf("expected label 'bug', got %s", cfg.Labels.Items[0].Name)
	}
	if cfg.Defaults.DeleteBranchOnMerge != true {
		t.Error("expected delete_branch_on_merge to be true")
	}
	if cfg.Security.Dependabot != true {
		t.Error("expected dependabot to be true")
	}
}

func TestImportOrgWithTeams(t *testing.T) {
	m := &mock.Client{
		IsOrganizationFn: func(name string) (bool, error) { return true, nil },
		ListReposFn: func(owner string, isOrg bool) ([]*gh.Repository, error) {
			return []*gh.Repository{
				{Name: gh.Ptr("repo1"), Archived: gh.Ptr(false), Fork: gh.Ptr(false)},
			}, nil
		},
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr(name),
				Private:             gh.Ptr(true),
				DefaultBranch:       gh.Ptr("main"),
				DeleteBranchOnMerge: gh.Ptr(false),
			}, nil
		},
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
		GetVulnerabilityAlertsFn: func(owner, repo string) (bool, error) {
			return false, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return nil, nil
		},
		ListOrgTeamsFn: func(org string) ([]*gh.Team, error) {
			return []*gh.Team{
				{Slug: gh.Ptr("backend"), Name: gh.Ptr("backend"), Description: gh.Ptr("Backend team"), Permission: gh.Ptr("push")},
			}, nil
		},
		ListTeamMembersFn: func(org, slug string) ([]*gh.User, error) {
			return []*gh.User{
				{Login: gh.Ptr("alice")},
				{Login: gh.Ptr("bob")},
			}, nil
		},
	}

	cfg, err := RunWith(m, Options{Account: "myorg", Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Account.Type != "organization" {
		t.Errorf("expected organization, got %s", cfg.Account.Type)
	}
	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Visibility != "private" {
		t.Errorf("expected private, got %s", cfg.Repos[0].Visibility)
	}
	if len(cfg.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(cfg.Teams))
	}
	if cfg.Teams[0].Name != "backend" {
		t.Errorf("expected team 'backend', got %s", cfg.Teams[0].Name)
	}
	if cfg.Teams[0].Permission != "write" {
		t.Errorf("expected permission 'write', got %s", cfg.Teams[0].Permission)
	}
	if len(cfg.Teams[0].Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(cfg.Teams[0].Members))
	}
}

func TestImportSkipsArchivedAndForked(t *testing.T) {
	m := &mock.Client{
		IsOrganizationFn: func(name string) (bool, error) { return false, nil },
		ListReposFn: func(owner string, isOrg bool) ([]*gh.Repository, error) {
			return []*gh.Repository{
				{Name: gh.Ptr("active"), Archived: gh.Ptr(false), Fork: gh.Ptr(false)},
				{Name: gh.Ptr("archived"), Archived: gh.Ptr(true), Fork: gh.Ptr(false)},
				{Name: gh.Ptr("forked"), Archived: gh.Ptr(false), Fork: gh.Ptr(true)},
			}, nil
		},
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr(name),
				Private:             gh.Ptr(false),
				DefaultBranch:       gh.Ptr("main"),
				DeleteBranchOnMerge: gh.Ptr(false),
			}, nil
		},
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
		GetVulnerabilityAlertsFn: func(owner, repo string) (bool, error) {
			return false, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return nil, nil
		},
	}

	cfg, err := RunWith(m, Options{Account: "testuser", Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Repos) != 1 {
		t.Fatalf("expected 1 repo (only active), got %d", len(cfg.Repos))
	}
	if cfg.Repos[0].Name != "active" {
		t.Errorf("expected 'active', got %s", cfg.Repos[0].Name)
	}
}

func TestImportDetectsCITemplate(t *testing.T) {
	m := &mock.Client{
		IsOrganizationFn: func(name string) (bool, error) { return false, nil },
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr(name),
				Private:             gh.Ptr(false),
				DefaultBranch:       gh.Ptr("main"),
				DeleteBranchOnMerge: gh.Ptr(false),
			}, nil
		},
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			if path == ".github/workflows/ci.yml" {
				return "name: CI\njobs:\n  test:\n    steps:\n      - run: go test ./...", nil, nil
			}
			return "", nil, nil
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
		GetVulnerabilityAlertsFn: func(owner, repo string) (bool, error) {
			return false, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return nil, nil
		},
	}

	cfg, err := RunWith(m, Options{Account: "testuser", RepoName: "my-go-repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Repos[0].CI != "go" {
		t.Errorf("expected CI template 'go', got %q", cfg.Repos[0].CI)
	}
}
