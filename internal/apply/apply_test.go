package apply

import (
	"sync"
	"testing"

	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/github/mock"
	gh "github.com/google/go-github/v68/github"
)

func TestApplyCreatesNewRepo(t *testing.T) {
	var created bool
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return nil, nil // doesn't exist
		},
		CreateRepoFn: func(owner string, isOrg bool, repo *gh.Repository) (*gh.Repository, error) {
			created = true
			if repo.GetName() != "new-repo" {
				t.Errorf("expected repo name 'new-repo', got %s", repo.GetName())
			}
			return repo, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
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
		Repos: []config.Repo{
			{Name: "new-repo", Description: "A new repo"},
		},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected repo to be created")
	}
}

func TestApplyUpdatesExistingRepo(t *testing.T) {
	var updated bool
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Description:         gh.Ptr("Old"),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(false),
			}, nil
		},
		UpdateRepoFn: func(owner, name string, repo *gh.Repository) (*gh.Repository, error) {
			updated = true
			if repo.GetDescription() != "New description" {
				t.Errorf("expected description 'New description', got %s", repo.GetDescription())
			}
			return repo, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
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
		Repos: []config.Repo{
			{Name: "my-repo", Description: "New description"},
		},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Error("expected repo to be updated")
	}
}

func TestApplySkipsUpToDateRepo(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Description:         gh.Ptr("Same"),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(true),
			}, nil
		},
		UpdateRepoFn: func(owner, name string, repo *gh.Repository) (*gh.Repository, error) {
			t.Error("UpdateRepo should not be called for up-to-date repo")
			return repo, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return nil, nil
		},
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil
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
			{Name: "my-repo", Description: "Same"},
		},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyDryRunDoesNotMutate(t *testing.T) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return nil, nil // doesn't exist
		},
		CreateRepoFn: func(owner string, isOrg bool, repo *gh.Repository) (*gh.Repository, error) {
			t.Error("CreateRepo should not be called in dry-run mode")
			return repo, nil
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

	err := RunWith(m, cfg, Options{DryRun: true, Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyLabelsCreateAndUpdate(t *testing.T) {
	var createdLabels, updatedLabels []string
	var mu sync.Mutex

	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(false),
			}, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return []*gh.Label{
				{Name: gh.Ptr("bug"), Color: gh.Ptr("old-color"), Description: gh.Ptr("Old desc")},
			}, nil
		},
		CreateLabelFn: func(owner, repo string, label config.Label) error {
			mu.Lock()
			createdLabels = append(createdLabels, label.Name)
			mu.Unlock()
			return nil
		},
		UpdateLabelFn: func(owner, repo, currentName string, label config.Label) error {
			mu.Lock()
			updatedLabels = append(updatedLabels, label.Name)
			mu.Unlock()
			return nil
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
		Labels: config.Labels{
			Items: []config.Label{
				{Name: "bug", Color: "new-color", Description: "New desc"},     // update
				{Name: "feature", Color: "0e8a16", Description: "New feature"}, // create
			},
		},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(createdLabels) != 1 || createdLabels[0] != "feature" {
		t.Errorf("expected created label 'feature', got %v", createdLabels)
	}
	if len(updatedLabels) != 1 || updatedLabels[0] != "bug" {
		t.Errorf("expected updated label 'bug', got %v", updatedLabels)
	}
}

func TestApplyBranchProtection(t *testing.T) {
	var protectionApplied bool
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
		ApplyBranchProtectionFn: func(owner, repo, branch string, bp config.BranchProtection) error {
			protectionApplied = true
			if branch != "main" {
				t.Errorf("expected branch 'main', got %s", branch)
			}
			return nil
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
			BranchProtection: config.BranchProtection{Preset: "standard"},
		},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !protectionApplied {
		t.Error("expected branch protection to be applied")
	}
}

func TestApplyRepoSettingsMergeAndFeatures(t *testing.T) {
	var updated *gh.Repository
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
				HasIssues:           gh.Ptr(true),
				HasWiki:             gh.Ptr(true),
				HasDiscussions:      gh.Ptr(false),
			}, nil
		},
		UpdateRepoFn: func(owner, name string, repo *gh.Repository) (*gh.Repository, error) {
			updated = repo
			return repo, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
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
			AllowSquashMerge: gh.Ptr(true),
			AllowMergeCommit: gh.Ptr(false), // disable merge commits
			AllowRebaseMerge: gh.Ptr(false), // disable rebase
			AllowAutoMerge:   true,
			HasIssues:        gh.Ptr(true),
			HasWiki:          gh.Ptr(false), // disable wiki
			HasDiscussions:   gh.Ptr(true),  // enable discussions
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Repos: []config.Repo{
			{Name: "my-repo"},
		},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("expected repo to be updated")
	}
	if updated.GetAllowMergeCommit() != false {
		t.Error("expected allow_merge_commit to be false")
	}
	if updated.GetAllowRebaseMerge() != false {
		t.Error("expected allow_rebase_merge to be false")
	}
	if updated.GetAllowAutoMerge() != true {
		t.Error("expected allow_auto_merge to be true")
	}
	if updated.GetHasWiki() != false {
		t.Error("expected has_wiki to be false")
	}
	if updated.GetHasDiscussions() != true {
		t.Error("expected has_discussions to be true")
	}
}

func TestApplyLabelsDeleteUnwanted(t *testing.T) {
	var deleted []string
	var mu sync.Mutex

	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(false),
			}, nil
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return []*gh.Label{
				{Name: gh.Ptr("bug"), Color: gh.Ptr("d73a4a"), Description: gh.Ptr("Something broken")},
				{Name: gh.Ptr("wontfix"), Color: gh.Ptr("ffffff"), Description: gh.Ptr("Not fixing")},
				{Name: gh.Ptr("duplicate"), Color: gh.Ptr("cfd3d7"), Description: gh.Ptr("Already exists")},
			}, nil
		},
		CreateLabelFn: func(owner, repo string, label config.Label) error {
			return nil
		},
		UpdateLabelFn: func(owner, repo, currentName string, label config.Label) error {
			return nil
		},
		DeleteLabelFn: func(owner, repo, name string) error {
			mu.Lock()
			deleted = append(deleted, name)
			mu.Unlock()
			return nil
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
		Labels: config.Labels{
			ReplaceDefaults: true,
			Items: []config.Label{
				{Name: "bug", Color: "d73a4a", Description: "Something broken"},
			},
		},
		Repos: []config.Repo{{Name: "my-repo"}},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deleted) != 2 {
		t.Fatalf("expected 2 labels deleted, got %d: %v", len(deleted), deleted)
	}
	for _, d := range deleted {
		if d != "wontfix" && d != "duplicate" {
			t.Errorf("unexpected label deleted: %s", d)
		}
	}
}

func TestApplyTeamRemovesMember(t *testing.T) {
	var removed []string
	var mu sync.Mutex

	m := &mock.Client{
		GetTeamFn: func(org, slug string) (*gh.Team, error) {
			return &gh.Team{Slug: gh.Ptr("backend")}, nil
		},
		UpdateTeamFn: func(org, slug, name, description, permission string) (*gh.Team, error) {
			return &gh.Team{Slug: gh.Ptr("backend")}, nil
		},
		ListTeamMembersFn: func(org, slug string) ([]*gh.User, error) {
			return []*gh.User{
				{Login: gh.Ptr("alice")},
				{Login: gh.Ptr("bob")},   // not in desired → should be removed
				{Login: gh.Ptr("carol")}, // not in desired → should be removed
			}, nil
		},
		AddTeamMemberFn: func(org, slug, username string) error {
			return nil
		},
		RemoveTeamMemberFn: func(org, slug, username string) error {
			mu.Lock()
			removed = append(removed, username)
			mu.Unlock()
			return nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "organization", Name: "myorg"},
		Defaults: config.Defaults{
			Visibility:       "private",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Teams: []config.Team{
			{Name: "backend", Description: "Backend devs", Permission: "write", Members: []string{"alice"}},
		},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("expected 2 members removed, got %d: %v", len(removed), removed)
	}
	for _, r := range removed {
		if r != "bob" && r != "carol" {
			t.Errorf("unexpected member removed: %s", r)
		}
	}
}

func TestApplyGovernanceFiles(t *testing.T) {
	var filesWritten []string
	var mu sync.Mutex

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
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil // file doesn't exist
		},
		CreateOrUpdateFileFn: func(owner, repo, path, message string, content []byte, sha *string) error {
			mu.Lock()
			filesWritten = append(filesWritten, path)
			mu.Unlock()
			return nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Governance: config.Governance{
			Contributing:   true,
			CodeOfConduct:  true,
			SecurityPolicy: true,
			Codeowners:     "* @testuser",
		},
		Repos: []config.Repo{{Name: "my-repo"}},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filesWritten) < 4 {
		t.Fatalf("expected at least 4 governance files, got %d: %v", len(filesWritten), filesWritten)
	}
	expected := map[string]bool{
		"CONTRIBUTING.md":    false,
		"CODE_OF_CONDUCT.md": false,
		"SECURITY.md":        false,
		".github/CODEOWNERS": false,
	}
	for _, f := range filesWritten {
		if _, ok := expected[f]; ok {
			expected[f] = true
		}
	}
	for f, found := range expected {
		if !found {
			t.Errorf("expected governance file %s to be written", f)
		}
	}
}

func TestApplyCIWorkflow(t *testing.T) {
	var writtenPath string
	var writtenContent []byte

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
		GetFileContentFn: func(owner, repo, path string) (string, *string, error) {
			return "", nil, nil
		},
		CreateOrUpdateFileFn: func(owner, repo, path, message string, content []byte, sha *string) error {
			if path == ".github/workflows/ci.yml" {
				writtenPath = path
				writtenContent = content
			}
			return nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "individual", Name: "testuser"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Repos: []config.Repo{{Name: "my-repo", CI: "go"}},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writtenPath != ".github/workflows/ci.yml" {
		t.Errorf("expected CI workflow to be written, got path %q", writtenPath)
	}
	if len(writtenContent) == 0 {
		t.Error("expected non-empty CI workflow content")
	}
}

func TestApplyOrgTeamCreate(t *testing.T) {
	var teamCreated bool
	var memberAdded string
	m := &mock.Client{
		GetTeamFn: func(org, slug string) (*gh.Team, error) {
			return nil, nil // team doesn't exist
		},
		CreateTeamFn: func(org, name, description, permission string) (*gh.Team, error) {
			teamCreated = true
			if name != "backend" {
				t.Errorf("expected team 'backend', got %s", name)
			}
			return &gh.Team{Slug: gh.Ptr("backend")}, nil
		},
		ListTeamMembersFn: func(org, slug string) ([]*gh.User, error) {
			return nil, nil
		},
		AddTeamMemberFn: func(org, slug, username string) error {
			memberAdded = username
			return nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "organization", Name: "myorg"},
		Defaults: config.Defaults{
			Visibility:       "private",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Teams: []config.Team{
			{Name: "backend", Description: "Backend devs", Permission: "write", Members: []string{"alice"}},
		},
	}

	err := RunWith(m, cfg, Options{Concurrency: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !teamCreated {
		t.Error("expected team to be created")
	}
	if memberAdded != "alice" {
		t.Errorf("expected member 'alice' to be added, got %q", memberAdded)
	}
}
