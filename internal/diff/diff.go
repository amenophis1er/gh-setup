package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/amenophis1er/gh-setup/internal/config"
	ghclient "github.com/amenophis1er/gh-setup/internal/github"
	"github.com/amenophis1er/gh-setup/internal/templates"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/sync/errgroup"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	addStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	removeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	changeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

// Change represents a single diff entry between desired and actual state.
type Change struct {
	Resource string `json:"resource"`
	Type     string `json:"type"`
	Field    string `json:"field"`
	Old      string `json:"old,omitempty"`
	New      string `json:"new,omitempty"`
	Action   string `json:"action"`
}

// DiffResult holds all collected changes from a diff run.
type DiffResult struct {
	Changes []Change `json:"changes"`
}

// Run compares the config against actual GitHub state using a new authenticated client.
func Run(cfg *config.Config, outputFormat string, concurrency int) error {
	client, err := ghclient.NewClient()
	if err != nil {
		return err
	}
	return RunWith(client, cfg, outputFormat, concurrency)
}

// RunWith compares the config against actual GitHub state using the provided client.
func RunWith(client ghclient.GitHubClient, cfg *config.Config, outputFormat string, concurrency int) error {
	result, err := Compute(client, cfg, concurrency)
	if err != nil {
		return err
	}

	if outputFormat == "json" {
		return RenderJSON(os.Stdout, result)
	}
	RenderText(os.Stdout, result)
	return nil
}

// Compute calculates the diff between config and live GitHub state.
func Compute(client ghclient.GitHubClient, cfg *config.Config, concurrency int) (DiffResult, error) {
	if concurrency < 1 {
		concurrency = 1
	}

	owner := cfg.Account.Name
	isOrg := cfg.Account.Type == "organization"

	repos := cfg.Repos
	if cfg.RepoScope == "all" {
		discovered, err := client.ListRepos(owner, isOrg)
		if err != nil {
			return DiffResult{}, fmt.Errorf("listing repos: %w", err)
		}
		names := make([]string, len(discovered))
		for i, d := range discovered {
			names[i] = d.GetName()
		}
		repos = config.MergeRepoScope(names, cfg.Repos)
	}

	var result DiffResult
	var mu sync.Mutex

	g := new(errgroup.Group)
	g.SetLimit(concurrency)

	for _, repo := range repos {
		repo := repo
		g.Go(func() error {
			var local DiffResult
			diffRepo(client, cfg, owner, repo, &local)
			mu.Lock()
			result.Changes = append(result.Changes, local.Changes...)
			mu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	if cfg.Account.Type == "organization" {
		for _, team := range cfg.Teams {
			diffTeam(client, owner, team, &result)
		}
	}

	return result, nil
}

// RenderJSON writes the diff result as indented JSON.
func RenderJSON(w io.Writer, result DiffResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling JSON: %w", err)
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

// RenderText writes the diff result as styled text.
func RenderText(w io.Writer, result DiffResult) {
	currentResource := ""

	for _, c := range result.Changes {
		if c.Resource != currentResource {
			currentResource = c.Resource
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprintln(w, headerStyle.Render(fmt.Sprintf("  %s %s", c.Type, c.Resource)))
		}

		switch c.Action {
		case "add":
			if c.Field != "" {
				_, _ = fmt.Fprintln(w, addStyle.Render(fmt.Sprintf("    %s: + %s", c.Field, c.New)))
			} else {
				_, _ = fmt.Fprintln(w, addStyle.Render(fmt.Sprintf("    + %s", c.New)))
			}
		case "remove":
			if c.Field != "" {
				_, _ = fmt.Fprintln(w, removeStyle.Render(fmt.Sprintf("    %s: - %s", c.Field, c.Old)))
			} else {
				_, _ = fmt.Fprintln(w, removeStyle.Render(fmt.Sprintf("    - %s", c.Old)))
			}
		case "change":
			_, _ = fmt.Fprintln(w, changeStyle.Render(fmt.Sprintf("    %s:  %s → %s", c.Field, c.Old, c.New)))
		case "ok":
			_, _ = fmt.Fprintln(w, okStyle.Render(fmt.Sprintf("    %s", c.New)))
		case "error":
			_, _ = fmt.Fprintf(w, "    %s: %s\n", c.Field, c.New)
		}
	}
}

func diffRepo(client ghclient.GitHubClient, cfg *config.Config, owner string, repo config.Repo, result *DiffResult) {
	resource := fmt.Sprintf("%s/%s", owner, repo.Name)
	resType := "repo"

	existing, err := client.GetRepo(owner, repo.Name)
	if err != nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: "error", Action: "error", New: fmt.Sprintf("%v", err),
		})
		return
	}
	if existing == nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Action: "add", New: "repo does not exist (will be created)",
		})
		return
	}

	changes := false

	visibility := repo.Visibility
	if visibility == "" {
		visibility = cfg.Defaults.Visibility
	}
	private := visibility == "private"
	if existing.GetPrivate() != private {
		changes = true
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: "visibility", Action: "change",
			Old: boolToVisibility(existing.GetPrivate()), New: visibility,
		})
	}

	if existing.GetDescription() != repo.Description {
		changes = true
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: "description", Action: "change",
			Old: fmt.Sprintf("%q", existing.GetDescription()), New: fmt.Sprintf("%q", repo.Description),
		})
	}

	if repo.Homepage != "" && existing.GetHomepage() != repo.Homepage {
		changes = true
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: "homepage", Action: "change",
			Old: fmt.Sprintf("%q", existing.GetHomepage()), New: fmt.Sprintf("%q", repo.Homepage),
		})
	}

	if existing.GetDeleteBranchOnMerge() != cfg.Defaults.DeleteBranchOnMerge {
		changes = true
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: "delete_branch_on_merge", Action: "change",
			Old: fmt.Sprintf("%v", existing.GetDeleteBranchOnMerge()), New: fmt.Sprintf("%v", cfg.Defaults.DeleteBranchOnMerge),
		})
	}

	if existing.GetAllowAutoMerge() != cfg.Defaults.AllowAutoMerge {
		changes = true
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: "allow_auto_merge", Action: "change",
			Old: fmt.Sprintf("%v", existing.GetAllowAutoMerge()), New: fmt.Sprintf("%v", cfg.Defaults.AllowAutoMerge),
		})
	}

	// Merge strategies and features — only diff when explicitly configured (non-nil)
	diffBoolPtr(existing.GetAllowSquashMerge(), cfg.Defaults.AllowSquashMerge, "allow_squash_merge", resource, resType, result, &changes)
	diffBoolPtr(existing.GetAllowMergeCommit(), cfg.Defaults.AllowMergeCommit, "allow_merge_commit", resource, resType, result, &changes)
	diffBoolPtr(existing.GetAllowRebaseMerge(), cfg.Defaults.AllowRebaseMerge, "allow_rebase_merge", resource, resType, result, &changes)
	diffBoolPtr(existing.GetHasIssues(), cfg.Defaults.HasIssues, "has_issues", resource, resType, result, &changes)
	diffBoolPtr(existing.GetHasWiki(), cfg.Defaults.HasWiki, "has_wiki", resource, resType, result, &changes)
	diffBoolPtr(existing.GetHasDiscussions(), cfg.Defaults.HasDiscussions, "has_discussions", resource, resType, result, &changes)

	// Labels diff
	changesBefore := len(result.Changes)
	diffLabels(client, owner, repo.Name, cfg.Labels, resource, result)

	// Branch protection diff
	branch := cfg.Defaults.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	diffProtection(client, cfg, owner, repo, branch, resource, result)

	// CI workflow
	diffCIWorkflow(client, owner, repo.Name, repo.CI, resource, result)

	// Dependabot config
	diffDependabot(client, owner, repo.Name, repo.CI, cfg.Security.Dependabot, resource, result)

	// CODEOWNERS
	diffCodeowners(client, owner, repo.Name, cfg.Governance.Codeowners, resource, result)

	// Security flags
	var secretScanStatus, advancedSecStatus string
	if sa := existing.GetSecurityAndAnalysis(); sa != nil {
		if sa.SecretScanning != nil {
			secretScanStatus = sa.SecretScanning.GetStatus()
		}
		if sa.AdvancedSecurity != nil {
			advancedSecStatus = sa.AdvancedSecurity.GetStatus()
		}
	}
	diffSecurityFlags(client, owner, repo.Name, cfg.Security, secretScanStatus, advancedSecStatus, resource, result)

	if !changes && len(result.Changes) == changesBefore {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Action: "ok", New: "\u2713 up to date",
		})
	}
}

func diffLabels(client ghclient.GitHubClient, owner, repo string, labels config.Labels, resource string, result *DiffResult) {
	existing, err := client.ListLabels(owner, repo)
	if err != nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "labels error", Action: "error", New: fmt.Sprintf("%v", err),
		})
		return
	}

	existingMap := make(map[string]string)
	for _, l := range existing {
		existingMap[strings.ToLower(l.GetName())] = l.GetColor()
	}

	desiredMap := make(map[string]config.Label)
	for _, l := range labels.Items {
		desiredMap[strings.ToLower(l.Name)] = l
	}

	for _, l := range labels.Items {
		key := strings.ToLower(l.Name)
		if _, ok := existingMap[key]; !ok {
			result.Changes = append(result.Changes, Change{
				Resource: resource, Type: "label", Field: "labels", Action: "add",
				New: fmt.Sprintf("%s (%s)", l.Name, l.Color),
			})
		}
	}

	if labels.ReplaceDefaults {
		for _, l := range existing {
			key := strings.ToLower(l.GetName())
			if _, ok := desiredMap[key]; !ok {
				result.Changes = append(result.Changes, Change{
					Resource: resource, Type: "label", Field: "labels", Action: "remove",
					Old: fmt.Sprintf("%s (%s)", l.GetName(), l.GetColor()),
				})
			}
		}
	}
}

func diffProtection(client ghclient.GitHubClient, cfg *config.Config, owner string, repo config.Repo, branch string, resource string, result *DiffResult) {
	bp := config.ResolveProtection(cfg.Defaults.BranchProtection.Preset)
	if cfg.Defaults.BranchProtection.Preset == "custom" {
		bp = cfg.Defaults.BranchProtection
	}
	if repo.ExtraProtection != nil {
		bp = *repo.ExtraProtection
	}

	existing, err := client.GetBranchProtection(owner, repo.Name, branch)
	if err != nil {
		return
	}

	if existing == nil && bp.Preset != "none" {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "branch_protection", Action: "add",
			New: "not set (will be created)",
		})
		return
	}

	if existing != nil {
		if bp.RequirePR {
			if existing.RequiredPullRequestReviews == nil {
				result.Changes = append(result.Changes, Change{
					Resource: resource, Type: "repo", Field: "branch_protection.require_pr", Action: "change",
					Old: "false", New: "true",
				})
			}
		}
		if existing.AllowForcePushes != nil && existing.AllowForcePushes.Enabled != bp.AllowForcePush {
			result.Changes = append(result.Changes, Change{
				Resource: resource, Type: "repo", Field: "branch_protection.allow_force_push", Action: "change",
				Old: fmt.Sprintf("%v", existing.AllowForcePushes.Enabled), New: fmt.Sprintf("%v", bp.AllowForcePush),
			})
		}
		if existing.AllowDeletions != nil && existing.AllowDeletions.Enabled != bp.AllowDeletions {
			result.Changes = append(result.Changes, Change{
				Resource: resource, Type: "repo", Field: "branch_protection.allow_deletions", Action: "change",
				Old: fmt.Sprintf("%v", existing.AllowDeletions.Enabled), New: fmt.Sprintf("%v", bp.AllowDeletions),
			})
		}
	}
}

func diffTeam(client ghclient.GitHubClient, org string, team config.Team, result *DiffResult) {
	resource := team.Name
	resType := "team"

	existing, err := client.GetTeam(org, team.Name)
	if err != nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: "error", Action: "error", New: fmt.Sprintf("%v", err),
		})
		return
	}
	if existing == nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Action: "add", New: "team does not exist (will be created)",
		})
		return
	}

	members, err := client.ListTeamMembers(org, team.Name)
	if err != nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: "members error", Action: "error", New: fmt.Sprintf("%v", err),
		})
		return
	}

	memberMap := make(map[string]bool)
	for _, m := range members {
		memberMap[m.GetLogin()] = true
	}

	desiredMap := make(map[string]bool)
	for _, m := range team.Members {
		desiredMap[m] = true
	}

	for _, m := range team.Members {
		if !memberMap[m] {
			result.Changes = append(result.Changes, Change{
				Resource: resource, Type: resType, Field: "member", Action: "add", New: m,
			})
		}
	}

	for _, m := range members {
		if !desiredMap[m.GetLogin()] {
			result.Changes = append(result.Changes, Change{
				Resource: resource, Type: resType, Field: "member", Action: "remove", Old: m.GetLogin(),
			})
		}
	}
}

func boolToVisibility(private bool) string {
	if private {
		return "private"
	}
	return "public"
}

// diffBoolPtr adds a change only when desired is explicitly set (non-nil) and differs from actual.
func diffBoolPtr(actual bool, desired *bool, field, resource, resType string, result *DiffResult, changes *bool) {
	if config.BoolPtrDiffers(actual, desired) {
		*changes = true
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: field, Action: "change",
			Old: fmt.Sprintf("%v", actual), New: fmt.Sprintf("%v", *desired),
		})
	}
}

func diffCIWorkflow(client ghclient.GitHubClient, owner, repoName, ciName, resource string, result *DiffResult) {
	if ciName == "" {
		return
	}

	desired, err := templates.CIWorkflow(ciName)
	if err != nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "ci error", Action: "error",
			New: fmt.Sprintf("%v", err),
		})
		return
	}

	actual, _, err := client.GetFileContent(owner, repoName, ".github/workflows/ci.yml")
	if err != nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "ci error", Action: "error",
			New: fmt.Sprintf("%v", err),
		})
		return
	}

	if actual == "" {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "ci_workflow", Action: "add",
			New: fmt.Sprintf("%s (will be created)", ciName),
		})
		return
	}

	if strings.TrimSpace(actual) != strings.TrimSpace(string(desired)) {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "ci_workflow", Action: "change",
			Old: "current", New: fmt.Sprintf("%s template", ciName),
		})
	}
}

func diffDependabot(client ghclient.GitHubClient, owner, repoName, ciName string, enabled bool, resource string, result *DiffResult) {
	if !enabled {
		return
	}

	ecosystem := templates.CIToEcosystem(ciName)
	desired := templates.DependabotConfig(ecosystem)

	actual, _, err := client.GetFileContent(owner, repoName, ".github/dependabot.yml")
	if err != nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "dependabot error", Action: "error",
			New: fmt.Sprintf("%v", err),
		})
		return
	}

	if actual == "" {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "dependabot.yml", Action: "add",
			New: "not present (will be created)",
		})
		return
	}

	if strings.TrimSpace(actual) != strings.TrimSpace(desired) {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "dependabot.yml", Action: "change",
			Old: "current", New: "desired config",
		})
	}
}

func diffCodeowners(client ghclient.GitHubClient, owner, repoName, desired, resource string, result *DiffResult) {
	if desired == "" {
		return
	}

	actual, _, err := client.GetFileContent(owner, repoName, ".github/CODEOWNERS")
	if err != nil {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "codeowners error", Action: "error",
			New: fmt.Sprintf("%v", err),
		})
		return
	}

	if actual == "" {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "CODEOWNERS", Action: "add",
			New: strings.TrimSpace(desired),
		})
		return
	}

	if strings.TrimSpace(actual) != strings.TrimSpace(desired) {
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "CODEOWNERS", Action: "change",
			Old: strings.TrimSpace(actual), New: strings.TrimSpace(desired),
		})
	}
}

func diffSecurityFlags(client ghclient.GitHubClient, owner, repoName string, sec config.Security, secretScanStatus, advancedSecStatus string, resource string, result *DiffResult) {
	if sec.Dependabot {
		alerts, err := client.GetVulnerabilityAlerts(owner, repoName)
		if err != nil {
			result.Changes = append(result.Changes, Change{
				Resource: resource, Type: "repo", Field: "dependabot error", Action: "error",
				New: fmt.Sprintf("%v", err),
			})
		} else if !alerts {
			result.Changes = append(result.Changes, Change{
				Resource: resource, Type: "repo", Field: "dependabot_alerts", Action: "change",
				Old: "disabled", New: "enabled",
			})
		}
	}

	if sec.SecretScanning && secretScanStatus != "enabled" {
		old := secretScanStatus
		if old == "" {
			old = "disabled"
		}
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "secret_scanning", Action: "change",
			Old: old, New: "enabled",
		})
	}

	if sec.CodeScanning && advancedSecStatus != "enabled" {
		old := advancedSecStatus
		if old == "" {
			old = "disabled"
		}
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: "repo", Field: "code_scanning", Action: "change",
			Old: old, New: "enabled",
		})
	}
}
