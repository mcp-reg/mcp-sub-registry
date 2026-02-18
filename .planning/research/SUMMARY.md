# Project Research Summary

**Project:** mcp-sub-registry CLI subcommands
**Domain:** Go CLI subcommand integration into existing HTTP server binary
**Researched:** 2026-02-18
**Confidence:** HIGH

## Executive Summary

This project adds two CLI subcommands (`add` and `validate`) to an existing Go HTTP server binary that currently has no CLI mode. The binary serves as an MCP server registry and must continue starting the HTTP server as its default behavior when invoked with no arguments — this backward compatibility constraint is the single most critical architectural requirement. The existing codebase has strong foundations (embedded schemas, validator package, model structs) that the CLI commands can reuse directly, making this a well-scoped extension rather than a new build.

The recommended approach uses Cobra v1.10.2 for CLI dispatch, with the root command's `Run` set to the existing server startup logic. The `add` command creates `mcps/author/name/server.json` and updates `registry.json`; the `validate` command validates all three file types (config, registry, server) using embedded schemas. All file writes must be atomic (write-to-temp-then-rename). The architecture follows a strict separation: CLI commands initialize only what they need — no cache, no GitHub client, no HTTP server setup — which keeps CLI startup under 50ms.

The primary risks are: breaking the default server dispatch during refactoring, corrupting `registry.json` with non-atomic writes, and introducing network dependencies into what should be a local validation tool. All three are avoidable with standard Go patterns that the architecture research specifies concretely. JSON output format (4-space indent, trailing newline) must match the existing Python scripts byte-for-byte to avoid noisy git diffs.

## Key Findings

### Recommended Stack

Cobra v1.10.2 is the clear choice for CLI dispatch. With only two subcommands, alternatives like Kong (struct-tag-based, 2,833 importers) and urfave/cli v3 (recent major rewrite) are unnecessary. Cobra's `rootCmd.Run` pattern directly solves the default-server-behavior requirement without hacks. pflag (Cobra's only dependency) handles the `--` separator natively via `ArgsLenAtDash()`, which is needed for the stdio command passthrough pattern (`mcp-registry add -- npx -y @pkg`). No other new dependencies are needed; all other functionality uses Go stdlib.

**Core technologies:**
- `spf13/cobra` v1.10.2: CLI framework + subcommand dispatch — root command `Run` = server mode; pflag handles `--` separator natively
- `spf13/pflag` v1.0.9: POSIX flag parsing (transitive via cobra) — provides `ArgsLenAtDash` for `-- npx -y @pkg` pattern
- `log/slog` (stdlib): Structured logging — already in use; consistent across CLI and server
- `encoding/json` (stdlib): JSON read/write — all file operations; must match Python's `indent=4` + trailing newline

### Expected Features

**Must have (v1 table stakes):**
- `add` subcommand: parse `author/name` arg, create `mcps/author/name/server.json`, update `registry.json` — core workflow replacing Python adder.py
- `add` supports stdio (npx/uvx) and remote (SSE/streamable-http) transports via flags
- `add` refuses to overwrite without `--force`
- `validate` subcommand: validate config.json, registry.json, and all server.json files against embedded schemas
- `validate` reports ALL errors (not just first) with file path + JSON path + message
- Both commands: `--quiet`, `--json`, `--help`, non-zero exit codes
- Errors to stderr, data to stdout (foundational for piping)
- Colored output with NO_COLOR / TTY detection
- `--version` flag (existing `internal/version` package)
- Embedded schemas for config and registry (server schema already embedded)

**Should have (v1.x after validation):**
- `--dry-run` for `add` — show planned changes without writing
- Auto-detect transport type from command args (npx/uvx = stdio, http(s):// = remote)
- Idempotent `add` — skip duplicate registry.json entries on re-run
- Summary count at end of validate output ("3 files checked, 1 error")
- Schema version awareness (fetch remote `$schema` URL as opt-in with `--remote-schema`)

**Defer (v2+):**
- `validate --watch` — requires fsnotify, niche use case
- Interactive prompts for `add` — blocks CI; only if non-technical users adopt the tool
- Plugin/hook system — way too early

**Anti-features (do not build):**
- Interactive prompts (blocks CI)
- Global `~/.mcp-registry/config` (config precedence hell)
- YAML support (MCP ecosystem is JSON-only)
- Colorized JSON output (breaks `--json | jq` piping)

### Architecture Approach

The architecture is a clean layered extension of the existing codebase. A new `internal/cmd/` package holds subcommand entry points (flag parsing + orchestration only). A new `internal/add/` package contains pure builder logic returning `model.Server` structs without touching the filesystem. A new `internal/registry/` package handles all local filesystem operations on registry files (shared by both commands). The existing `internal/validator/` is extended with two new methods and two new embedded schemas. The dispatch in `main.go` branches before any server initialization, so CLI commands pay zero startup cost for cache/GitHub/HTTP setup.

**Major components:**
1. `main.go` (modified) — dispatch: no-args routes to `startServer()`, subcommands route to `cmd.RunAdd/RunValidate`
2. `internal/cmd/add.go` + `internal/cmd/validate.go` — flag parsing, orchestration, output formatting
3. `internal/add/builder.go` — pure function: CLI inputs -> `model.Server` struct (no I/O)
4. `internal/registry/editor.go` + `reader.go` — atomic read/write of registry.json, config.json, server.json
5. `internal/validator/` (extended) — add `ValidateConfig()`, `ValidateRegistry()`, embed 2 new schemas; inject `SchemaFetcher` for remote validation

### Critical Pitfalls

1. **Breaking HTTP server default dispatch** — Set `rootCmd.Run` (not `RunE` without a subcommand) to the server startup logic. Verify with smoke test: bare `./mcp-registry` still binds port 8080. Must be first thing verified in Phase 1.

2. **Non-atomic registry.json writes** — Use write-to-temp-then-rename everywhere (`os.CreateTemp` + `os.Rename`). `os.Create` truncates eagerly; a crash mid-write leaves empty file. Consider `github.com/google/renameio` for Windows cross-platform safety.

3. **JSON format mismatch with Python scripts** — Use `json.MarshalIndent(data, "", "    ")` + append `"\n"`, or `json.Encoder` with `SetIndent("", "    ")`. Test output byte-for-byte against Python-generated golden files. Git will show noisy diffs if this is wrong.

4. **Remote schema fetch as network dependency** — Default to embedded schemas for both `add` and `validate`. Remote `$schema` URL validation must be opt-in via `--remote-schema` flag. The existing `internal/validator` already embeds server.schema.json; follow that pattern.

5. **`--` separator for stdio command args** — Cobra/pflag handle `--` natively, but `-y` in `npx -y @pkg` is parsed as a Cobra flag unless `--` is used. Require `--` before the command: `mcp-registry add -- npx -y @pkg`. Use `cmd.ArgsLenAtDash()`. Test matrix: with `--`, without, empty command after `--`.

6. **Init-heavy main.go slowing CLI** — Branch BEFORE any server initialization. CLI commands (`add`, `validate`) should not trigger config loading, cache init, or GitHub client construction. Startup target: <50ms.

## Implications for Roadmap

Based on research, the architecture file defines a clear 6-phase build order driven by dependency constraints. This maps directly to a roadmap:

### Phase 1: CLI Scaffold + Dispatch

**Rationale:** Must establish backward-compatible dispatch before any subcommand logic. The most critical pitfall (breaking server default) is only verifiable once dispatch exists. Nothing else can be built until main.go is refactored.
**Delivers:** Modified `main.go` with Cobra root command, `serve` as default, `version` subcommand. Existing server behavior fully preserved.
**Addresses:** `--version` flag, backward compatibility
**Avoids:** Pitfall 1 (broken dispatch), Pitfall 6 (init-heavy startup)

### Phase 2: Registry Filesystem Package

**Rationale:** Both `add` and `validate` need to read registry.json and config.json from disk. Building this shared package before either command avoids duplication and establishes the atomic write pattern from the start.
**Delivers:** `internal/registry/editor.go` + `reader.go` with atomic writes, unit tests
**Addresses:** `add` foundation, `validate` foundation
**Avoids:** Pitfall 2 (non-atomic writes), Pitfall 3 (JSON format mismatch)

### Phase 3: Extend Validator + Embed Schemas

**Rationale:** `validate` command depends on config and registry schemas being embedded. Extending the existing validator (rather than creating new one) preserves the established pattern and reuses compiled schema logic.
**Delivers:** `ValidateConfig()`, `ValidateRegistry()`, `ValidateAgainstRemoteSchema()` methods; embedded `config.schema.json` and `registry.schema.json`
**Addresses:** Validate core (all file types), schema version awareness foundation
**Avoids:** Pitfall 4 (network dependency — inject `SchemaFetcher` interface, default to embedded)

### Phase 4: `validate` Command

**Rationale:** Simpler than `add` (read-only, no file creation), so implement first to prove the validate pipeline works end-to-end. Also validates that Phase 2 and 3 building blocks are correct.
**Delivers:** `mcp-registry validate` working end-to-end: all errors collected, file+path+message format, `--quiet`, `--json`, exit codes
**Addresses:** All v1 validate table stakes
**Avoids:** Pitfall 4 (embedded schema default)

### Phase 5: `add` Command

**Rationale:** More complex than validate (file creation, directory setup, registry mutation). Depends on registry editor (Phase 2) and builder being correct.
**Delivers:** `mcp-registry add --name author/server --transport stdio -- npx -y @pkg` creating correct files, updating registry.json atomically, refusing overwrite without `--force`
**Addresses:** All v1 add table stakes
**Avoids:** Pitfall 2 (atomic writes), Pitfall 3 (JSON format), Pitfall 5 (`--` separator)

### Phase 6: Integration Testing + Polish

**Rationale:** Golden file tests comparing Go output to Python-generated fixtures catch format drift. Integration tests against fixture registries verify end-to-end flows.
**Delivers:** Full test coverage, golden file tests, `make test` and `make test-e2e` green
**Addresses:** CI integration, error message polish, colored output
**Avoids:** All pitfalls verified with the "looks done but isn't" checklist

### Phase Ordering Rationale

- Phases 1-2 are pure infrastructure with no dependencies on each other's content but must precede command phases
- Phase 3 before Phase 4 because validate command requires the extended validator
- Phase 4 before Phase 5 because validate is read-only (safer to build first) and shares the registry reader from Phase 2
- Phase 6 last because it tests the full system; golden files can only be written once the commands produce stable output
- This order matches the architecture research's explicit build order

### Research Flags

Phases with standard patterns (skip research-phase — well-documented):
- **Phase 1:** Cobra root command dispatch is a well-documented pattern; cobra docs + pitfalls research give concrete implementation
- **Phase 2:** Atomic file writes + JSON marshaling are standard Go; no novel decisions
- **Phase 3:** Extending existing validator with `go:embed` follows established pattern already in codebase
- **Phase 6:** Standard Go testing patterns; testify optional

Phases that may benefit from targeted investigation during planning:
- **Phase 5 (`add` command):** The `--` separator handling in Cobra has known edge cases (Pitfall 5). Review `cmd.ArgsLenAtDash()` behavior carefully before implementation, and define the arg parsing contract in a design note.
- **Phase 4 (remote schema):** Schema URLs fetched from `$schema` fields in server.json files need allowlisting to prevent SSRF (Pitfall 4 / Security). Clarify the allowlist domain before implementation.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Cobra verified on pkg.go.dev; pflag `ArgsLenAtDash` documented; no alternatives are clearly better for 2 subcommands |
| Features | HIGH | clig.dev + mcp-publisher + sourcemeta/jsonschema CLI all consulted; Python scripts analyzed directly |
| Architecture | HIGH | Based on direct codebase analysis; existing patterns (go:embed, validator, model) constrain decisions well |
| Pitfalls | HIGH | Based on direct analysis of existing Python scripts + existing Go code; pitfalls are concrete, not theoretical |

**Overall confidence:** HIGH

### Gaps to Address

- **config.schema.json and registry.schema.json locations:** Must fetch from mcp-registry-template repo before Phase 3. These schemas don't exist in the current Go codebase yet.
- **Windows atomic rename:** `os.Rename` is not atomic on Windows. The Makefile builds Windows binaries. Decision needed: add `github.com/google/renameio` dependency or document as best-effort on Windows.
- **registry.json key ordering:** Go's `encoding/json` sorts keys alphabetically; Python's `json.dump` preserves insertion order. This WILL produce git diffs when Go tools touch Python-generated files. Accept and document in CONTRIBUTING.md, or use `json.RawMessage` to preserve non-modified fields.
- **`$schema` URL allowlist:** Security research recommends allowlisting `static.modelcontextprotocol.io` domain for remote schema fetches. Confirm the correct domain before implementing the opt-in remote validation feature.

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis: `main.go`, `internal/validator/`, `internal/model/`, `internal/config/` — architecture and pitfall recommendations
- Direct script analysis: `mcp-registry-template/scripts/adder.py`, `validator.py`, `registry.py` — feature parity requirements
- [spf13/cobra pkg.go.dev](https://pkg.go.dev/github.com/spf13/cobra) v1.10.2 — stack recommendation
- [Command Line Interface Guidelines (clig.dev)](https://clig.dev/) — feature UX requirements
- [MCP Registry CLI Tool (mcp-publisher)](https://modelcontextprotocol.info/tools/registry/cli/) — competitor feature analysis

### Secondary (MEDIUM confidence)
- [JetBrains Go Ecosystem 2025](https://blog.jetbrains.com/go/2025/11/10/go-language-trends-ecosystem-2025/) — Cobra ecosystem dominance
- [sourcemeta/jsonschema CLI](https://github.com/sourcemeta/jsonschema) — validate command feature patterns
- [Atomically writing files in Go](https://michael.stapelberg.ch/posts/2017-01-28-golang_atomically_writing/) — atomic write pattern

### Tertiary (informational)
- [Go `encoding/json` trailing newline issue #13520](https://github.com/golang/go/issues/13520) — JSON format pitfall
- [alecthomas/kong](https://pkg.go.dev/github.com/alecthomas/kong) v1.14.0 — evaluated, not chosen
- [urfave/cli v3](https://pkg.go.dev/github.com/urfave/cli/v3) v3.6.2 — evaluated, not chosen

---
*Research completed: 2026-02-18*
*Ready for roadmap: yes*
