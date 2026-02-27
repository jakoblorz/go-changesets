# go-changeset

A CLI for managing changesets, versions, changelogs, and GitHub releases in **Go workspaces** (monorepos). Inspired by `@changesets/cli`.

## Quick start

### Install

```bash
go install github.com/jakoblorz/go-changesets/cmd/changeset@latest
```

### Prerequisites

- A Go workspace with a `go.work` at the repo root
- One or more Go modules referenced via `go.work use`

`go-changeset` also supports Node workspace projects via `package.json`, but Go workspaces are the primary workflow.


### Typical workflow

1) Add a changeset

```bash
changeset
# or
changeset add
```

2) Apply changesets to a project (bumps version + updates changelog)

```bash
changeset version --project <project>
```

3) Publish a GitHub release (creates git tag + GitHub release)

```bash
export GITHUB_TOKEN=...
changeset publish --project <project> --owner <org> --repo <repo>
```

### Batch operations

Run commands per project with filters:

```bash
# Version all projects that have pending changesets
changeset each --filter=open-changesets -- changeset version

# Publish all projects where the local version is newer than the latest tag
changeset each --filter=outdated-versions -- \
  changeset publish --owner <org> --repo <repo>
```

### GitHub Actions

Reusable workflows and composite actions ship with this repo:

- `.github/workflows/changesets.yml` (version PRs + publish)
- `.github/workflows/snapshot.yml` (RC snapshots)

Example usage:

```yaml
jobs:
  changesets:
    uses: jakoblorz/go-changesets/.github/workflows/changesets.yml@vX.Y.Z
    secrets:
      version_token: ${{ secrets.CI_PAT }}
      publish_token: ${{ secrets.CI_PAT }}
```

Tokens are optional; the workflows use `GITHUB_TOKEN`/`GH_TOKEN` if provided by the job.

## Documentation

Extended guides live in `docs/`:

- `docs/01_intro-to-using-changesets.mdx`
- `docs/02_adding-a-changeset.mdx`
- `docs/03_changeset-groups.mdx`
- `docs/04_snapshotting.mdx`
- `docs/05_github-integration.mdx`
- `docs/06_concepts.mdx`
- `docs/07_cli-reference.mdx`

## Architecture (high level)

- Business logic is separated from IO via mockable interfaces:
  - `internal/filesystem.FileSystem`
  - `internal/git.GitClient`
  - `internal/github.GitHubClient`
- Core domain types live in `internal/models`.
- CLI is handling imperative flow: `internal/cli`

## Testing

```bash
go test ./...
```

E2E tests:

```bash
go test ./test/e2e/... -v
```

## Kitchen sink example

A complete demo monorepo lives in `kitchensink/`.

## License

MIT
