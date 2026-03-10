package diff

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/github/mock"
	"github.com/charmbracelet/lipgloss"
	gh "github.com/google/go-github/v68/github"
)

var update = flag.Bool("update", false, "update golden files")

func TestMain(m *testing.M) {
	// Disable ANSI colors so golden files are deterministic plain text.
	lipgloss.SetColorProfile(0)
	os.Exit(m.Run())
}

func goldenPath(name string) string {
	return filepath.Join("testdata", name+".golden")
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := goldenPath(name)

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0644); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden file %s (run with -update to create): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("output does not match golden file %s\n--- want ---\n%s\n--- got ---\n%s", path, want, got)
	}
}

func mixedScenario() (*mock.Client, *config.Config) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			if name == "existing-repo" {
				return &gh.Repository{
					Name:                gh.Ptr("existing-repo"),
					Description:         gh.Ptr("Old description"),
					Homepage:            gh.Ptr(""),
					Private:             gh.Ptr(true),
					DeleteBranchOnMerge: gh.Ptr(false),
					AllowAutoMerge:      gh.Ptr(false),
				}, nil
			}
			return nil, nil // new-repo doesn't exist
		},
		ListLabelsFn: func(owner, repo string) ([]*gh.Label, error) {
			return []*gh.Label{
				{Name: gh.Ptr("wontfix"), Color: gh.Ptr("ffffff"), Description: gh.Ptr("")},
			}, nil
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
			ReplaceDefaults: true,
			Items: []config.Label{
				{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
			},
		},
		Repos: []config.Repo{
			{Name: "existing-repo", Description: "New description"},
			{Name: "new-repo", Description: "Brand new"},
		},
	}
	return m, cfg
}

func upToDateScenario() (*mock.Client, *config.Config) {
	m := &mock.Client{
		GetRepoFn: func(owner, name string) (*gh.Repository, error) {
			return &gh.Repository{
				Name:                gh.Ptr("my-repo"),
				Description:         gh.Ptr("A repo"),
				Homepage:            gh.Ptr(""),
				Private:             gh.Ptr(false),
				DeleteBranchOnMerge: gh.Ptr(true),
				AllowAutoMerge:      gh.Ptr(false),
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
			{Name: "my-repo", Description: "A repo"},
		},
	}
	return m, cfg
}

func teamScenario() (*mock.Client, *config.Config) {
	m := &mock.Client{
		GetTeamFn: func(org, slug string) (*gh.Team, error) {
			return &gh.Team{Slug: gh.Ptr("core")}, nil
		},
		ListTeamMembersFn: func(org, slug string) ([]*gh.User, error) {
			return []*gh.User{
				{Login: gh.Ptr("alice")},
				{Login: gh.Ptr("charlie")},
			}, nil
		},
	}

	cfg := &config.Config{
		Account: config.Account{Type: "organization", Name: "myorg"},
		Defaults: config.Defaults{
			Visibility:       "public",
			DefaultBranch:    "main",
			BranchProtection: config.BranchProtection{Preset: "none"},
		},
		Teams: []config.Team{
			{Name: "core", Permission: "admin", Members: []string{"alice", "bob"}},
		},
	}
	return m, cfg
}

func TestGoldenJSON_MixedChanges(t *testing.T) {
	m, cfg := mixedScenario()
	result, err := Compute(m, cfg, 1)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, result); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "mixed_json", buf.Bytes())
}

func TestGoldenText_MixedChanges(t *testing.T) {
	m, cfg := mixedScenario()
	result, err := Compute(m, cfg, 1)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	RenderText(&buf, result)
	assertGolden(t, "mixed_text", buf.Bytes())
}

func TestGoldenJSON_UpToDate(t *testing.T) {
	m, cfg := upToDateScenario()
	result, err := Compute(m, cfg, 1)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, result); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "uptodate_json", buf.Bytes())
}

func TestGoldenText_UpToDate(t *testing.T) {
	m, cfg := upToDateScenario()
	result, err := Compute(m, cfg, 1)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	RenderText(&buf, result)
	assertGolden(t, "uptodate_text", buf.Bytes())
}

func TestGoldenJSON_TeamChanges(t *testing.T) {
	m, cfg := teamScenario()
	result, err := Compute(m, cfg, 1)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := RenderJSON(&buf, result); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "team_json", buf.Bytes())
}

func TestGoldenText_TeamChanges(t *testing.T) {
	m, cfg := teamScenario()
	result, err := Compute(m, cfg, 1)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	RenderText(&buf, result)
	assertGolden(t, "team_text", buf.Bytes())
}
