package apply

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/amenophis1er/gh-setup/internal/config"
	ghclient "github.com/amenophis1er/gh-setup/internal/github"
	"github.com/amenophis1er/gh-setup/internal/templates"
	"github.com/charmbracelet/huh"
	gh "github.com/google/go-github/v68/github"
	"golang.org/x/sync/errgroup"
)

// Run applies the config to GitHub using a new authenticated client.
func Run(cfg *config.Config, opts Options) error {
	client, err := ghclient.NewClient()
	if err != nil {
		return err
	}
	return RunWith(client, cfg, opts)
}

// Options configures the apply behavior.
type Options struct {
	DryRun         bool
	Interactive    bool
	NonInteractive bool
	Concurrency    int
}

// RunWith applies the config to GitHub using the provided client.
func RunWith(client ghclient.GitHubClient, cfg *config.Config, opts Options) error {
	owner := cfg.Account.Name
	isOrg := cfg.Account.Type == "organization"

	var errs []string
	var mu sync.Mutex

	concurrency := opts.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	// Interactive mode requires sequential execution for prompts
	if opts.Interactive {
		concurrency = 1
	}

	repos := cfg.Repos
	if cfg.RepoScope == "all" {
		discovered, err := client.ListRepos(owner, isOrg)
		if err != nil {
			return fmt.Errorf("listing repos: %w", err)
		}
		names := make([]string, len(discovered))
		for i, d := range discovered {
			names[i] = d.GetName()
		}
		repos = config.MergeRepoScope(names, cfg.Repos)
	}

	// Apply repos
	logHeader("Repositories")
	g := new(errgroup.Group)
	g.SetLimit(concurrency)

	for _, repo := range repos {
		repo := repo
		g.Go(func() error {
			if err := applyRepo(client, cfg, owner, isOrg, repo, opts); err != nil {
				logError("apply", repo.Name, err)
				mu.Lock()
				errs = append(errs, fmt.Sprintf("repo %s: %s", repo.Name, err))
				mu.Unlock()
			}
			return nil
		})
	}
	_ = g.Wait()

	// Apply teams (org only)
	if isOrg && len(cfg.Teams) > 0 {
		logHeader("Teams")
		for _, team := range cfg.Teams {
			if err := applyTeam(client, owner, team, opts); err != nil {
				logError("apply", team.Name, err)
				errs = append(errs, fmt.Sprintf("team %s: %s", team.Name, err))
				continue
			}
		}
	}

	if len(errs) > 0 {
		fmt.Println()
		return fmt.Errorf("%d error(s) during apply:\n  - %s", len(errs), strings.Join(errs, "\n  - "))
	}

	return nil
}

func applyRepo(client ghclient.GitHubClient, cfg *config.Config, owner string, isOrg bool, repo config.Repo, opts Options) error {
	visibility := repo.Visibility
	if visibility == "" {
		visibility = cfg.Defaults.Visibility
	}

	existing, err := client.GetRepo(owner, repo.Name)
	if err != nil {
		return fmt.Errorf("fetching repo %s: %w", repo.Name, err)
	}

	if existing == nil {
		if opts.DryRun {
			logDryRun("create", "repo "+repo.Name)
		} else {
			if opts.Interactive && !confirm(fmt.Sprintf("Create repo %s?", repo.Name), opts) {
				logSkip("repo " + repo.Name)
				return nil
			}
			private := visibility == "private"
			newRepo := buildRepoPayload(repo, cfg.Defaults, private)
			newRepo.Name = gh.Ptr(repo.Name)
			newRepo.AutoInit = gh.Ptr(true)
			_, err := client.CreateRepo(owner, isOrg, newRepo)
			if err != nil {
				return fmt.Errorf("creating repo %s: %w", repo.Name, err)
			}
			logSuccess("created", "repo "+repo.Name)
		}
	} else {
		private := visibility == "private"
		d := cfg.Defaults
		needsUpdate := existing.GetDescription() != repo.Description ||
			existing.GetHomepage() != repo.Homepage ||
			existing.GetPrivate() != private ||
			existing.GetDeleteBranchOnMerge() != d.DeleteBranchOnMerge ||
			existing.GetAllowAutoMerge() != d.AllowAutoMerge ||
			config.BoolPtrDiffers(existing.GetAllowSquashMerge(), d.AllowSquashMerge) ||
			config.BoolPtrDiffers(existing.GetAllowMergeCommit(), d.AllowMergeCommit) ||
			config.BoolPtrDiffers(existing.GetAllowRebaseMerge(), d.AllowRebaseMerge) ||
			config.BoolPtrDiffers(existing.GetHasIssues(), d.HasIssues) ||
			config.BoolPtrDiffers(existing.GetHasWiki(), d.HasWiki) ||
			config.BoolPtrDiffers(existing.GetHasDiscussions(), d.HasDiscussions)

		if needsUpdate {
			if opts.DryRun {
				logDryRun("update", "repo "+repo.Name)
			} else {
				if opts.Interactive && !confirm(fmt.Sprintf("Update repo %s settings?", repo.Name), opts) {
					logSkip("repo " + repo.Name)
				} else {
					_, err := client.UpdateRepo(owner, repo.Name, buildRepoPayload(repo, cfg.Defaults, private))
					if err != nil {
						return fmt.Errorf("updating repo %s: %w", repo.Name, err)
					}
					logSuccess("updated", "repo "+repo.Name)
				}
			}
		} else {
			logSkip("repo " + repo.Name)
		}
	}

	// Topics
	if len(repo.Topics) > 0 {
		if opts.DryRun {
			logDryRun("set topics", repo.Name)
		} else {
			if err := client.SetTopics(owner, repo.Name, repo.Topics); err != nil {
				logError("set topics", repo.Name, err)
			} else {
				logSuccess("set topics", repo.Name)
			}
		}
	}

	// Labels
	if err := applyLabels(client, owner, repo.Name, cfg.Labels, opts); err != nil {
		logError("apply labels", repo.Name, err)
	}

	// Branch protection
	bp := config.ResolveProtection(cfg.Defaults.BranchProtection.Preset)
	if cfg.Defaults.BranchProtection.Preset == "custom" {
		bp = cfg.Defaults.BranchProtection
	}
	if repo.ExtraProtection != nil {
		bp = *repo.ExtraProtection
	}

	branch := cfg.Defaults.DefaultBranch
	if branch == "" {
		branch = "main"
	}

	if bp.Preset != "none" {
		if opts.DryRun {
			logDryRun("apply protection", repo.Name+"/"+branch)
		} else {
			if err := client.ApplyBranchProtection(owner, repo.Name, branch, bp); err != nil {
				logError("apply protection", repo.Name, err)
			} else {
				logSuccess("applied protection", repo.Name+"/"+branch)
			}
		}
	}

	// CI workflow
	if repo.CI != "" {
		if err := applyCIWorkflow(client, owner, repo.Name, repo.CI, opts); err != nil {
			logError("apply CI", repo.Name, err)
		}
	}

	// Dependabot config
	if cfg.Security.Dependabot {
		if err := applyDependabot(client, owner, repo.Name, repo.CI, opts); err != nil {
			logError("apply dependabot config", repo.Name, err)
		}
	}

	// Governance files
	if err := applyGovernance(client, owner, repo.Name, cfg.Governance, opts); err != nil {
		logError("apply governance", repo.Name, err)
	}

	// Security
	if err := applySecurity(client, owner, repo.Name, cfg.Security, opts); err != nil {
		logError("apply security", repo.Name, err)
	}

	// Secrets
	for _, secret := range cfg.Secrets {
		if err := applySecret(client, owner, repo.Name, isOrg, secret, opts); err != nil {
			logError("apply secret", secret.Name, err)
		}
	}

	return nil
}

func applyLabels(client ghclient.GitHubClient, owner, repo string, labels config.Labels, opts Options) error {
	if len(labels.Items) == 0 {
		return nil
	}

	existing, err := client.ListLabels(owner, repo)
	if err != nil {
		return err
	}

	existingMap := make(map[string]*gh.Label)
	for _, l := range existing {
		existingMap[strings.ToLower(l.GetName())] = l
	}

	// Delete default labels if configured
	if labels.ReplaceDefaults {
		desiredMap := make(map[string]bool)
		for _, l := range labels.Items {
			desiredMap[strings.ToLower(l.Name)] = true
		}
		for _, l := range existing {
			if !desiredMap[strings.ToLower(l.GetName())] {
				if opts.DryRun {
					logDryRun("delete label", l.GetName())
				} else {
					if err := client.DeleteLabel(owner, repo, l.GetName()); err != nil {
						logError("delete label", l.GetName(), err)
					} else {
						logSuccess("deleted label", l.GetName())
					}
				}
			}
		}
	}

	// Create or update desired labels
	for _, label := range labels.Items {
		if ex, ok := existingMap[strings.ToLower(label.Name)]; ok {
			if ex.GetColor() != label.Color || ex.GetDescription() != label.Description {
				if opts.DryRun {
					logDryRun("update label", label.Name)
				} else {
					if err := client.UpdateLabel(owner, repo, ex.GetName(), label); err != nil {
						logError("update label", label.Name, err)
					} else {
						logSuccess("updated label", label.Name)
					}
				}
			}
		} else {
			if opts.DryRun {
				logDryRun("create label", label.Name)
			} else {
				if err := client.CreateLabel(owner, repo, label); err != nil {
					logError("create label", label.Name, err)
				} else {
					logSuccess("created label", label.Name)
				}
			}
		}
	}

	return nil
}

func applyCIWorkflow(client ghclient.GitHubClient, owner, repo, ciName string, opts Options) error {
	content, err := templates.CIWorkflow(ciName)
	if err != nil {
		return fmt.Errorf("loading CI template %s: %w", ciName, err)
	}

	path := ".github/workflows/ci.yml"
	if opts.DryRun {
		logDryRun("create CI workflow", repo+" ("+ciName+")")
		return nil
	}

	_, sha, err := client.GetFileContent(owner, repo, path)
	if err != nil {
		return err
	}

	if err := client.CreateOrUpdateFile(owner, repo, path, "ci: add CI workflow", content, sha); err != nil {
		return err
	}
	logSuccess("applied CI workflow", repo+" ("+ciName+")")
	return nil
}

func applyDependabot(client ghclient.GitHubClient, owner, repo, ci string, opts Options) error {
	ecosystem := templates.CIToEcosystem(ci)
	content := templates.DependabotConfig(ecosystem)
	path := ".github/dependabot.yml"

	if opts.DryRun {
		logDryRun("create dependabot config", repo)
		return nil
	}

	existingContent, sha, err := client.GetFileContent(owner, repo, path)
	if err != nil {
		return err
	}

	// Skip if already exists with same content
	if existingContent == content {
		return nil
	}

	if err := client.CreateOrUpdateFile(owner, repo, path, "ci: add dependabot config", []byte(content), sha); err != nil {
		return err
	}
	logSuccess("applied dependabot config", repo)
	return nil
}

func applyGovernance(client ghclient.GitHubClient, owner, repo string, gov config.Governance, opts Options) error {
	files := map[string]func() string{}

	if gov.Contributing {
		files["CONTRIBUTING.md"] = func() string { return templates.Contributing(repo) }
	}
	if gov.CodeOfConduct {
		files["CODE_OF_CONDUCT.md"] = func() string { return templates.CodeOfConduct() }
	}
	if gov.SecurityPolicy {
		files["SECURITY.md"] = func() string { return templates.SecurityPolicy(repo) }
	}
	if gov.Codeowners != "" {
		files[".github/CODEOWNERS"] = func() string { return gov.Codeowners + "\n" }
	}

	for path, contentFn := range files {
		if opts.DryRun {
			logDryRun("create", path+" in "+repo)
			continue
		}

		_, sha, err := client.GetFileContent(owner, repo, path)
		if err != nil {
			return err
		}

		content := contentFn()
		if err := client.CreateOrUpdateFile(owner, repo, path, "docs: add "+path, []byte(content), sha); err != nil {
			logError("create", path, err)
		} else {
			logSuccess("applied", path+" in "+repo)
		}
	}

	return nil
}

func applySecurity(client ghclient.GitHubClient, owner, repo string, sec config.Security, opts Options) error {
	if opts.DryRun {
		logDryRun("apply security settings", repo)
		return nil
	}

	if sec.Dependabot {
		if err := client.EnableVulnerabilityAlerts(owner, repo); err != nil {
			logError("enable dependabot", repo, err)
		} else {
			logSuccess("enabled dependabot alerts", repo)
		}
	}

	if err := client.UpdateSecurityAndAnalysis(owner, repo, sec.SecretScanning, sec.CodeScanning); err != nil {
		logError("update security", repo, err)
	} else {
		logSuccess("applied security settings", repo)
	}

	return nil
}

func applyTeam(client ghclient.GitHubClient, org string, team config.Team, opts Options) error {
	existing, err := client.GetTeam(org, team.Name)
	if err != nil {
		return fmt.Errorf("fetching team %s: %w", team.Name, err)
	}

	if existing == nil {
		if opts.DryRun {
			logDryRun("create", "team "+team.Name)
		} else {
			if opts.Interactive && !confirm(fmt.Sprintf("Create team %s?", team.Name), opts) {
				logSkip("team " + team.Name)
				return nil
			}
			_, err := client.CreateTeam(org, team.Name, team.Description, team.Permission)
			if err != nil {
				return fmt.Errorf("creating team %s: %w", team.Name, err)
			}
			logSuccess("created", "team "+team.Name)
		}
	} else {
		if opts.DryRun {
			logSkip("team " + team.Name)
		} else {
			if _, err := client.UpdateTeam(org, team.Name, team.Name, team.Description, team.Permission); err != nil {
				logError("update", "team "+team.Name, err)
			}
		}
	}

	// Sync members
	if opts.DryRun {
		logDryRun("sync members", "team "+team.Name)
		return nil
	}

	currentMembers, err := client.ListTeamMembers(org, team.Name)
	if err != nil {
		return fmt.Errorf("listing team members: %w", err)
	}

	currentMap := make(map[string]bool)
	for _, m := range currentMembers {
		currentMap[m.GetLogin()] = true
	}

	desiredMap := make(map[string]bool)
	for _, m := range team.Members {
		desiredMap[m] = true
	}

	for _, m := range team.Members {
		if !currentMap[m] {
			if err := client.AddTeamMember(org, team.Name, m); err != nil {
				logError("add member", m+" to "+team.Name, err)
			} else {
				logSuccess("added member", m+" to "+team.Name)
			}
		}
	}

	for _, m := range currentMembers {
		if !desiredMap[m.GetLogin()] {
			if err := client.RemoveTeamMember(org, team.Name, m.GetLogin()); err != nil {
				logError("remove member", m.GetLogin()+" from "+team.Name, err)
			} else {
				logSuccess("removed member", m.GetLogin()+" from "+team.Name)
			}
		}
	}

	return nil
}

func applySecret(client ghclient.GitHubClient, owner, repo string, isOrg bool, secret config.Secret, opts Options) error {
	if opts.DryRun {
		logDryRun("set secret", secret.Name+" ("+secret.Scope+")")
		return nil
	}

	// Check for env var GH_SETUP_SECRET_<NAME> (uppercased)
	envKey := "GH_SETUP_SECRET_" + strings.ToUpper(secret.Name)
	value := os.Getenv(envKey)

	if value == "" {
		if opts.NonInteractive {
			return fmt.Errorf("secret %s: value must be provided via env var %s in non-interactive mode", secret.Name, envKey)
		}

		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Enter value for secret %s", secret.Name)).
					EchoMode(huh.EchoModePassword).
					Value(&value),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return err
		}
	}

	if secret.Scope == "org" && isOrg {
		if err := client.SetOrgSecret(owner, secret.Name, value); err != nil {
			return err
		}
	} else {
		if err := client.SetRepoSecret(owner, repo, secret.Name, value); err != nil {
			return err
		}
	}
	logSuccess("set secret", secret.Name)
	return nil
}

func buildRepoPayload(repo config.Repo, defaults config.Defaults, private bool) *gh.Repository {
	r := &gh.Repository{
		Description:         gh.Ptr(repo.Description),
		Homepage:            gh.Ptr(repo.Homepage),
		Private:             gh.Ptr(private),
		DeleteBranchOnMerge: gh.Ptr(defaults.DeleteBranchOnMerge),
		AllowAutoMerge:      gh.Ptr(defaults.AllowAutoMerge),
	}
	if defaults.AllowSquashMerge != nil {
		r.AllowSquashMerge = defaults.AllowSquashMerge
	}
	if defaults.AllowMergeCommit != nil {
		r.AllowMergeCommit = defaults.AllowMergeCommit
	}
	if defaults.AllowRebaseMerge != nil {
		r.AllowRebaseMerge = defaults.AllowRebaseMerge
	}
	if defaults.HasIssues != nil {
		r.HasIssues = defaults.HasIssues
	}
	if defaults.HasWiki != nil {
		r.HasWiki = defaults.HasWiki
	}
	if defaults.HasDiscussions != nil {
		r.HasDiscussions = defaults.HasDiscussions
	}
	return r
}

func confirm(msg string, opts Options) bool {
	if opts.NonInteractive {
		return true
	}
	var yes bool
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(msg).
				Value(&yes),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return false
	}
	return yes
}
