# Requirements: mcp-sub-registry CLI

**Defined:** 2026-02-18
**Core Value:** Users can manage their private MCP registry with a single installed binary, getting updates automatically when they upgrade rather than re-cloning the template.

## v1 Requirements

### CLI Infrastructure

- [ ] **CLI-01**: Bare `mcp-registry` invocation (no subcommand) starts the HTTP server as before
- [ ] **CLI-02**: All commands and subcommands respond to `--help` with usage information
- [ ] **CLI-03**: Commands exit with non-zero status code on any failure
- [ ] **CLI-04**: Error messages go to stderr; normal output goes to stdout
- [ ] **CLI-05**: All commands support `--json` flag to emit structured JSON output instead of human-readable text
- [ ] **CLI-06**: `mcp-registry version` subcommand prints version string
- [ ] **CLI-07**: `mcp-registry cli` subcommand groups all local registry management commands (`add`, `validate`)

### Add Command

- [ ] **ADD-01**: User can specify server name in `author/name` format; invalid format is rejected with clear error
- [ ] **ADD-02**: User can add stdio transport server via `mcp-registry cli add` by passing command after `--` separator (e.g., `mcp-registry cli add --name author/name -- npx -y @pkg/server`)
- [ ] **ADD-03**: User can add remote transport server with `--transport sse|streamable-http --url <url>` flags via `mcp-registry cli add`
- [ ] **ADD-04**: Add command creates `mcps/author/name/server.json` with correct `$schema` field and transport definition
- [ ] **ADD-05**: Add command atomically updates `registry.json` to include the new server path under the private registry entry (write-to-temp-then-rename)
- [ ] **ADD-06**: User can provide `--description` flag to set server description in server.json
- [ ] **ADD-07**: User can pass `--dry-run` to preview the server.json that would be created and the registry.json change, without writing any files
- [ ] **ADD-08**: User can pass `--force` to overwrite an existing `server.json` (without --force, adding a duplicate name is an error)

### Validate Command

- [ ] **VAL-01**: `mcp-registry cli validate` validates `config.json` against the embedded `config.schema.json` schema
- [ ] **VAL-02**: `mcp-registry cli validate` validates `registry.json` against the embedded `registry.schema.json` schema
- [ ] **VAL-03**: `mcp-registry cli validate` validates every private `server.json` path listed in `registry.json`
- [ ] **VAL-04**: Validate command collects and reports ALL validation errors before exiting (not fail-fast)
- [ ] **VAL-05**: Each error is reported in `file: json.path: message` format
- [ ] **VAL-06**: Validate command fetches the remote schema URL from each server.json's `$schema` field to validate against
- [ ] **VAL-07**: Validate command exits with non-zero status code when any validation error is found

## v2 Requirements

### CLI Infrastructure

- **CLI-07**: `--quiet` flag suppresses non-error output across all commands

### Add Command

- **ADD-09**: `--env KEY` or `--env KEY=default` flags for stdio transport environment variable definitions
- **ADD-10**: `--package-name` and `--identifier` flags for explicit package registry metadata

### Validate Command

- **VAL-08**: `--skip-remote` flag to validate server.json files against embedded schema only (no network)
- **VAL-09**: `--watch` mode that re-validates on file changes

## Out of Scope

| Feature | Reason |
|---------|--------|
| compiler.py, fetcher.py, fetch_all_servers.py, registry.py | Server-side ops, not user-facing CLI tools |
| Interactive wizard / prompts | Flags-based CLI sufficient; survey lib is archived |
| Remote server management (create/delete via API) | Different concern from local file management |
| `mcp-registry serve` explicit alias | Backward compat handled by Cobra root default; alias adds confusion |
| JSON key ordering preservation | Go sorts alphabetically; document as known difference from Python |
| Windows atomic rename via renameio | Best-effort acceptable for v1 |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CLI-01 | Phase 1 | Pending |
| CLI-02 | Phase 1 | Pending |
| CLI-03 | Phase 1 | Pending |
| CLI-04 | Phase 1 | Pending |
| CLI-05 | Phase 1 | Pending |
| CLI-06 | Phase 1 | Pending |
| CLI-07 | Phase 1 | Pending |
| ADD-01 | Phase 2 | Pending |
| ADD-02 | Phase 2 | Pending |
| ADD-03 | Phase 2 | Pending |
| ADD-04 | Phase 2 | Pending |
| ADD-05 | Phase 2 | Pending |
| ADD-06 | Phase 2 | Pending |
| ADD-07 | Phase 2 | Pending |
| ADD-08 | Phase 2 | Pending |
| VAL-01 | Phase 3 | Pending |
| VAL-02 | Phase 3 | Pending |
| VAL-03 | Phase 3 | Pending |
| VAL-04 | Phase 3 | Pending |
| VAL-05 | Phase 3 | Pending |
| VAL-06 | Phase 3 | Pending |
| VAL-07 | Phase 3 | Pending |

**Coverage:**
- v1 requirements: 22 total
- Mapped to phases: 21
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-18*
*Last updated: 2026-02-18 after initial definition*
