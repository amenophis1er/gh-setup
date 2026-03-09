package diff

import (
	"fmt"
	"strings"

	"github.com/amenophis1er/gh-setup/internal/config"
	ghclient "github.com/amenophis1er/gh-setup/internal/github"
	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	addStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	removeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	changeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

// Run compares the config against actual GitHub state and prints differences.
func Run(cfg *config.Config) error {
	client, err := ghclient.NewClient()
	if err != nil {
		return err
	}

	owner := cfg.Account.Name

	for _, repo := range cfg.Repos {
		diffRepo(client, cfg, owner, repo)
	}

	if cfg.Account.Type == "organization" {
		for _, team := range cfg.Teams {
			diffTeam(client, owner, team)
		}
	}

	return nil
}

func diffRepo(client *ghclient.Client, cfg *config.Config, owner string, repo config.Repo) {
	fmt.Println()
	fmt.Println(headerStyle.Render(fmt.Sprintf("  repo %s/%s", owner, repo.Name)))

	existing, err := client.GetRepo(owner, repo.Name)
	if err != nil {
		fmt.Printf("    error: %v\n", err)
		return
	}
	if existing == nil {
		fmt.Println(addStyle.Render("    + repo does not exist (will be created)"))
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
		fmt.Println(changeStyle.Render(fmt.Sprintf("    visibility:  %s → %s",
			boolToVisibility(existing.GetPrivate()), visibility)))
	}

	if existing.GetDescription() != repo.Description {
		changes = true
		fmt.Println(changeStyle.Render(fmt.Sprintf("    description:  %q → %q",
			existing.GetDescription(), repo.Description)))
	}

	if existing.GetDeleteBranchOnMerge() != cfg.Defaults.DeleteBranchOnMerge {
		changes = true
		fmt.Println(changeStyle.Render(fmt.Sprintf("    delete_branch_on_merge:  %v → %v",
			existing.GetDeleteBranchOnMerge(), cfg.Defaults.DeleteBranchOnMerge)))
	}

	// Labels diff
	diffLabels(client, owner, repo.Name, cfg.Labels)

	// Branch protection diff
	branch := cfg.Defaults.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	diffProtection(client, cfg, owner, repo, branch)

	if !changes {
		fmt.Println(okStyle.Render("    ✓ up to date"))
	}
}

func diffLabels(client *ghclient.Client, owner, repo string, labels config.Labels) {
	existing, err := client.ListLabels(owner, repo)
	if err != nil {
		fmt.Printf("    labels error: %v\n", err)
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
			fmt.Println(addStyle.Render(fmt.Sprintf("    labels: + %s (%s)", l.Name, l.Color)))
		}
	}

	if labels.ReplaceDefaults {
		for _, l := range existing {
			key := strings.ToLower(l.GetName())
			if _, ok := desiredMap[key]; !ok {
				fmt.Println(removeStyle.Render(fmt.Sprintf("    labels: - %s (%s)", l.GetName(), l.GetColor())))
			}
		}
	}
}

func diffProtection(client *ghclient.Client, cfg *config.Config, owner string, repo config.Repo, branch string) {
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
		fmt.Println(addStyle.Render("    branch_protection: not set (will be created)"))
		return
	}

	if existing != nil {
		if bp.RequirePR {
			if existing.RequiredPullRequestReviews == nil {
				fmt.Println(changeStyle.Render("    branch_protection.require_pr:  false → true"))
			}
		}
		if existing.AllowForcePushes != nil && existing.AllowForcePushes.Enabled != bp.AllowForcePush {
			fmt.Println(changeStyle.Render(fmt.Sprintf("    branch_protection.allow_force_push:  %v → %v",
				existing.AllowForcePushes.Enabled, bp.AllowForcePush)))
		}
		if existing.AllowDeletions != nil && existing.AllowDeletions.Enabled != bp.AllowDeletions {
			fmt.Println(changeStyle.Render(fmt.Sprintf("    branch_protection.allow_deletions:  %v → %v",
				existing.AllowDeletions.Enabled, bp.AllowDeletions)))
		}
	}
}

func diffTeam(client *ghclient.Client, org string, team config.Team) {
	fmt.Println()
	fmt.Println(headerStyle.Render(fmt.Sprintf("  team %s", team.Name)))

	existing, err := client.GetTeam(org, team.Name)
	if err != nil {
		fmt.Printf("    error: %v\n", err)
		return
	}
	if existing == nil {
		fmt.Println(addStyle.Render("    + team does not exist (will be created)"))
		return
	}

	members, err := client.ListTeamMembers(org, team.Name)
	if err != nil {
		fmt.Printf("    members error: %v\n", err)
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
			fmt.Println(addStyle.Render(fmt.Sprintf("    + member: %s", m)))
		}
	}

	for _, m := range members {
		if !desiredMap[m.GetLogin()] {
			fmt.Println(removeStyle.Render(fmt.Sprintf("    - member: %s", m.GetLogin())))
		}
	}
}

func boolToVisibility(private bool) string {
	if private {
		return "private"
	}
	return "public"
}
