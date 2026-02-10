# MCP Sub-Registry

Go backend + React frontend for MCP server registry.

## Dev Server

```bash
make dev          # runs Go backend + Vite frontend concurrently
make run          # backend only
make web-dev      # frontend only
```

Frontend needs `web/npm install` first (`make web-install`).

## Tests

```bash
make test         # unit tests (go test -v -race ./...)
make test-e2e     # e2e tests (requires -tags=e2e)
make lint         # go vet + golangci-lint
```

## Release

CI auto-creates GitHub releases on version tags. To release:

```bash
git tag v0.x.x
git push origin v0.x.x
```

This triggers `.github/workflows/ci.yml` which builds cross-platform binaries and creates a GitHub release with auto-generated notes.

To build release binaries locally: `make release-cli`
