# mcp-sub-registry

## What This Is

A Go binary that serves as both an MCP Registry REST API and a CLI tool for managing user-owned registry repos. Users clone the [mcp-registry-template](https://github.com/mcp-reg/mcp-registry-template), then use this binary to add private MCP servers and validate their registry — eliminating the need for Python scripts bundled in the template.

## Core Value

Users can manage their private MCP registry with a single installed binary, getting updates automatically when they upgrade the binary rather than re-cloning the template.

## Requirements

### Validated

- ✓ Binary serves MCP Registry REST API (`GET /{org}/{repo}/{branch}/v0.1/servers`, etc.) — existing
- ✓ JSON schema validation for server definitions — existing (`internal/validator`)
- ✓ Install script (`scripts/install.sh`) for binary distribution — existing
- ✓ Embedded React SPA frontend — existing

### Active

- [ ] `mcp-registry cli add` subcommand: create `server.json` in `mcps/author/name/` and register it in `registry.json`, run from within a cloned template repo
- [ ] `mcp-registry cli validate` subcommand: validate `config.json`, `registry.json`, and all referenced private `server.json` files against their JSON schemas, run from within a cloned template repo

### Out of Scope

- compiler.py, fetcher.py, fetch_all_servers.py, registry.py — server-side ops, not user-facing CLI
- Remote URL validation (network calls in CI) — defer to v2
- Interactive/wizard add mode — flags-based CLI is sufficient for v1

## Context

The template repo (`mcp-registry-template`) currently bundles Python scripts that users must run locally. This approach breaks when scripts are updated — users who cloned before the update don't get the fix. The binary solves this: it's versioned and upgradeable via install.sh.

**adder.py does:**
- Parses `author/name` format
- Builds `server.json` for stdio (npx/uvx), SSE, or streamable-http transports
- Creates `mcps/author/name/server.json`
- Adds the path to `registry.json` under the private registry entry

**validator.py does:**
- Validates `config.json` against local `schemas/config.schema.json`
- Validates `registry.json` against local `schemas/registry.schema.json`
- For each private server path in registry.json: fetches the `$schema` URL and validates `server.json`

**Current binary CLI:** None — `main.go` starts the HTTP server unconditionally. Adding subcommand dispatch is the first architectural change needed.

## Constraints

- **Tech stack**: Go only — no Python dependency in the binary
- **Schema fetching**: validator needs HTTP client (already exists in `internal/service/http_client.go`)
- **Run context**: commands run from the user's registry root dir; default to CWD, allow `--dir` override
- **Compatibility**: existing `mcp-registry` (no subcommand) must still start the HTTP server

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Subcommand dispatch in main.go | Keeps single binary, standard Go CLI pattern | — Pending |
| Reuse internal/validator for schema validation | Already validates server.json; extend for config/registry schemas | — Pending |
| Default to CWD for registry root | Matches template repo UX (cd my-registry && mcp-registry validate) | — Pending |

---
*Last updated: 2026-02-18 after initialization*
