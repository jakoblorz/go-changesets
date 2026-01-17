# AGENTS.md — go-changeset

This file is guidance for agentic coding assistants working in this repo.

## Project Overview

`go-changeset` is a CLI for managing changesets, versions, changelogs, and GitHub releases in **Go workspaces**.

Core concepts:

- A **workspace** is detected by a `go.work` file (walking up from CWD).
- A **project** is a `go.mod` referenced by `go.work use`.
- Versions are stored in `version.txt` in the project root.
- Changesets are Markdown files in `.changeset/` with YAML frontmatter.
- Releases are represented by git tags `{project}@v{version}`.

## Repo Layout (high level)

- `cmd/changeset/main.go -> internal/cli/root.go`: CLI entrypoint
- `internal/cli/`: cobra commands (wires IO + domain)
- `internal/changeset/`: read/write changesets + PR enrichment
- `internal/versioning/`: version bumping, changelog formatting/appending, snapshot notes
- `internal/workspace/`: workspace/project discovery
- `internal/git/`, `internal/github/`: integrations + mocks
- `internal/filesystem/`: filesystem abstraction + mock
- `internal/models/`: shared domain types
- `test/e2e/`: e2e tests (in-memory, mocked)

## Commands

### Build

- Build binary: `go build -o changeset ./cmd/changeset`
- Install (from module): `go install github.com/jakoblorz/go-changesets/cmd/changeset@latest`

### Format

- Format all packages: `gofmt -w $(git ls-files '*.go')`
- Alternative: `go fmt ./...`

### Test

- Run all tests: `go test ./...`
- Run a package: `go test ./internal/versioning`
- Run a single test (by name regex): `go test -run '^TestName$' ./internal/versioning`
- Run a single subtest: `go test -run '^TestName$/subtest$' ./internal/versioning`
- Disable test caching (useful while iterating): `go test -count=1 ./...`
- Verbose output: `go test -v ./...`
- E2E tests only: `go test ./test/e2e/... -v`

### Snapshot tests (go-snaps)

Snapshot tests are used to ensure formatting output does not change.

- Update snapshots: `UPDATE_SNAPS=true go test ./...`
- Typical snapshot location: `internal/**/__snapshots__/`

### Lint / static checks

This repo does not enforce a dedicated linter in-tree.

- Always run: `go test ./...`
- Recommended: `go vet ./...`
- If installed locally: `golangci-lint run` (do not add new lint tooling unless requested).

## Coding Guidelines

### Imports

- Group imports as:
  1) standard library
  2) third-party
  3) internal (`github.com/jakoblorz/go-changesets/...`)
- Separate groups with a blank line.

### Formatting

- Use `gofmt`. Do not hand-format.
- Prefer small diffs: avoid unrelated whitespace churn.

### Error handling

- Wrap errors with context: `fmt.Errorf("<context>: %w", err)`.
- Avoid swallowing errors. If the error indicates invalid configuration/input, fail the command.
- Template errors (e.g., `.changeset/changelog.tmpl`) are considered hard failures.

### Types and interfaces

- Prefer explicit types over `interface{}` / `any`.
- Define interfaces in the package that *consumes* them.
- Inject dependencies via interfaces, not globals.

Key IO interfaces:

- `internal/filesystem.FileSystem`
- `internal/git.GitClient`
- `internal/github.GitHubClient`

### Naming

- Exported: `PascalCase`; unexported: `camelCase`.
- Avoid stuttering: prefer `models.Changeset` not `models.ChangesetModel`.
- Prefer descriptive names over abbreviations, except common ones (`fs`, `ctx`, `err`).

### Project detection and versions

- Workspace root is found by locating `go.work`.
- Projects come from `go.work use` entries.
- Version source of truth is `version.txt`.
- Disabling a project: `version.txt` contains `false` (case-insensitive).

### Changesets

- Changesets live in `.changeset/*.md`.
- Content is Markdown with YAML frontmatter mapping project name → bump type.
- The CLI may create multiple changeset files for one logical change (one per project).

### Changelog formatting

- Changelog formatting is centralized under `internal/versioning/`.
- Formatting outputs are snapshot-tested; do not change formatting unless explicitly requested.
- Custom formatting can be provided via `.changeset/changelog.tmpl`.

## Testing Guidelines

- Unit Tests should not use the real filesystem or network.
- Prefer in-memory mocks:
  - `filesystem.NewMockFileSystem()`
  - `git.NewMockGitClient()`
  - `github.NewMockGitHubClient()`
- Prefer `internal/workspace/workspace_builder` for end-to-end / workflow style tests.

When adding tests:

- Keep them deterministic.
- Avoid time-dependent assertions unless the code explicitly includes time.
- Use snapshot tests for human-readable formatting output.

## CLI Behavior Notes

- Running `changeset` with no subcommand defaults to `changeset add`.
- Many commands support being invoked via `changeset each` by reading JSON context from STDIN.

## Editor / Assistant Rules

- Cursor rules: none found in `.cursor/rules/` or `.cursorrules`.
- GitHub Copilot rules: none found in `.github/copilot-instructions.md`.

## Safety / Scope

- Do not add new tooling (linters, formatters) unless requested.
- Avoid refactors that change output formats; protect with snapshot tests.
- Keep changes focused and minimal; update docs only when required by the change.
