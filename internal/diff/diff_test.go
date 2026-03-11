package diff

import (
	"testing"

	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/github/mock"
	"github.com/amenophis1er/gh-setup/internal/templates"
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

func TestDiffDetectsMergeStrategyChanges(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(false),
				AllowSquashMerge:    gh.Ptr(true),
				AllowMergeCommit:    gh.Ptr(true),
				AllowRebaseMerge:    gh.Ptr(true),
				AllowAutoMerge:      gh.Ptr(false),
				HasWiki:             gh.Ptr(true),
				HasDiscussions:      gh.Ptr(false),
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
			AllowSquashMerge: gh.Ptr(true),
			AllowMergeCommit: gh.Ptr(false), // want to disable
			AllowRebaseMerge: gh.Ptr(true),
			AllowAutoMerge:   true, // want to enable
			HasWiki:          gh.Ptr(false),
			HasDiscussions:   gh.Ptr(true),
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	expected := map[string]bool{
		"allow_merge_commit": false,
		"allow_auto_merge":   false,
		"has_wiki":           false,
		"has_discussions":    false,
	}
	for _, c := range result.Changes {
		if _, ok := expected[c.Field]; ok && c.Action == "change" {
			expected[c.Field] = true
		}
	}
	for field, found := range expected {
		if !found {
			t.Errorf("expected change for %s, but not found", field)
		}
	}
}

func TestDiffIgnoresUnsetBoolPtrs(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(false),
				AllowSquashMerge:    gh.Ptr(true),
				AllowMergeCommit:    gh.Ptr(true),
				AllowRebaseMerge:    gh.Ptr(true),
				HasIssues:           gh.Ptr(true),
				HasWiki:             gh.Ptr(true),
				HasDiscussions:      gh.Ptr(false),
			}, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return nil, nil
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
	}

	// All *bool fields left nil — should produce no changes for those fields
	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	boolPtrFields := map[string]bool{
		"allow_squash_merge": true,
		"allow_merge_commit": true,
		"allow_rebase_merge": true,
		"has_issues":         true,
		"has_wiki":           true,
		"has_discussions":    true,
	}
	for _, c := range result.Changes {
		if boolPtrFields[c.Field] {
			t.Errorf("unexpected change for unset field %s", c.Field)
		}
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

func TestDiffCIWorkflowMissing(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
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
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil // file doesn't exist
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
			{Name: "my-repo", CI: "go"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	found := false
	for _, c := range result.Changes {
		if c.Field == "ci_workflow" && c.Action == "add" {
			found = true
		}
	}
	if !found {
		t.Error("expected ci_workflow add, but not found")
	}
}

func TestDiffCIWorkflowUpToDate(t *testing.T) {
	goTemplate, _ := templates.CIWorkflow("go")

	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
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
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			if path == ".github/workflows/ci.yml" {
				return string(goTemplate), gh.Ptr("abc123"), nil
			}
			return "", nil, nil
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
			{Name: "my-repo", CI: "go"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	for _, c := range result.Changes {
		if c.Field == "ci_workflow" {
			t.Errorf("expected no ci_workflow change, got action=%s", c.Action)
		}
	}
}

func TestDiffCIWorkflowDrift(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
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
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			if path == ".github/workflows/ci.yml" {
				return "name: Old CI\non: push\n", gh.Ptr("abc123"), nil
			}
			return "", nil, nil
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
			{Name: "my-repo", CI: "go"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	found := false
	for _, c := range result.Changes {
		if c.Field == "ci_workflow" && c.Action == "change" {
			found = true
		}
	}
	if !found {
		t.Error("expected ci_workflow change, but not found")
	}
}

func TestDiffDependabotMissing(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
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
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil
		},
		GetVulnerabilityAlertsFn: func(owner, repo string) (bool, error) {
			return true, nil // already enabled
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Security: config.Security{Dependabot: true},
		Repos: []config.Repo{
			{Name: "my-repo", CI: "go"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	found := false
	for _, c := range result.Changes {
		if c.Field == "dependabot.yml" && c.Action == "add" {
			found = true
		}
	}
	if !found {
		t.Error("expected dependabot.yml add, but not found")
	}
}

func TestDiffCodeownersMissing(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
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
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Governance: config.Governance{Codeowners: "* @myorg/core"},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	found := false
	for _, c := range result.Changes {
		if c.Field == "CODEOWNERS" && c.Action == "add" {
			found = true
			if c.New != "* @myorg/core" {
				t.Errorf("expected CODEOWNERS content '* @myorg/core', got %q", c.New)
			}
		}
	}
	if !found {
		t.Error("expected CODEOWNERS add, but not found")
	}
}

func TestDiffCodeownersDrift(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
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
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			if path == ".github/CODEOWNERS" {
				return "* @old-team\n", gh.Ptr("sha1"), nil
			}
			return "", nil, nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Governance: config.Governance{Codeowners: "* @myorg/core"},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	found := false
	for _, c := range result.Changes {
		if c.Field == "CODEOWNERS" && c.Action == "change" {
			found = true
			if c.Old != "* @old-team" || c.New != "* @myorg/core" {
				t.Errorf("expected CODEOWNERS change from '* @old-team' to '* @myorg/core', got %q→%q", c.Old, c.New)
			}
		}
	}
	if !found {
		t.Error("expected CODEOWNERS change, but not found")
	}
}

func TestDiffSecurityFlagsDependabotDisabled(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
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
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil
		},
		GetVulnerabilityAlertsFn: func(owner, repo string) (bool, error) {
			return false, nil // disabled
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Security: config.Security{
			Dependabot:     true,
			SecretScanning: true,
		},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	fields := map[string]bool{
		"dependabot_alerts": false,
		"secret_scanning":   false,
	}
	for _, c := range result.Changes {
		if _, ok := fields[c.Field]; ok && c.Action == "change" {
			fields[c.Field] = true
		}
	}
	for field, found := range fields {
		if !found {
			t.Errorf("expected change for %s, but not found", field)
		}
	}
}

func TestDiffSecurityFlagsAlreadyEnabled(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(false),
				SecurityAndAnalysis: &gh.SecurityAndAnalysis{
					SecretScanning:  &gh.SecretScanning{Status: gh.Ptr("enabled")},
					AdvancedSecurity: &gh.AdvancedSecurity{Status: gh.Ptr("enabled")},
				},
			}, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return nil, nil
		},
		GetBranchProtectionFn: func(owner, repo, branch string) (*gh.Protection, error) {
			return nil, nil
		},
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil
		},
		GetVulnerabilityAlertsFn: func(owner, repo string) (bool, error) {
			return true, nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Security: config.Security{
			Dependabot:     true,
			SecretScanning: true,
			CodeScanning:   true,
		},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	var result DiffResult
	diffRepo(m, cfg, "testuser", cfg.Repos[0], &result)

	securityFields := map[string]bool{
		"dependabot_alerts": true,
		"secret_scanning":   true,
		"code_scanning":     true,
	}
	for _, c := range result.Changes {
		if securityFields[c.Field] {
			t.Errorf("unexpected change for %s when already enabled", c.Field)
		}
	}
}
