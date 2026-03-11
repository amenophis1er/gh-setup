package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")

	cfg := &Config{
		Account: Account{Type: "organization", Name: "test-org"},
		Defaults: Defaults{
			Visibility:          "public",
			DefaultBranch:       "main",
			DeleteBranchOnMerge: true,
			BranchProtection:    BranchProtection{Preset: "standard"},
		},
		Labels: Labels{
			ReplaceDefaults: true,
			Items: []Label{
				{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
			},
		},
		Repos: []Repo{
			{
				Name:        "my-repo",
				Description: "A test repo",
				Topics:      []string{"go", "test"},
				CI:          "go",
			},
		},
		Teams: []Team{
			{Name: "core", Permission: "admin", Members: []string{"user1"}},
		},
		Governance: Governance{
			Contributing:   true,
			CodeOfConduct:  true,
			SecurityPolicy: true,
			Codeowners:     "* @test-org/core",
		},
		Security: Security{
			Dependabot:     true,
			SecretScanning: true,
		},
		Secrets: []Secret{
			{Name: "MY_TOKEN", Scope: "org"},
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify key fields
	if loaded.Account.Type != "organization" {
		t.Errorf("Account.Type = %q, want %q", loaded.Account.Type, "organization")
	}
	if loaded.Account.Name != "test-org" {
		t.Errorf("Account.Name = %q, want %q", loaded.Account.Name, "test-org")
	}
	if loaded.Defaults.Visibility != "public" {
		t.Errorf("Defaults.Visibility = %q, want %q", loaded.Defaults.Visibility, "public")
	}
	if loaded.Defaults.BranchProtection.Preset != "standard" {
		t.Errorf("BranchProtection.Preset = %q, want %q", loaded.Defaults.BranchProtection.Preset, "standard")
	}
	if len(loaded.Repos) != 1 {
		t.Fatalf("len(Repos) = %d, want 1", len(loaded.Repos))
	}
	if loaded.Repos[0].Name != "my-repo" {
		t.Errorf("Repos[0].Name = %q, want %q", loaded.Repos[0].Name, "my-repo")
	}
	if len(loaded.Repos[0].Topics) != 2 {
		t.Errorf("len(Repos[0].Topics) = %d, want 2", len(loaded.Repos[0].Topics))
	}
	if len(loaded.Teams) != 1 {
		t.Fatalf("len(Teams) = %d, want 1", len(loaded.Teams))
	}
	if loaded.Teams[0].Members[0] != "user1" {
		t.Errorf("Teams[0].Members[0] = %q, want %q", loaded.Teams[0].Members[0], "user1")
	}
	if len(loaded.Secrets) != 1 {
		t.Fatalf("len(Secrets) = %d, want 1", len(loaded.Secrets))
	}
	if loaded.Secrets[0].Scope != "org" {
		t.Errorf("Secrets[0].Scope = %q, want %q", loaded.Secrets[0].Scope, "org")
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("Load() expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(path, []byte(":::invalid"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for invalid YAML")
	}
}

func TestResolveProtection(t *testing.T) {
	tests := []struct {
		preset              string
		wantRequirePR       bool
		wantAllowForcePush  bool
		wantRequireStatus   bool
		wantRequireUpToDate bool
	}{
		{"none", false, false, false, false},
		{"basic", false, false, false, false},
		{"standard", true, false, false, false},
		{"strict", true, false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			bp := ResolveProtection(tt.preset)
			if bp.RequirePR != tt.wantRequirePR {
				t.Errorf("RequirePR = %v, want %v", bp.RequirePR, tt.wantRequirePR)
			}
			if bp.AllowForcePush != tt.wantAllowForcePush {
				t.Errorf("AllowForcePush = %v, want %v", bp.AllowForcePush, tt.wantAllowForcePush)
			}
			if bp.RequireStatusChecks != tt.wantRequireStatus {
				t.Errorf("RequireStatusChecks = %v, want %v", bp.RequireStatusChecks, tt.wantRequireStatus)
			}
			if bp.RequireUpToDate != tt.wantRequireUpToDate {
				t.Errorf("RequireUpToDate = %v, want %v", bp.RequireUpToDate, tt.wantRequireUpToDate)
			}
		})
	}
}

func TestValidateValid(t *testing.T) {
	cfg := &Config{
		Account:  Account{Type: "organization", Name: "my-org"},
		Defaults: Defaults{Visibility: "public", BranchProtection: BranchProtection{Preset: "standard"}},
		Repos:    []Repo{{Name: "my-repo"}},
		Teams:    []Team{{Name: "core", Permission: "admin"}},
		Secrets:  []Secret{{Name: "TOKEN", Scope: "org"}},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestValidateEmptyAccountName(t *testing.T) {
	cfg := &Config{
		Account: Account{Type: "individual", Name: ""},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for empty account name")
	}
}

func TestValidateInvalidAccountType(t *testing.T) {
	cfg := &Config{
		Account: Account{Type: "enterprise", Name: "foo"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for invalid account type")
	}
}

func TestValidateInvalidVisibility(t *testing.T) {
	cfg := &Config{
		Account:  Account{Type: "individual", Name: "foo"},
		Defaults: Defaults{Visibility: "internal"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for invalid visibility")
	}
}

func TestValidateDuplicateRepos(t *testing.T) {
	cfg := &Config{
		Account: Account{Type: "individual", Name: "foo"},
		Repos:   []Repo{{Name: "dup"}, {Name: "dup"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for duplicate repos")
	}
}

func TestValidateEmptyRepoName(t *testing.T) {
	cfg := &Config{
		Account: Account{Type: "individual", Name: "foo"},
		Repos:   []Repo{{Name: ""}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for empty repo name")
	}
}

func TestValidateInvalidRepoName(t *testing.T) {
	cfg := &Config{
		Account: Account{Type: "individual", Name: "foo"},
		Repos:   []Repo{{Name: "repo with spaces"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for invalid repo name")
	}
}

func TestValidateTeamsOnIndividual(t *testing.T) {
	cfg := &Config{
		Account: Account{Type: "individual", Name: "foo"},
		Teams:   []Team{{Name: "core", Permission: "admin"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for teams on individual account")
	}
}

func TestValidateInvalidTeamPermission(t *testing.T) {
	cfg := &Config{
		Account: Account{Type: "organization", Name: "foo"},
		Teams:   []Team{{Name: "core", Permission: "superadmin"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for invalid team permission")
	}
}

func TestValidateInvalidSecretScope(t *testing.T) {
	cfg := &Config{
		Account: Account{Type: "individual", Name: "foo"},
		Secrets: []Secret{{Name: "TOKEN", Scope: "global"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for invalid secret scope")
	}
}

func TestValidateInvalidPreset(t *testing.T) {
	cfg := &Config{
		Account:  Account{Type: "individual", Name: "foo"},
		Defaults: Defaults{BranchProtection: BranchProtection{Preset: "ultra"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for invalid preset")
	}
}

func TestResolveProtectionApprovals(t *testing.T) {
	bp := ResolveProtection("standard")
	if bp.RequiredApprovals != 1 {
		t.Errorf("RequiredApprovals = %d, want 1", bp.RequiredApprovals)
	}

	bp = ResolveProtection("strict")
	if bp.RequiredApprovals != 1 {
		t.Errorf("RequiredApprovals = %d, want 1", bp.RequiredApprovals)
	}
}
