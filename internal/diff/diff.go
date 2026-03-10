package diff

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/amenophis1er/gh-setup/internal/config"
	ghclient "github.com/amenophis1er/gh-setup/internal/github"
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
	if concurrency < 1 {
		concurrency = 1
	}

	owner := cfg.Account.Name
	isOrg := cfg.Account.Type == "organization"

	repos := cfg.Repos
	if cfg.RepoScope == "all" {
		discovered, err := client.ListRepos(owner, isOrg)
		if err != nil {
			return fmt.Errorf("listing repos: %w", err)
		}

		overrides := make(map[string]config.Repo)
		for _, r := range cfg.Repos {
			overrides[r.Name] = r
		}

		merged := make([]config.Repo, 0, len(discovered))
		seen := make(map[string]bool)
		for _, d := range discovered {
			name := d.GetName()
			seen[name] = true
			if override, ok := overrides[name]; ok {
				merged = append(merged, override)
			} else {
				merged = append(merged, config.Repo{Name: name})
			}
		}
		for _, r := range cfg.Repos {
			if !seen[r.Name] {
				merged = append(merged, r)
			}
		}
		repos = merged
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

	if outputFormat == "json" {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshalling JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		printTextOutput(result, owner, cfg)
	}

	return nil
}

func printTextOutput(result DiffResult, owner string, cfg *config.Config) {
	// Group changes by resource for display
	currentResource := ""
	resourceHasChanges := make(map[string]bool)

	// Pre-scan to determine which resources have real changes (not just "ok")
	for _, c := range result.Changes {
		if c.Action != "ok" {
			resourceHasChanges[c.Resource] = true
		}
	}

	for _, c := range result.Changes {
		if c.Resource != currentResource {
			currentResource = c.Resource
			fmt.Println()
			fmt.Println(headerStyle.Render(fmt.Sprintf("  %s %s", c.Type, c.Resource)))
		}

		switch c.Action {
		case "add":
			if c.Field != "" {
				fmt.Println(addStyle.Render(fmt.Sprintf("    %s: + %s", c.Field, c.New)))
			} else {
				fmt.Println(addStyle.Render(fmt.Sprintf("    + %s", c.New)))
			}
		case "remove":
			if c.Field != "" {
				fmt.Println(removeStyle.Render(fmt.Sprintf("    %s: - %s", c.Field, c.Old)))
			} else {
				fmt.Println(removeStyle.Render(fmt.Sprintf("    - %s", c.Old)))
			}
		case "change":
			fmt.Println(changeStyle.Render(fmt.Sprintf("    %s:  %s → %s", c.Field, c.Old, c.New)))
		case "ok":
			fmt.Println(okStyle.Render(fmt.Sprintf("    %s", c.New)))
		case "error":
			fmt.Printf("    %s: %s\n", c.Field, c.New)
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
	diffLabels(client, owner, repo.Name, cfg.Labels, resource, result)

	// Branch protection diff
	branch := cfg.Defaults.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	diffProtection(client, cfg, owner, repo, branch, resource, result)

	if !changes {
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
	if desired != nil && actual != *desired {
		*changes = true
		result.Changes = append(result.Changes, Change{
			Resource: resource, Type: resType, Field: field, Action: "change",
			Old: fmt.Sprintf("%v", actual), New: fmt.Sprintf("%v", *desired),
		})
	}
}
