# gh-setup

A CLI tool that declaratively configures GitHub accounts, repositories, branch protection, teams, labels, CI workflows, governance files, and security settings from a single YAML config file.

Installable as a standalone binary or as a [`gh` CLI](https://cli.github.com/) extension.

## Features

- **Interactive wizard** — generates a complete `gh-setup.yaml` through guided prompts
- **Idempotent apply** — only mutates what has drifted, safe to run repeatedly
- **Dry-run mode** — preview every change before it happens
- **Diff** — compare your config against live GitHub state
- **Branch protection presets** — none, basic, standard, strict, or fully custom
- **CI workflow templates** — Go, Rust, Node.js, Python (embedded, zero external deps)
- **Governance files** — CONTRIBUTING.md, Code of Conduct, SECURITY.md, CODEOWNERS
- **Security** — Dependabot, secret scanning, code scanning, dependabot.yml generation
- **Secrets management** — org and repo-level secrets (values prompted securely at apply time)
- **Config validation** — catches errors before any API calls are made

## Installation

### Homebrew

```bash
brew tap amenophis1er/tap
brew install gh-setup
```

### As a `gh` CLI extension (recommended)

```bash
gh extension install amenophis1er/gh-setup
```

Then use it as `gh setup <command>`.

### From source

```bash
go install github.com/amenophis1er/gh-setup@latest
```

### Build locally

```bash
git clone https://github.com/amenophis1er/gh-setup.git
cd gh-setup
make build
```

## Quick Start

```bash
# 1. Generate a config interactively
gh setup init

# 2. Review the generated file
cat gh-setup.yaml

# 3. Preview what would change
gh setup apply --dry-run

# 4. Apply for real
gh setup apply
```

## Authentication

`gh-setup` reads your GitHub token from environment variables:

```bash
export GITHUB_TOKEN="ghp_..."
# or
export GH_TOKEN="ghp_..."
```

If you use the `gh` CLI, you likely already have `GH_TOKEN` set. The token needs the following scopes:

| Scope | Required for |
|-------|-------------|
| `repo` | Repository creation, settings, branch protection, file commits |
| `admin:org` | Organization settings, team management (org accounts only) |
| `workflow` | CI workflow file commits |

## Commands

### `gh setup init`

Interactive wizard that walks you through every configuration option and writes `gh-setup.yaml`.

```bash
gh setup init
gh setup init -c custom-config.yaml
```

### `gh setup apply`

Reads the config and applies it to GitHub. Each resource is fetched, compared, and only mutated if different.

```bash
gh setup apply                # apply all changes
gh setup apply --dry-run      # preview changes without mutating
gh setup apply -i             # confirm each change interactively
gh setup apply -c other.yaml  # use a different config file
```

### `gh setup diff`

Compares your config file against the actual GitHub state and prints the differences.

```bash
gh setup diff
```

Example output:

```
  repo x-phone/xphone-rust
    visibility:  private → public
    branch_protection.require_pr:  false → true
    labels: + breaking (e11d48)
    labels: - wontfix (ffffff)

  repo x-phone/xphone-go
    ✓ up to date

  team core
    + member: new-contributor
```

### `gh setup version`

```bash
gh setup version
```

## Config Reference

The full config file with all available options:

```yaml
# gh-setup.yaml

account:
  type: organization          # individual | organization
  name: my-org

defaults:
  visibility: public          # public | private
  default_branch: main
  delete_branch_on_merge: true
  branch_protection:
    preset: standard           # none | basic | standard | strict | custom
    # Custom overrides (only when preset: custom):
    # require_pr: true
    # required_approvals: 1
    # dismiss_stale_reviews: false
    # require_status_checks: false
    # status_checks: []        # e.g. ["ci", "lint"]
    # require_up_to_date: false
    # enforce_admins: false
    # allow_force_push: false
    # allow_deletions: false

labels:
  replace_defaults: true       # remove GitHub's default labels first
  items:
    - { name: "bug",         color: "d73a4a",  description: "Something isn't working" }
    - { name: "enhancement", color: "a2eeef",  description: "New feature or request" }
    - { name: "breaking",    color: "e11d48",  description: "Breaking change" }
    - { name: "docs",        color: "0075ca",  description: "Documentation" }
    - { name: "ci",          color: "e4e669",  description: "CI/CD changes" }
    - { name: "chore",       color: "cfd3d7",  description: "Maintenance" }

repos:
  - name: my-api
    description: "REST API service"
    topics: ["api", "rest", "go"]
    visibility: private        # overrides default
    homepage: "https://example.com"
    ci: go                     # go | rust | node | python
    extra_protection: {}       # repo-specific protection overrides
  - name: my-frontend
    description: "Web frontend"
    topics: ["frontend", "react"]
    ci: node

teams:                         # organization only
  - name: core
    description: "Core maintainers"
    permission: admin          # read | write | admin
    members: ["user1", "user2"]
  - name: contributors
    description: "External contributors"
    permission: write
    members: []

governance:
  contributing: true           # generate CONTRIBUTING.md
  code_of_conduct: true        # Contributor Covenant
  security_policy: true        # SECURITY.md
  codeowners: |                # .github/CODEOWNERS
    * @my-org/core

security:
  dependabot: true             # enable alerts + generate dependabot.yml
  secret_scanning: true
  code_scanning: false         # requires GitHub Advanced Security on private repos

secrets:                       # names only — values prompted at apply time
  - name: DEPLOY_TOKEN
    scope: org                 # org | repo
  - name: NPM_TOKEN
    scope: repo
```

## Branch Protection Presets

| Rule | None | Basic | Standard | Strict |
|------|------|-------|----------|--------|
| Require PR | | | 1 approval | 1 approval |
| Require status checks | | | | yes |
| Require up-to-date | | | | yes |
| Block force push | | yes | yes | yes |
| Block deletion | | yes | yes | yes |

Choose `custom` to configure each rule individually in the wizard or YAML.

## CI Workflow Templates

Built-in templates embedded in the binary:

| Template | Steps |
|----------|-------|
| **go** | `go vet`, `golangci-lint`, `go test ./...` |
| **rust** | `cargo fmt --check`, `cargo clippy -- -D warnings`, `cargo test` |
| **node** | `npm ci`, `npm run lint`, `npm test` |
| **python** | `ruff check`, `mypy`, `pytest` |

Templates are written to `.github/workflows/ci.yml` in each repository.

## Apply Behavior

Each resource follows the same pattern:

1. **Fetch** current state from the GitHub API
2. **Compare** with the desired config
3. **Skip** if already matching (idempotent)
4. **Mutate** if different (create or update)
5. **Report** the action taken

In `--dry-run` mode, step 4 is replaced with a preview log.
In `-i` (interactive) mode, step 4 requires confirmation.

## Config Validation

Before any API calls, the config is validated for:

- Required fields (`account.name`, repo names, team names, secret names)
- Valid enum values (account type, visibility, preset, permissions, secret scope)
- Character restrictions (no spaces or special characters in names)
- Duplicate detection (repo names, team names)
- Logical checks (teams require organization account type)

## Project Structure

```
gh-setup/
├── main.go                     # entry point
├── cmd/
│   ├── root.go                 # root command, --config flag, version
│   ├── init.go                 # init wizard command
│   ├── apply.go                # apply command (--dry-run, -i)
│   └── diff.go                 # diff command
├── internal/
│   ├── config/config.go        # YAML structs, Load/Save, Validate, presets
│   ├── wizard/wizard.go        # interactive wizard (charmbracelet/huh)
│   ├── github/
│   │   ├── client.go           # authenticated GitHub client
│   │   ├── org.go              # organization settings
│   │   ├── repo.go             # repo CRUD, topics, file content
│   │   ├── protection.go       # branch protection rules
│   │   ├── labels.go           # label management
│   │   ├── teams.go            # team and membership management
│   │   ├── security.go         # Dependabot, secret/code scanning
│   │   └── secrets.go          # encrypted secrets (NaCl box)
│   ├── templates/
│   │   ├── ci.go               # embedded CI workflow loader
│   │   ├── governance.go       # CONTRIBUTING, CoC, SECURITY templates
│   │   ├── dependabot.go       # dependabot.yml generation
│   │   └── workflows/          # CI YAML templates (go, rust, node, python)
│   ├── apply/
│   │   ├── apply.go            # idempotent apply logic
│   │   └── output.go           # styled terminal output
│   └── diff/diff.go            # config vs live state comparison
├── Makefile
├── .goreleaser.yml
└── .github/workflows/
    ├── ci.yml                  # CI pipeline
    └── release.yml             # goreleaser on tag push
```

## Development

```bash
make vet      # run go vet
make test     # run tests
make build    # build binary
make all      # vet + test + build
make clean    # remove binary
```

## License

MIT
