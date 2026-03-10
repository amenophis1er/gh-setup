package importer

import (
	"fmt"
	"strings"

	"github.com/amenophis1er/gh-setup/internal/config"
	ghclient "github.com/amenophis1er/gh-setup/internal/github"
	"github.com/charmbracelet/log"
	gh "github.com/google/go-github/v68/github"
)

// Options configures the import behavior.
type Options struct {
	Account  string
	RepoName string // if set, import only this repo
}

// Run imports the current GitHub state into a Config.
func Run(opts Options) (*config.Config, error) {
	client, err := ghclient.NewClient()
	if err != nil {
		return nil, err
	}

	isOrg, err := client.IsOrganization(opts.Account)
	if err != nil {
		return nil, fmt.Errorf("checking account type: %w", err)
	}

	accountType := "individual"
	if isOrg {
		accountType = "organization"
	}

	cfg := &config.Config{
		Account: config.Account{
			Type: accountType,
			Name: opts.Account,
		},
	}

	// Import repos
	if opts.RepoName != "" {
		repo, err := importRepo(client, opts.Account, opts.RepoName)
		if err != nil {
			return nil, fmt.Errorf("importing repo %s: %w", opts.RepoName, err)
		}
		cfg.Repos = []config.Repo{repo}
		inferDefaults(cfg, client, opts.Account, opts.RepoName)
	} else {
		repos, err := client.ListRepos(opts.Account, isOrg)
		if err != nil {
			return nil, fmt.Errorf("listing repos: %w", err)
		}

		log.Info("Discovered repos", "count", len(repos))

		for _, r := range repos {
			if r.GetArchived() || r.GetFork() {
				continue
			}
			repo, err := importRepo(client, opts.Account, r.GetName())
			if err != nil {
				log.Error("Failed to import repo", "repo", r.GetName(), "err", err)
				continue
			}
			cfg.Repos = append(cfg.Repos, repo)
		}

		if len(cfg.Repos) > 0 {
			inferDefaults(cfg, client, opts.Account, cfg.Repos[0].Name)
		}
	}

	// Import teams (org only)
	if isOrg {
		teams, err := importTeams(client, opts.Account)
		if err != nil {
			log.Error("Failed to import teams", "err", err)
		} else {
			cfg.Teams = teams
		}
	}

	// Import labels from first repo as the default label set
	if len(cfg.Repos) > 0 {
		labels, err := importLabels(client, opts.Account, cfg.Repos[0].Name)
		if err != nil {
			log.Error("Failed to import labels", "err", err)
		} else {
			cfg.Labels = config.Labels{
				ReplaceDefaults: false,
				Items:           labels,
			}
		}
	}

	// Governance — check first repo
	if len(cfg.Repos) > 0 {
		cfg.Governance = importGovernance(client, opts.Account, cfg.Repos[0].Name)
	}

	return cfg, nil
}

func importRepo(client *ghclient.Client, owner, name string) (config.Repo, error) {
	r, err := client.GetRepo(owner, name)
	if err != nil {
		return config.Repo{}, err
	}
	if r == nil {
		return config.Repo{}, fmt.Errorf("repo %s not found", name)
	}

	repo := config.Repo{
		Name:        r.GetName(),
		Description: r.GetDescription(),
		Homepage:    r.GetHomepage(),
	}

	if r.GetPrivate() {
		repo.Visibility = "private"
	} else {
		repo.Visibility = "public"
	}

	repo.Topics = r.Topics

	content, _, err := client.GetFileContent(owner, name, ".github/workflows/ci.yml")
	if err == nil && content != "" {
		repo.CI = detectCITemplate(content)
	}

	return repo, nil
}

func inferDefaults(cfg *config.Config, client *ghclient.Client, owner, repoName string) {
	repo, err := client.GetRepo(owner, repoName)
	if err != nil || repo == nil {
		return
	}

	cfg.Defaults.DefaultBranch = repo.GetDefaultBranch()
	cfg.Defaults.DeleteBranchOnMerge = repo.GetDeleteBranchOnMerge()

	if repo.GetPrivate() {
		cfg.Defaults.Visibility = "private"
	} else {
		cfg.Defaults.Visibility = "public"
	}

	branch := repo.GetDefaultBranch()
	if branch == "" {
		branch = "main"
	}

	prot, err := client.GetBranchProtection(owner, repoName, branch)
	if err == nil && prot != nil {
		cfg.Defaults.BranchProtection = detectProtectionPreset(prot)
	} else {
		cfg.Defaults.BranchProtection = config.BranchProtection{Preset: "none"}
	}

	dependabot, err := client.GetVulnerabilityAlerts(owner, repoName)
	if err == nil {
		cfg.Security.Dependabot = dependabot
	}

	if repo.GetSecurityAndAnalysis() != nil {
		sa := repo.GetSecurityAndAnalysis()
		if sa.SecretScanning != nil && sa.SecretScanning.GetStatus() == "enabled" {
			cfg.Security.SecretScanning = true
		}
		if sa.AdvancedSecurity != nil && sa.AdvancedSecurity.GetStatus() == "enabled" {
			cfg.Security.CodeScanning = true
		}
	}
}

func detectProtectionPreset(prot *gh.Protection) config.BranchProtection {
	hasPR := prot.GetRequiredPullRequestReviews() != nil
	hasStatus := prot.GetRequiredStatusChecks() != nil
	allowForce := prot.GetAllowForcePushes() != nil && prot.GetAllowForcePushes().Enabled
	allowDelete := prot.GetAllowDeletions() != nil && prot.GetAllowDeletions().Enabled
	requireUpToDate := hasStatus && prot.GetRequiredStatusChecks().Strict

	switch {
	case !hasPR && !hasStatus && !allowForce && !allowDelete:
		return config.BranchProtection{Preset: "basic"}
	case hasPR && !hasStatus && !allowForce && !allowDelete:
		approvals := 0
		if prot.GetRequiredPullRequestReviews() != nil {
			approvals = prot.GetRequiredPullRequestReviews().RequiredApprovingReviewCount
		}
		if approvals == 1 {
			return config.BranchProtection{Preset: "standard"}
		}
	case hasPR && hasStatus && requireUpToDate && !allowForce && !allowDelete:
		return config.BranchProtection{Preset: "strict"}
	}

	// Custom — fill in all fields
	bp := config.BranchProtection{Preset: "custom"}
	if hasPR {
		bp.RequirePR = true
		bp.RequiredApprovals = prot.GetRequiredPullRequestReviews().RequiredApprovingReviewCount
		bp.DismissStaleReviews = prot.GetRequiredPullRequestReviews().DismissStaleReviews
	}
	if hasStatus {
		bp.RequireStatusChecks = true
		bp.RequireUpToDate = prot.GetRequiredStatusChecks().Strict
		if prot.GetRequiredStatusChecks().Checks != nil {
			for _, check := range *prot.GetRequiredStatusChecks().Checks {
				bp.StatusChecks = append(bp.StatusChecks, check.Context)
			}
		}
	}
	bp.AllowForcePush = allowForce
	bp.AllowDeletions = allowDelete
	if prot.GetEnforceAdmins() != nil {
		bp.EnforceAdmins = prot.GetEnforceAdmins().Enabled
	}

	return bp
}

func importLabels(client *ghclient.Client, owner, repo string) ([]config.Label, error) {
	ghLabels, err := client.ListLabels(owner, repo)
	if err != nil {
		return nil, err
	}

	var labels []config.Label
	for _, l := range ghLabels {
		labels = append(labels, config.Label{
			Name:        l.GetName(),
			Color:       l.GetColor(),
			Description: l.GetDescription(),
		})
	}
	return labels, nil
}

func importTeams(client *ghclient.Client, org string) ([]config.Team, error) {
	ghTeams, err := client.ListOrgTeams(org)
	if err != nil {
		return nil, err
	}

	var teams []config.Team
	for _, t := range ghTeams {
		members, err := client.ListTeamMembers(org, t.GetSlug())
		if err != nil {
			log.Error("Failed to list team members", "team", t.GetName(), "err", err)
			continue
		}

		var memberNames []string
		for _, m := range members {
			memberNames = append(memberNames, m.GetLogin())
		}

		perm := "read"
		if t.GetPermission() == "admin" {
			perm = "admin"
		} else if t.GetPermission() == "push" {
			perm = "write"
		}

		teams = append(teams, config.Team{
			Name:        t.GetSlug(),
			Description: t.GetDescription(),
			Permission:  perm,
			Members:     memberNames,
		})
	}

	return teams, nil
}

func importGovernance(client *ghclient.Client, owner, repo string) config.Governance {
	gov := config.Governance{}

	if content, _, err := client.GetFileContent(owner, repo, "CONTRIBUTING.md"); err == nil && content != "" {
		gov.Contributing = true
	}
	if content, _, err := client.GetFileContent(owner, repo, "CODE_OF_CONDUCT.md"); err == nil && content != "" {
		gov.CodeOfConduct = true
	}
	if content, _, err := client.GetFileContent(owner, repo, "SECURITY.md"); err == nil && content != "" {
		gov.SecurityPolicy = true
	}
	if content, _, err := client.GetFileContent(owner, repo, ".github/CODEOWNERS"); err == nil && content != "" {
		gov.Codeowners = strings.TrimSpace(content)
	}

	return gov
}

func detectCITemplate(content string) string {
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(lower, "cargo test") || strings.Contains(lower, "cargo clippy"):
		return "rust"
	case strings.Contains(lower, "go test") || strings.Contains(lower, "go vet"):
		return "go"
	case strings.Contains(lower, "npm test") || strings.Contains(lower, "npm ci"):
		return "node"
	case strings.Contains(lower, "pytest") || strings.Contains(lower, "ruff"):
		return "python"
	case strings.Contains(lower, "docker/build-push-action") || strings.Contains(lower, "docker build"):
		return "docker"
	case strings.Contains(lower, "terraform validate") || strings.Contains(lower, "terraform plan"):
		return "terraform"
	case strings.Contains(lower, "mvn ") || strings.Contains(lower, "maven"):
		return "java"
	case strings.Contains(lower, "rspec") || strings.Contains(lower, "rubocop"):
		return "ruby"
	default:
		return ""
	}
}
