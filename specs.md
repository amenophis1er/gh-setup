# gh-setup — GitHub Account & Repository Setup Tool

A Go CLI that interactively configures GitHub individuals/organizations, repositories, branch protection, teams, labels, CI, and governance — then persists everything as a YAML config for repeatable, idempotent apply. Distributable as a `gh` CLI extension (`gh setup`).

## Commands

| Command | Description |
|---------|-------------|
| `gh-setup init` | Interactive wizard — generates `gh-setup.yaml` |
| `gh-setup apply` | Apply config from `gh-setup.yaml` (auto, idempotent) |
| `gh-setup apply -i` | Apply with confirmation prompt at each step |
| `gh-setup apply --dry-run` | Show what would change without mutating |
| `gh-setup diff` | Compare config vs actual GitHub state |

## Config Format

```yaml
# gh-setup.yaml
account:
  type: organization  # individual | organization
  name: x-phone

defaults:
  visibility: public            # public | private
  default_branch: main
  delete_branch_on_merge: true
  branch_protection:
    preset: standard            # none | basic | standard | strict | custom
    # custom overrides (only when preset: custom):
    require_pr: true
    required_approvals: 1
    dismiss_stale_reviews: false
    require_status_checks: false
    status_checks: []           # e.g. ["ci", "lint"]
    require_up_to_date: false
    enforce_admins: false
    allow_force_push: false
    allow_deletions: false

labels:
  replace_defaults: true        # remove GitHub's default labels first
  items:
    - { name: "bug",         color: "d73a4a",  description: "Something isn't working" }
    - { name: "enhancement", color: "a2eeef",  description: "New feature or request" }
    - { name: "breaking",    color: "e11d48",  description: "Breaking change" }
    - { name: "docs",        color: "0075ca",  description: "Documentation" }
    - { name: "ci",          color: "e4e669",  description: "CI/CD changes" }
    - { name: "chore",       color: "cfd3d7",  description: "Maintenance" }

repos:
  - name: xphone-rust
    description: "SIP telephony library for Rust"
    topics: ["sip", "voip", "telephony", "rust"]
    visibility: public          # overrides default
    homepage: "https://crates.io/crates/xphone"
    ci: rust                    # workflow template name
    extra_protection: {}        # repo-specific overrides
  - name: xphone-go
    description: "SIP telephony library for Go"
    topics: ["sip", "voip", "telephony", "go"]
    ci: go

teams:                          # organization only
  - name: core
    description: "Core maintainers"
    permission: admin           # read | write | admin
    members: ["amenophis1er"]
  - name: contributors
    description: "External contributors"
    permission: write
    members: []

governance:
  contributing: true            # generate CONTRIBUTING.md template
  code_of_conduct: true         # Contributor Covenant
  security_policy: true         # SECURITY.md template
  codeowners: |
    * @x-phone/core

security:
  dependabot: true
  secret_scanning: true
  code_scanning: false          # requires GitHub Advanced Security on private repos

secrets: []                     # names only — values prompted at apply time
  # - name: CRATES_IO_TOKEN
  #   scope: org                # org | repo
```

## Interactive Flow (`init`)

```
Welcome to gh-setup!

? Account type: [Individual / Organization]
? GitHub username or org: x-phone
? Create new org or configure existing? [Existing]

── Defaults ──────────────────────────────────
? Default repo visibility: [Public / Private]
? Default branch name: (main)
? Delete branch on merge? [Y/n]
? Branch protection preset: [None / Basic / Standard / Strict / Custom]
  Basic   — block force push + deletion
  Standard — require PR (1 approval) + block force push + deletion
  Strict   — require PR + CI checks + up-to-date + block force push + deletion
  Custom   — configure each rule individually

── Labels ────────────────────────────────────
? Replace GitHub default labels with custom set? [Y/n]
? Add custom labels? (name:color, empty to skip)
  > breaking:e11d48
  > (enter)

── Repositories ──────────────────────────────
? Add a repository? [Y/n]
? Repo name: xphone-rust
? Description: SIP telephony library for Rust
? Topics (comma-separated): sip,voip,telephony
? Homepage URL (optional):
? CI template: [None / Rust / Go / Node / Python / Custom]
? Add another repository? [Y/n]

── Teams (org only) ──────────────────────────
? Add a team? [Y/n]
? Team name: core
? Description: Core maintainers
? Permission: [Read / Write / Admin]
? Members (comma-separated): amenophis1er
? Add another team? [Y/n]

── Governance ────────────────────────────────
? Generate CONTRIBUTING.md? [Y/n]
? Add Code of Conduct (Contributor Covenant)? [Y/n]
? Add SECURITY.md? [Y/n]
? CODEOWNERS pattern: (* @x-phone/core)

── Security ──────────────────────────────────
? Enable Dependabot? [Y/n]
? Enable secret scanning? [Y/n]
? Enable code scanning? [y/N]

── Secrets ───────────────────────────────────
? Add org/repo secrets? [y/N]
? Secret name: CRATES_IO_TOKEN
? Scope: [Org / Repo]
? Add another? [y/N]

✓ Config written to gh-setup.yaml
  Review it, then run `gh-setup apply` to apply.
```

## Architecture

```
gh-setup/
├── go.mod
├── go.sum
├── main.go                  # entry point
├── cmd/
│   ├── root.go              # root cobra command
│   ├── init.go              # init wizard command
│   ├── apply.go             # apply command (--dry-run, -i flags)
│   └── diff.go              # diff command
├── internal/
│   ├── config/
│   │   └── config.go        # YAML config structs
│   ├── wizard/
│   │   └── wizard.go        # interactive init wizard (huh/bubbletea)
│   ├── github/
│   │   ├── client.go        # GitHub client wrapper (go-github)
│   │   ├── org.go           # org creation/settings
│   │   ├── repo.go          # repo creation/settings
│   │   ├── protection.go    # branch protection rules
│   │   ├── labels.go        # label management
│   │   ├── teams.go         # team + membership
│   │   ├── security.go      # dependabot, scanning
│   │   └── secrets.go       # secrets (prompted values)
│   ├── templates/
│   │   ├── ci.go            # workflow templates (rust, go, node, etc.)
│   │   └── governance.go    # CONTRIBUTING, CoC, SECURITY templates
│   ├── apply/
│   │   └── apply.go         # idempotent apply logic
│   └── diff/
│       └── diff.go          # config vs actual state comparison
```

## Branch Protection Presets

| Rule | None | Basic | Standard | Strict |
|------|------|-------|----------|--------|
| Require PR | - | - | 1 approval | 1 approval |
| Require status checks | - | - | - | yes |
| Require up-to-date | - | - | - | yes |
| Block force push | - | yes | yes | yes |
| Block deletion | - | yes | yes | yes |
| Enforce for admins | - | - | - | - |

## CI Workflow Templates

Built-in templates for common stacks:

- **rust** — `cargo fmt --check`, `cargo clippy -- -D warnings`, `cargo test`
- **go** — `go vet`, `golangci-lint`, `go test ./...`
- **node** — `npm ci`, `npm run lint`, `npm test`
- **python** — `ruff check`, `mypy`, `pytest`

Templates are embedded in the binary via `go:embed` and written to `.github/workflows/ci.yml` in each repo.

## Dependencies

| Module | Purpose |
|--------|---------|
| `github.com/spf13/cobra` | CLI command framework |
| `github.com/charmbracelet/huh` | Interactive form prompts |
| `github.com/charmbracelet/lipgloss` | Terminal styling and colors |
| `github.com/charmbracelet/log` | Structured logging |
| `github.com/google/go-github/v68` | GitHub REST API client |
| `golang.org/x/oauth2` | GitHub token auth |
| `gopkg.in/yaml.v3` | YAML config serialization |

## Apply Logic

Each resource follows the same pattern:

1. **Fetch** current state from GitHub API
2. **Compare** with desired config
3. **Skip** if already matching (idempotent)
4. **Mutate** if different (create/update)
5. **Report** action taken (created / updated / skipped / error)

In `--dry-run` mode, step 4 is replaced with a log of what would happen.
In `-i` mode, step 4 is gated by a confirmation prompt.

## Diff Output

```
$ gh-setup diff

  repo x-phone/xphone-rust
    branch_protection.require_pr:  false → true
    branch_protection.allow_force_push:  true → false
    labels:
      + breaking (e11d48)
      - wontfix (ffffff)

  repo x-phone/xphone-go
    ✓ up to date

  team core
    + member: new-contributor
```

## Distribution

Can be installed as a standalone binary or as a `gh` CLI extension:

```bash
# standalone
go install github.com/amenophis1er/gh-setup@latest

# as gh extension
gh extension install amenophis1er/gh-setup
gh setup init
```

## Future Ideas

- `gh-setup export` — reverse-engineer config from existing GitHub state
- `gh-setup template` — share configs across orgs (like Terraform modules)
- Plugin system for custom resource types (webhooks, deploy keys, etc.)
- GitHub App mode (run as a bot that enforces config on schedule)
