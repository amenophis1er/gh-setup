package wizard

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/templates"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// Run executes the interactive wizard and returns the generated config.
func Run() (*config.Config, error) {
	cfg := &config.Config{}

	fmt.Println(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Render("Welcome to gh-setup!"))
	fmt.Println()

	// Account
	if err := runAccountForm(cfg); err != nil {
		return nil, err
	}

	// Defaults
	if err := runDefaultsForm(cfg); err != nil {
		return nil, err
	}

	// Custom branch protection (if preset is custom)
	if cfg.Defaults.BranchProtection.Preset == "custom" {
		if err := runCustomProtectionForm(cfg); err != nil {
			return nil, err
		}
	}

	// Labels
	if err := runLabelsForm(cfg); err != nil {
		return nil, err
	}

	// Repos
	if err := runReposForm(cfg); err != nil {
		return nil, err
	}

	// Teams (org only)
	if cfg.Account.Type == "organization" {
		if err := runTeamsForm(cfg); err != nil {
			return nil, err
		}
	}

	// Governance
	if err := runGovernanceForm(cfg); err != nil {
		return nil, err
	}

	// Security
	if err := runSecurityForm(cfg); err != nil {
		return nil, err
	}

	// Secrets
	if err := runSecretsForm(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func runAccountForm(cfg *config.Config) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Account type").
				Options(
					huh.NewOption("Individual", "individual"),
					huh.NewOption("Organization", "organization"),
				).
				Value(&cfg.Account.Type),

			huh.NewInput().
				Title("GitHub username or org").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					if !validNameRe.MatchString(s) {
						return fmt.Errorf("invalid characters in name")
					}
					return nil
				}).
				Value(&cfg.Account.Name),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
}

func runDefaultsForm(cfg *config.Config) error {
	cfg.Defaults.DefaultBranch = "main"
	cfg.Defaults.DeleteBranchOnMerge = true

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Default repo visibility").
				Options(
					huh.NewOption("Public", "public"),
					huh.NewOption("Private", "private"),
				).
				Value(&cfg.Defaults.Visibility),

			huh.NewInput().
				Title("Default branch name").
				Value(&cfg.Defaults.DefaultBranch),

			huh.NewConfirm().
				Title("Delete branch on merge?").
				Value(&cfg.Defaults.DeleteBranchOnMerge),

			huh.NewSelect[string]().
				Title("Branch protection preset").
				Description("Basic: block force push/deletion\nStandard: require PR (1 approval) + block force push/deletion\nStrict: require PR + CI checks + up-to-date + block force push/deletion").
				Options(
					huh.NewOption("None", "none"),
					huh.NewOption("Basic", "basic"),
					huh.NewOption("Standard", "standard"),
					huh.NewOption("Strict", "strict"),
					huh.NewOption("Custom", "custom"),
				).
				Value(&cfg.Defaults.BranchProtection.Preset),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
}

func runCustomProtectionForm(cfg *config.Config) error {
	bp := &cfg.Defaults.BranchProtection
	bp.RequirePR = true
	bp.RequiredApprovals = 1

	var approvals int
	var statusChecksStr string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Require pull request?").
				Value(&bp.RequirePR),

			huh.NewSelect[int]().
				Title("Required approvals").
				Options(
					huh.NewOption("0", 0),
					huh.NewOption("1", 1),
					huh.NewOption("2", 2),
					huh.NewOption("3", 3),
				).
				Value(&approvals),

			huh.NewConfirm().
				Title("Dismiss stale reviews?").
				Value(&bp.DismissStaleReviews),

			huh.NewConfirm().
				Title("Require status checks?").
				Value(&bp.RequireStatusChecks),

			huh.NewInput().
				Title("Status checks (comma-separated, e.g. ci,lint)").
				Value(&statusChecksStr),

			huh.NewConfirm().
				Title("Require branches to be up to date?").
				Value(&bp.RequireUpToDate),

			huh.NewConfirm().
				Title("Enforce for admins?").
				Value(&bp.EnforceAdmins),

			huh.NewConfirm().
				Title("Allow force push?").
				Value(&bp.AllowForcePush),

			huh.NewConfirm().
				Title("Allow deletions?").
				Value(&bp.AllowDeletions),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}

	bp.RequiredApprovals = approvals

	if statusChecksStr != "" {
		for _, s := range strings.Split(statusChecksStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				bp.StatusChecks = append(bp.StatusChecks, s)
			}
		}
	}

	return nil
}

func runLabelsForm(cfg *config.Config) error {
	cfg.Labels.ReplaceDefaults = true

	var addCustom bool
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Replace GitHub default labels with custom set?").
				Value(&cfg.Labels.ReplaceDefaults),

			huh.NewConfirm().
				Title("Add custom labels?").
				Value(&addCustom),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}

	// Set sensible default labels
	cfg.Labels.Items = []config.Label{
		{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
		{Name: "enhancement", Color: "a2eeef", Description: "New feature or request"},
		{Name: "breaking", Color: "e11d48", Description: "Breaking change"},
		{Name: "docs", Color: "0075ca", Description: "Documentation"},
		{Name: "ci", Color: "e4e669", Description: "CI/CD changes"},
		{Name: "chore", Color: "cfd3d7", Description: "Maintenance"},
	}

	if addCustom {
		for {
			var input string
			var addMore bool

			err := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Label (name:color:description, empty to stop)").
						Value(&input),
				),
			).WithTheme(huh.ThemeCatppuccin()).Run()
			if err != nil {
				return err
			}

			if input == "" {
				break
			}

			parts := strings.SplitN(input, ":", 3)
			label := config.Label{Name: parts[0]}
			if len(parts) > 1 {
				label.Color = parts[1]
			}
			if len(parts) > 2 {
				label.Description = parts[2]
			}
			cfg.Labels.Items = append(cfg.Labels.Items, label)

			err = huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Add another label?").
						Value(&addMore),
				),
			).WithTheme(huh.ThemeCatppuccin()).Run()
			if err != nil {
				return err
			}
			if !addMore {
				break
			}
		}
	}

	return nil
}

func runReposForm(cfg *config.Config) error {
	for {
		var addRepo bool
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Add a repository?").
					Value(&addRepo),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return err
		}
		if !addRepo {
			break
		}

		repo := config.Repo{}
		var topicsStr string
		ciOptions := []huh.Option[string]{huh.NewOption("None", "")}
		for _, name := range templates.CITemplateNames() {
			ciOptions = append(ciOptions, huh.NewOption(cases.Title(language.English).String(name), name))
		}

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Repo name").
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("repo name is required")
						}
						if !validNameRe.MatchString(s) {
							return fmt.Errorf("invalid characters in repo name")
						}
						return nil
					}).
					Value(&repo.Name),

				huh.NewInput().
					Title("Description").
					Value(&repo.Description),

				huh.NewSelect[string]().
					Title("Visibility").
					Description("Leave as default to inherit from defaults").
					Options(
						huh.NewOption("Default ("+cfg.Defaults.Visibility+")", ""),
						huh.NewOption("Public", "public"),
						huh.NewOption("Private", "private"),
					).
					Value(&repo.Visibility),

				huh.NewInput().
					Title("Topics (comma-separated)").
					Value(&topicsStr),

				huh.NewInput().
					Title("Homepage URL (optional)").
					Value(&repo.Homepage),

				huh.NewSelect[string]().
					Title("CI template").
					Options(ciOptions...).
					Value(&repo.CI),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return err
		}

		if topicsStr != "" {
			for _, t := range strings.Split(topicsStr, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					repo.Topics = append(repo.Topics, t)
				}
			}
		}

		cfg.Repos = append(cfg.Repos, repo)
	}
	return nil
}

func runTeamsForm(cfg *config.Config) error {
	for {
		var addTeam bool
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Add a team?").
					Value(&addTeam),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return err
		}
		if !addTeam {
			break
		}

		team := config.Team{}
		var membersStr string

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Team name").
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("team name is required")
						}
						return nil
					}).
					Value(&team.Name),

				huh.NewInput().
					Title("Description").
					Value(&team.Description),

				huh.NewSelect[string]().
					Title("Permission").
					Options(
						huh.NewOption("Read", "read"),
						huh.NewOption("Write", "write"),
						huh.NewOption("Admin", "admin"),
					).
					Value(&team.Permission),

				huh.NewInput().
					Title("Members (comma-separated)").
					Value(&membersStr),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return err
		}

		if membersStr != "" {
			for _, m := range strings.Split(membersStr, ",") {
				m = strings.TrimSpace(m)
				if m != "" {
					team.Members = append(team.Members, m)
				}
			}
		}

		cfg.Teams = append(cfg.Teams, team)
	}
	return nil
}

func runGovernanceForm(cfg *config.Config) error {
	cfg.Governance.Contributing = true
	cfg.Governance.CodeOfConduct = true
	cfg.Governance.SecurityPolicy = true

	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Generate CONTRIBUTING.md?").
				Value(&cfg.Governance.Contributing),

			huh.NewConfirm().
				Title("Add Code of Conduct (Contributor Covenant)?").
				Value(&cfg.Governance.CodeOfConduct),

			huh.NewConfirm().
				Title("Add SECURITY.md?").
				Value(&cfg.Governance.SecurityPolicy),

			huh.NewInput().
				Title("CODEOWNERS pattern (e.g. * @org/team)").
				Value(&cfg.Governance.Codeowners),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
}

func runSecurityForm(cfg *config.Config) error {
	cfg.Security.Dependabot = true
	cfg.Security.SecretScanning = true

	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Enable Dependabot?").
				Value(&cfg.Security.Dependabot),

			huh.NewConfirm().
				Title("Enable secret scanning?").
				Value(&cfg.Security.SecretScanning),

			huh.NewConfirm().
				Title("Enable code scanning?").
				Value(&cfg.Security.CodeScanning),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
}

func runSecretsForm(cfg *config.Config) error {
	var addSecrets bool
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add org/repo secrets?").
				Affirmative("Yes").
				Negative("No").
				Value(&addSecrets),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil || !addSecrets {
		return err
	}

	for {
		secret := config.Secret{}

		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Secret name").
					Value(&secret.Name),

				huh.NewSelect[string]().
					Title("Scope").
					Options(
						huh.NewOption("Org", "org"),
						huh.NewOption("Repo", "repo"),
					).
					Value(&secret.Scope),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return err
		}

		if secret.Name == "" {
			break
		}

		cfg.Secrets = append(cfg.Secrets, secret)

		var addMore bool
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Add another secret?").
					Value(&addMore),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return err
		}
		if !addMore {
			break
		}
	}

	return nil
}

