package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// Config is the top-level gh-setup configuration.
type Config struct {
	Account    Account    `yaml:"account" json:"account"`
	Defaults   Defaults   `yaml:"defaults" json:"defaults"`
	Labels     Labels     `yaml:"labels" json:"labels"`
	RepoScope  string     `yaml:"repo_scope,omitempty" json:"repo_scope,omitempty"`
	Repos      []Repo     `yaml:"repos" json:"repos"`
	Teams      []Team     `yaml:"teams,omitempty" json:"teams,omitempty"`
	Governance Governance `yaml:"governance" json:"governance"`
	Security   Security   `yaml:"security" json:"security"`
	Secrets    []Secret   `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

type Account struct {
	Type string `yaml:"type" json:"type"` // individual | organization
	Name string `yaml:"name" json:"name"`
}

type Defaults struct {
	Visibility          string           `yaml:"visibility" json:"visibility"`
	DefaultBranch       string           `yaml:"default_branch" json:"default_branch"`
	DeleteBranchOnMerge bool             `yaml:"delete_branch_on_merge" json:"delete_branch_on_merge"`
	AllowSquashMerge    *bool            `yaml:"allow_squash_merge,omitempty" json:"allow_squash_merge,omitempty"`
	AllowMergeCommit    *bool            `yaml:"allow_merge_commit,omitempty" json:"allow_merge_commit,omitempty"`
	AllowRebaseMerge    *bool            `yaml:"allow_rebase_merge,omitempty" json:"allow_rebase_merge,omitempty"`
	AllowAutoMerge      bool             `yaml:"allow_auto_merge" json:"allow_auto_merge"`
	HasIssues           *bool            `yaml:"has_issues,omitempty" json:"has_issues,omitempty"`
	HasWiki             *bool            `yaml:"has_wiki,omitempty" json:"has_wiki,omitempty"`
	HasDiscussions      *bool            `yaml:"has_discussions,omitempty" json:"has_discussions,omitempty"`
	BranchProtection    BranchProtection `yaml:"branch_protection" json:"branch_protection"`
}

type BranchProtection struct {
	Preset              string   `yaml:"preset" json:"preset"` // none | basic | standard | strict | custom
	RequirePR           bool     `yaml:"require_pr,omitempty" json:"require_pr,omitempty"`
	RequiredApprovals   int      `yaml:"required_approvals,omitempty" json:"required_approvals,omitempty"`
	DismissStaleReviews bool     `yaml:"dismiss_stale_reviews,omitempty" json:"dismiss_stale_reviews,omitempty"`
	RequireStatusChecks bool     `yaml:"require_status_checks,omitempty" json:"require_status_checks,omitempty"`
	StatusChecks        []string `yaml:"status_checks,omitempty" json:"status_checks,omitempty"`
	RequireUpToDate     bool     `yaml:"require_up_to_date,omitempty" json:"require_up_to_date,omitempty"`
	EnforceAdmins       bool     `yaml:"enforce_admins,omitempty" json:"enforce_admins,omitempty"`
	AllowForcePush      bool     `yaml:"allow_force_push,omitempty" json:"allow_force_push,omitempty"`
	AllowDeletions      bool     `yaml:"allow_deletions,omitempty" json:"allow_deletions,omitempty"`
}

type Labels struct {
	ReplaceDefaults bool    `yaml:"replace_defaults" json:"replace_defaults"`
	Items           []Label `yaml:"items" json:"items"`
}

type Label struct {
	Name        string `yaml:"name" json:"name"`
	Color       string `yaml:"color" json:"color"`
	Description string `yaml:"description" json:"description"`
}

type Repo struct {
	Name            string            `yaml:"name" json:"name"`
	Description     string            `yaml:"description,omitempty" json:"description,omitempty"`
	Topics          []string          `yaml:"topics,omitempty" json:"topics,omitempty"`
	Visibility      string            `yaml:"visibility,omitempty" json:"visibility,omitempty"`
	Homepage        string            `yaml:"homepage,omitempty" json:"homepage,omitempty"`
	CI              string            `yaml:"ci,omitempty" json:"ci,omitempty"`
	ExtraProtection *BranchProtection `yaml:"extra_protection,omitempty" json:"extra_protection,omitempty"`
}

type Team struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Permission  string   `yaml:"permission" json:"permission"` // read | write | admin
	Members     []string `yaml:"members" json:"members"`
}

type Governance struct {
	Contributing   bool   `yaml:"contributing" json:"contributing"`
	CodeOfConduct  bool   `yaml:"code_of_conduct" json:"code_of_conduct"`
	SecurityPolicy bool   `yaml:"security_policy" json:"security_policy"`
	Codeowners     string `yaml:"codeowners,omitempty" json:"codeowners,omitempty"`
}

type Security struct {
	Dependabot     bool `yaml:"dependabot" json:"dependabot"`
	SecretScanning bool `yaml:"secret_scanning" json:"secret_scanning"`
	CodeScanning   bool `yaml:"code_scanning" json:"code_scanning"`
}

type Secret struct {
	Name  string `yaml:"name" json:"name"`
	Scope string `yaml:"scope" json:"scope"` // org | repo
}

// Load reads and parses a gh-setup.yaml file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Marshal serializes the config to the given format ("yaml" or "json").
func Marshal(cfg *Config, format string) ([]byte, error) {
	switch format {
	case "json":
		return json.MarshalIndent(cfg, "", "  ")
	default:
		return yaml.Marshal(cfg)
	}
}

// Save writes the config to a YAML file.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Validate checks the config for common errors before apply.
func (c *Config) Validate() error {
	var errs []string

	if c.Account.Name == "" {
		errs = append(errs, "account.name is required")
	} else if !validNameRe.MatchString(c.Account.Name) {
		errs = append(errs, fmt.Sprintf("account.name %q contains invalid characters", c.Account.Name))
	}

	if c.Account.Type != "individual" && c.Account.Type != "organization" {
		errs = append(errs, fmt.Sprintf("account.type must be 'individual' or 'organization', got %q", c.Account.Type))
	}

	if c.Defaults.Visibility != "" && c.Defaults.Visibility != "public" && c.Defaults.Visibility != "private" {
		errs = append(errs, fmt.Sprintf("defaults.visibility must be 'public' or 'private', got %q", c.Defaults.Visibility))
	}

	validPresets := map[string]bool{"none": true, "basic": true, "standard": true, "strict": true, "custom": true}
	if c.Defaults.BranchProtection.Preset != "" && !validPresets[c.Defaults.BranchProtection.Preset] {
		errs = append(errs, fmt.Sprintf("branch_protection.preset %q is not valid", c.Defaults.BranchProtection.Preset))
	}

	if c.RepoScope != "" && c.RepoScope != "all" {
		errs = append(errs, fmt.Sprintf("repo_scope must be empty or 'all', got %q", c.RepoScope))
	}

	repoNames := make(map[string]bool)
	for i, r := range c.Repos {
		if r.Name == "" {
			errs = append(errs, fmt.Sprintf("repos[%d].name is required", i))
		} else if !validNameRe.MatchString(r.Name) {
			errs = append(errs, fmt.Sprintf("repos[%d].name %q contains invalid characters", i, r.Name))
		}
		if repoNames[r.Name] {
			errs = append(errs, fmt.Sprintf("repos[%d].name %q is duplicated", i, r.Name))
		}
		repoNames[r.Name] = true

		if r.Visibility != "" && r.Visibility != "public" && r.Visibility != "private" {
			errs = append(errs, fmt.Sprintf("repos[%d].visibility must be 'public' or 'private', got %q", i, r.Visibility))
		}
	}

	if c.Account.Type != "organization" && len(c.Teams) > 0 {
		errs = append(errs, "teams can only be configured for organizations")
	}

	teamNames := make(map[string]bool)
	for i, t := range c.Teams {
		if t.Name == "" {
			errs = append(errs, fmt.Sprintf("teams[%d].name is required", i))
		}
		if teamNames[t.Name] {
			errs = append(errs, fmt.Sprintf("teams[%d].name %q is duplicated", i, t.Name))
		}
		teamNames[t.Name] = true

		validPerms := map[string]bool{"read": true, "write": true, "admin": true}
		if t.Permission != "" && !validPerms[t.Permission] {
			errs = append(errs, fmt.Sprintf("teams[%d].permission must be 'read', 'write', or 'admin', got %q", i, t.Permission))
		}
	}

	for i, s := range c.Secrets {
		if s.Name == "" {
			errs = append(errs, fmt.Sprintf("secrets[%d].name is required", i))
		}
		if s.Scope != "org" && s.Scope != "repo" {
			errs = append(errs, fmt.Sprintf("secrets[%d].scope must be 'org' or 'repo', got %q", i, s.Scope))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// ResolveProtection returns the effective BranchProtection for a preset.
func ResolveProtection(preset string) BranchProtection {
	switch preset {
	case "basic":
		return BranchProtection{
			Preset:         "basic",
			AllowForcePush: false,
			AllowDeletions: false,
		}
	case "standard":
		return BranchProtection{
			Preset:            "standard",
			RequirePR:         true,
			RequiredApprovals: 1,
			AllowForcePush:    false,
			AllowDeletions:    false,
		}
	case "strict":
		return BranchProtection{
			Preset:              "strict",
			RequirePR:           true,
			RequiredApprovals:   1,
			RequireStatusChecks: true,
			RequireUpToDate:     true,
			AllowForcePush:      false,
			AllowDeletions:      false,
		}
	default: // "none" or "custom"
		return BranchProtection{Preset: preset}
	}
}

// BoolPtrDiffers returns true when desired is explicitly set and differs from actual.
func BoolPtrDiffers(actual bool, desired *bool) bool {
	return desired != nil && actual != *desired
}

// MergeRepoScope merges discovered repo names with explicitly configured repos.
// Configured repos override discovered ones by name; undiscovered config repos are appended.
func MergeRepoScope(discoveredNames []string, configRepos []Repo) []Repo {
	overrides := make(map[string]Repo, len(configRepos))
	for _, r := range configRepos {
		overrides[r.Name] = r
	}

	merged := make([]Repo, 0, len(discoveredNames)+len(configRepos))
	seen := make(map[string]bool, len(discoveredNames))
	for _, name := range discoveredNames {
		seen[name] = true
		if override, ok := overrides[name]; ok {
			merged = append(merged, override)
		} else {
			merged = append(merged, Repo{Name: name})
		}
	}
	for _, r := range configRepos {
		if !seen[r.Name] {
			merged = append(merged, r)
		}
	}
	return merged
}
