package diff

import (
	"testing"

	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/github/mock"
	gh "github.com/google/go-github/v68/github"
)

func TestDiffRepoUpToDate(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Description:         gh.Ptr("A test repo"),
				Homepage:            gh.Ptr(""),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(true),
			}, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return nil, nil
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:          "public",
			DefaultBranch:       "main",
			DeleteBranchOnMerge: true,
			BranchProtection:    config.BranchProtection{Preset: "none"},
		},
		Repos: []config.Repo{
			{Name: "my-repo", Description: "A test repo"},
		},
	}

	err := RunWith(m, cfg, "json", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiffDetectsVisibilityChange(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Private:             gh.Ptr(true), // currently private
				DeleteBranchOnMerge: gh.Ptr(false),
			}, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return nil, nil
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public", // desired public
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	// Capture the result by running with JSON output
	var result DiffResult
	var repo = cfg.Repos[0]
	diffRepo(m, cfg, "testuser", repo, &result)

	found := false
	for _, c := range result.Changes {
		if c.Field == "visibility" && c.Action == "change" {
			found = true
			if c.Old != "private" || c.New != "public" {
				t.Errorf("expected private→public, got %s→%s", c.Old, c.New)
			}
		}
	}
	if !found {
		t.Error("expected visibility change, but not found in diff result")
	}
}

func TestDiffDetectsNewRepo(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return nil, nil // repo doesn't exist
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Repos: []config.Repo{
			{Name: "new-repo"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	if len(result.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(result.Changes))
	}
	if result.Changes[0].Action != "add" {
		t.Errorf("expected action 'add', got %s", result.Changes[0].Action)
	}
}

func TestDiffDetectsMissingLabels(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(false),
			}, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return []*gh.Label{}, nil // no labels exist
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Labels: config.Labels{
			Items: []config.Label{
				{Name: "bug", Color: "d73a4a", Description: "Bug report"},
			},
		},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	found := false
	for _, c := range result.Changes {
		if c.Action == "add" && c.Field == "labels" {
			found = true
		}
	}
	if !found {
		t.Error("expected label add change, but not found")
	}
}

func TestDiffTeamMemberChanges(t *testing.T) {
	m := &mock.Client{
		GetTeamFn: func(org, slug string) (*gh.Team, error) {
			return &gh.Team{Slug: gh.Ptr("devs")}, nil
		},
		ListTeamMembersFn: func(org, slug string) ([]*gh.User, error) {
			return []*gh.User{
				{Login: gh.Ptr("alice")},
				{Login: gh.Ptr("charlie")}, // should be removed
			}, nil
		},
	}

	team := config.Team{
		Name:    "devs",
		Members: []string{"alice", "bob"}, // bob should be added
	}

	var result DiffResult
	diffTeam(m, "myorg", team, &result)

	addFound := false
	removeFound := false
	for _, c := range result.Changes {
		if c.Action == "add" && c.New == "bob" {
			addFound = true
		}
		if c.Action == "remove" && c.Old == "charlie" {
			removeFound = true
		}
	}
	if !addFound {
		t.Error("expected add for 'bob'")
	}
	if !removeFound {
		t.Error("expected remove for 'charlie'")
	}
}

func TestDiffDescriptionChange(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Description:         gh.Ptr("Old description"),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(false),
			}, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return nil, nil
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Repos: []config.Repo{
			{Name: "my-repo", Description: "New description"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	found := false
	for _, c := range result.Changes {
		if c.Field == "description" && c.Action == "change" {
			found = true
		}
	}
	if !found {
		t.Error("expected description change, but not found")
	}
}
