package apply

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	skipStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	dryRunStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
)

// stats tracks action counts across the entire apply run.
type stats struct {
	mu      sync.Mutex
	created int
	updated int
	skipped int
	errors  int
}

var current stats

func resetStats() {
	current = stats{}
}

func logSuccess(action, resource string) {
	fmt.Println(successStyle.Render(fmt.Sprintf("  ✓ %s %s", action, resource)))
	current.mu.Lock()
	if strings.HasPrefix(action, "created") || strings.HasPrefix(action, "create") {
		current.created++
	} else {
		current.updated++
	}
	current.mu.Unlock()
}

func logSkip(resource string) {
	fmt.Println(skipStyle.Render(fmt.Sprintf("  - %s (up to date)", resource)))
	current.mu.Lock()
	current.skipped++
	current.mu.Unlock()
}

func logDryRun(action, resource string) {
	fmt.Println(dryRunStyle.Render(fmt.Sprintf("  ~ would %s %s", action, resource)))
	current.mu.Lock()
	if strings.HasPrefix(action, "create") {
		current.created++
	} else {
		current.updated++
	}
	current.mu.Unlock()
}

func logError(action, resource string, err error) {
	fmt.Println(errorStyle.Render(fmt.Sprintf("  ✗ %s %s: %s", action, resource, err)))
	current.mu.Lock()
	current.errors++
	current.mu.Unlock()
}

func logHeader(section string) {
	fmt.Println()
	fmt.Println(headerStyle.Render(fmt.Sprintf("── %s ──", section)))
}

func printSummary(dryRun bool) {
	s := &current
	total := s.created + s.updated + s.skipped + s.errors
	if total == 0 {
		return
	}

	fmt.Println()
	fmt.Println(headerStyle.Render("── Summary ──"))

	var parts []string
	if s.created > 0 {
		label := "created"
		if dryRun {
			label = "to create"
		}
		parts = append(parts, successStyle.Render(fmt.Sprintf("%d %s", s.created, label)))
	}
	if s.updated > 0 {
		label := "updated"
		if dryRun {
			label = "to update"
		}
		parts = append(parts, successStyle.Render(fmt.Sprintf("%d %s", s.updated, label)))
	}
	if s.skipped > 0 {
		parts = append(parts, skipStyle.Render(fmt.Sprintf("%d skipped", s.skipped)))
	}
	if s.errors > 0 {
		parts = append(parts, errorStyle.Render(fmt.Sprintf("%d error(s)", s.errors)))
	}

	fmt.Println("  " + strings.Join(parts, "  "))
}
