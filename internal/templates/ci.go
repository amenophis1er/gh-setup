package templates

import "embed"

//go:embed workflows/*.yml
var workflowFS embed.FS

// CIWorkflow returns the CI workflow template content for the given name.
// Supported names: go, rust, node, python.
func CIWorkflow(name string) ([]byte, error) {
	return workflowFS.ReadFile("workflows/" + name + ".yml")
}

// CITemplateNames returns the list of available CI template names.
func CITemplateNames() []string {
	return []string{"go", "rust", "node", "python"}
}
