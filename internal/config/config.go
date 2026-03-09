package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// Config is the top-level gh-setup configuration.
type Config struct {
	Account    Account    `yaml:"account"`
	Defaults   Defaults   `yaml:"defaults"`
	Labels     Labels     `yaml:"labels"`
	RepoScope  string     `yaml:"repo_scope,omitempty"`
	Repos      []Repo     `yaml:"repos"`
	Teams      []Team     `yaml:"teams,omitempty"`
	Governance Governance `yaml:"governance"`
	Security   Security   `yaml:"security"`
	Secrets    []Secret   `yaml:"secrets,omitempty"`
}

type Account struct {
	Type string `yaml:"type"` // individual | organization
	Name string `yaml:"name"`
}

type Defaults struct {
	Visibility          string           `yaml:"visibility"`
	DefaultBranch       string           `yaml:"default_branch"`
	DeleteBranchOnMerge bool             `yaml:"delete_branch_on_merge"`
	BranchProtection    BranchProtection `yaml:"branch_protection"`
}

type BranchProtection struct {
	Preset              string   `yaml:"preset"` // none | basic | standard | strict | custom
	RequirePR           bool     `yaml:"require_pr,omitempty"`
	RequiredApprovals   int      `yaml:"required_approvals,omitempty"`
	DismissStaleReviews bool     `yaml:"dismiss_stale_reviews,omitempty"`
	RequireStatusChecks bool     `yaml:"require_status_checks,omitempty"`
	StatusChecks        []string `yaml:"status_checks,omitempty"`
	RequireUpToDate     bool     `yaml:"require_up_to_date,omitempty"`
	EnforceAdmins       bool     `yaml:"enforce_admins,omitempty"`
	AllowForcePush      bool     `yaml:"allow_force_push,omitempty"`
	AllowDeletions      bool     `yaml:"allow_deletions,omitempty"`
}

type Labels struct {
	ReplaceDefaults bool    `yaml:"replace_defaults"`
	Items           []Label `yaml:"items"`
}

type Label struct {
	Name        string `yaml:"name"`
	Color       string `yaml:"color"`
	Description string `yaml:"description"`
}

type Repo struct {
	Name            string            `yaml:"name"`
	Description     string            `yaml:"description,omitempty"`
	Topics          []string          `yaml:"topics,omitempty"`
	Visibility      string            `yaml:"visibility,omitempty"`
	Homepage        string            `yaml:"homepage,omitempty"`
	CI              string            `yaml:"ci,omitempty"`
	ExtraProtection *BranchProtection `yaml:"extra_protection,omitempty"`
}

type Team struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Permission  string   `yaml:"permission"` // read | write | admin
	Members     []string `yaml:"members"`
}

type Governance struct {
	Contributing   bool   `yaml:"contributing"`
	CodeOfConduct  bool   `yaml:"code_of_conduct"`
	SecurityPolicy bool   `yaml:"security_policy"`
	Codeowners     string `yaml:"codeowners,omitempty"`
}

type Security struct {
	Dependabot    bool `yaml:"dependabot"`
	SecretScanning bool `yaml:"secret_scanning"`
	CodeScanning  bool `yaml:"code_scanning"`
}

type Secret struct {
	Name  string `yaml:"name"`
	Scope string `yaml:"scope"` // org | repo
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
