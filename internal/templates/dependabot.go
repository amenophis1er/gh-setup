package templates

// DependabotConfig returns a dependabot.yml for the given ecosystem.
// If ecosystem is empty, it defaults to a generic config with just github-actions.
func DependabotConfig(ecosystem string) string {
	base := `version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
`
	if ecosystem == "" {
		return base
	}

	ecosystemBlock := `  - package-ecosystem: "` + ecosystem + `"
    directory: "/"
    schedule:
      interval: "weekly"
`
	return base + ecosystemBlock
}

// CIToEcosystem maps a CI template name to a Dependabot package ecosystem.
func CIToEcosystem(ci string) string {
	switch ci {
	case "go":
		return "gomod"
	case "rust":
		return "cargo"
	case "node":
		return "npm"
	case "python":
		return "pip"
	default:
		return ""
	}
}
