# Kitchen Sink Example

This folder contains a small, runnable monorepo you can use to try `go-changeset` end-to-end.

## Structure

```
kitchensink/
├── go.work
├── apps/
│   ├── backend/   # HTTP server on :8080
│   └── www/       # HTTP server on :8081
└── packages/
    └── shared/    # shared health handler
```

## Run the apps

From the `kitchensink/` directory:

```bash
go run ./apps/backend/main.go
```

```bash
go run ./apps/www/main.go
```

## Try the changeset workflow

From `kitchensink/`:

1) Create changesets

```bash
changeset add
# Select: backend, www, shared
# Pick bump type
# Write a message
```

2) Version projects

```bash
changeset each --filter=open-changesets -- changeset version
```

3) Inspect results

- `apps/*/version.txt`
- `apps/*/CHANGELOG.md`
- `packages/*/version.txt`
- `packages/*/CHANGELOG.md`

For CI/CD and snapshot/RC workflows, see `docs/github-integration.mdx` and `docs/snapshotting.mdx`.
