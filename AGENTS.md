# AGENTS.md — go-changesets

Guidance for agentic coding assistants collaborating in this repository. Skim this file before touching code or tooling.

## 1. Project Snapshot
- `go-changesets` is a Go CLI that manages workspace-aware changesets, version files, changelog generation, and GitHub releases.
- Workspaces are rooted by `go.work`; each referenced module is a “project”.
- Versions live in `version.txt`; disabling a project is done by writing `false` (case-insensitive) into that file.
- Changesets are Markdown files in `.changeset/` with YAML front matter mapping project → bump type.
- Releases corollate to git tags shaped like `{project}@v{version}` and may include RC suffixes.
- Snapshot formatting (powered by `go-snaps`) guards user-facing text.

## 2. Build, Lint, and Test Commands
Always run commands from the repo root (`/Volumes/Develop/go-changesets`). Prefer targeted runs while iterating.

### Build / Install
- Build CLI: `go build -o changeset ./cmd/changeset`
- Install module globally: `go install github.com/jakoblorz/go-changesets/cmd/changeset@latest`

### Formatting
- Format entire tree: `gofmt -w $(git ls-files '*.go')`
- Package-wide alternative: `go fmt ./...`

### Lint / Static Checks
- Baseline: `go test ./...` (run after every non-trivial change)
- Extra safety: `go vet ./...`
- Optional (only if already available): `golangci-lint run`

### Tests (granular)
- Everything: `go test ./...`
- Single package: `go test ./internal/versioning`
- Single test (regex): `go test -run '^TestName$' ./internal/versioning`
- Single subtest: `go test -run '^TestName$/Subcase$' ./internal/versioning`
- Disable caching during iteration: `go test -count=1 ./...`
- Verbose mode: `go test -v ./...`
- E2E suite only: `go test ./test/e2e/... -v`

### Snapshot Tests (go-snaps)
- Update all snapshots: `UPDATE_SNAPS=true go test ./...`
- Snapshots live under `__snapshots__/` directories adjacent to their tests.
- Never hand-edit snapshot files; re-run with `UPDATE_SNAPS=true` instead.

## 3. Repository Layout (High Level)
- `cmd/changeset/main.go` → CLI entry; delegates to `internal/cli/root.go`
- `internal/cli` — Cobra commands, IO wiring
- `internal/changeset` — changeset persistence + PR metadata enrichment
- `internal/versioning` — bump logic, changelog formatting, append helpers
- `internal/workspace` — workspace discovery, builder utilities for tests
- `internal/git`, `internal/github` — integration layers + mocks
- `internal/filesystem` — filesystem abstraction/mocks
- `internal/models` — shared domain structs
- `internal/snapshot_tet.go` and `internal/workflow_test.go` — mocked end-to-end workflows

## 4. Code Style & Conventions
### Imports
1. Standard library
2. Third-party modules
3. Internal packages (`github.com/jakoblorz/go-changesets/...`)
Each group separated by a blank line. Avoid unused imports—`go test` will fail.

### Formatting & Structure
- Always run `gofmt`; do not rely on manual spacing.
- Keep diffs minimal; no opportunistic whitespace churn.
- Prefer early returns over deeply nested conditionals.
- Organize test helpers near their consumers when practical.

### Naming
- Exported identifiers: `PascalCase`
- Unexported: `camelCase`
- Avoid stutter (`version.Version` → `models.Version`).
- Use descriptive names; abbreviations only when universally understood (`ctx`, `err`, `fs`).

### Types & Interfaces
- Favor explicit structs over `map[string]interface{}` or `any`.
- Define interfaces in the consuming package; inject implementations from callers.
- Primary IO interfaces: `filesystem.FileSystem`, `git.GitClient`, `github.GitHubClient`.
- Avoid global state; pass dependencies via constructors.

### Error Handling
- Wrap errors with context: `fmt.Errorf("failed to read version: %w", err)`.
- Never swallow errors silently; bubble them up unless there is a compelling UX reason.
- Template failures (e.g., `.changeset/changelog.tmpl`) are fatal and must be surfaced.
- When comparisons require string search, prefer `strings.Contains` over manual loops.

### Logging / Output
- CLI commands should return errors and let Cobra print them; avoid `fmt.Println` for failure cases.
- Keep test output deterministic; avoid logging unless diagnosing failures.

## 5. Workspace, Changeset, and Changelog Rules
- Workspace root detection relies on `go.work`; never assume current directory is the root.
- Use `workspace.NewWorkspaceBuilder` + mocks in tests to stage projects/changesets quickly.
- `version.txt` is the canonical version; editing other representations without updating `version.txt` is incorrect.
- `internal/versioning` owns changelog rendering; do not reimplement formatting elsewhere.
- Snapshot tests guard markdown output—extend snapshots whenever formatting changes intentionally.

## 6. Testing Guidance
- Unit tests must use mocks: `filesystem.NewMockFileSystem`, `git.NewMockGitClient`, `github.NewMockGitHubClient`.
- Avoid real network/filesystem interactions.
- Use snapshot tests (`snaps.MatchSnapshot`) for human-readable content.
- Keep tests deterministic; avoid relying on current time unless injecting via `time.Date`.
- Prefer helper builders (e.g., `workspace.NewWorkspaceBuilder`) for complex setups.

## 7. CLI Behavior Notes
- Running `changeset` with no subcommand defaults to `changeset add`.
- Many commands support `changeset each` by reading JSON contexts from STDIN.
- Releases are created via git tags and optionally GitHub releases; tests typically stub those clients.

## 8. Tooling & Editor Rules
- Cursor rules: **none** (no `.cursor/rules/` or `.cursorrules` present).
- GitHub Copilot instructions: **none** (no `.github/copilot-instructions.md`). If these files appear later, update this section.
- When editing, prefer repo tools (`Read`, `Write`, `Edit`) over shell redirection; keep files ASCII unless existing content proves otherwise.

## 9. Safety & Scope for Agents
- Do not add new tooling (linters, formatters, dependencies) unless explicitly requested by the user.
- Avoid refactors that change CLI output or changelog text unless snapshots are updated intentionally.
- Never run destructive git commands (`reset --hard`, force push) without instruction.
- Respect existing uncommitted user work; do not revert or overwrite unrelated files.
- Keep PR-sized changes focused; skip documentation churn unless required by the change.

## 10. Recap Checklist (Before Hand-off)
1. `gofmt` applied to modified Go files.
2. `go test ./...` (and targeted packages) pass locally.
3. Snapshots updated via `UPDATE_SNAPS=true go test ./...` when formatting expectations change.
4. Imports grouped properly; no unused symbols.
5. Error messages include helpful context with `%w` wrapping where appropriate.
6. Tests rely on mocks/builders; no hidden filesystem/network dependencies.
7. Git working tree reviewed (`git status`) to ensure only intended files touched.

Following these rules keeps agent contributions predictable, reviewable, and aligned with project practices. Stay focused, test thoroughly, and prefer clarity over cleverness. 