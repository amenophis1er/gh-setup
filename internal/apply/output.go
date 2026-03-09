package apply

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	skipStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	dryRunStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
)

func logSuccess(action, resource string) {
	fmt.Println(successStyle.Render(fmt.Sprintf("  ✓ %s %s", action, resource)))
}

func logSkip(resource string) {
	fmt.Println(skipStyle.Render(fmt.Sprintf("  - %s (up to date)", resource)))
}

func logDryRun(action, resource string) {
	fmt.Println(dryRunStyle.Render(fmt.Sprintf("  ~ would %s %s", action, resource)))
}

func logError(action, resource string, err error) {
	fmt.Println(errorStyle.Render(fmt.Sprintf("  ✗ %s %s: %s", action, resource, err)))
}

func logHeader(section string) {
	fmt.Println()
	fmt.Println(headerStyle.Render(fmt.Sprintf("── %s ──", section)))
}
