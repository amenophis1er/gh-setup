# Contributing to gh-setup

Thanks for your interest in contributing!

## Getting Started

```bash
git clone https://github.com/amenophis1er/gh-setup.git
cd gh-setup
make all   # vet + test + build
```

## Development Workflow

1. Fork the repo and create a feature branch
2. Make your changes
3. Run `make vet` and `make test`
4. Open a pull request against `main`

## Project Layout

- `cmd/` — Cobra commands (root, init, apply, diff, import)
- `internal/config/` — YAML config structs, validation, presets
- `internal/wizard/` — Interactive init wizard (charmbracelet/huh)
- `internal/github/` — GitHub API modules
- `internal/templates/` — Embedded CI and governance templates
- `internal/apply/` — Idempotent apply logic
- `internal/diff/` — Config vs live state comparison
- `internal/importer/` — Reverse-engineer config from GitHub

## Adding a CI Template

1. Create `internal/templates/workflows/<name>.yml`
2. Add the name to `CITemplateNames()` in `internal/templates/ci.go`
3. Add the ecosystem mapping in `internal/templates/dependabot.go`
4. Update the CI templates table in `README.md`

## Code Style

- Run `go vet` before committing
- Keep functions focused and small
- Follow existing patterns in the codebase

## Reporting Issues

Open an issue at https://github.com/amenophis1er/gh-setup/issues with:
- What you expected
- What happened instead
- Steps to reproduce
