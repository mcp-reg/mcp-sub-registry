# Feature Research

**Domain:** Go CLI tool for managing local JSON config files (MCP server registry)
**Researched:** 2026-02-18
**Confidence:** HIGH

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

#### `add` subcommand

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Parse `author/name` format from positional arg | Every registry tool does this; npm, cargo, pip all accept `pkg@version` style args | LOW | Validate format with regex before doing anything else |
| Generate well-formed `server.json` with `$schema` URL | The whole point of the command | MEDIUM | Support both stdio (npx/uvx) and remote (SSE/streamable-http) transport types |
| Create `mcps/author/name/server.json` directory tree | Users expect `add` to handle directory creation | LOW | Use `os.MkdirAll` |
| Update `registry.json` with new server path | Core workflow; without this the server isn't registered | MEDIUM | Must read-modify-write atomically; handle concurrent access |
| Refuse to overwrite existing `server.json` by default | Prevents accidental data loss; every scaffold tool does this | LOW | Exit non-zero with clear message pointing to `--force` |
| `--force` flag to overwrite existing files | Standard pattern (npm init --force, ef scaffold --force, wp scaffold --force) | LOW | Only overwrite server.json; still append to registry.json |
| `--quiet` / `-q` flag | Scripts/CI need silent operation; clig.dev mandates this | LOW | Suppress all human-facing output on stdout; errors still go to stderr |
| `--json` flag for machine-readable output | CI pipelines, editor integrations, and jq users need structured output | LOW | Output `{"path": "mcps/author/name/server.json", "registry_updated": true}` on stdout |
| Non-zero exit code on failure | Every CLI convention; scripts rely on `$?` | LOW | Exit 0 success, 1 validation/user error, 2 system/IO error |
| Human-readable success message showing what changed | clig.dev: "explain what changed when commands modify state" | LOW | Print file paths created/modified; suggest next command (`mcp-registry validate`) |
| `--help` / `-h` with usage examples | Universal expectation; cobra provides this free | LOW | Lead with examples, not flag reference |

#### `validate` subcommand

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Validate `config.json` against config schema | Core purpose of the command | MEDIUM | Use embedded schema via `go:embed` (already exists for server.schema.json) |
| Validate `registry.json` against registry schema | Core purpose | MEDIUM | Need to embed registry.schema.json |
| Validate each private `server.json` against server schema | Core purpose; iterate `servers_relative_path` entries | MEDIUM | Already have server schema validation in `internal/validator` |
| Report ALL errors, not just first | Every modern validator does this; ajv-cli, check-jsonschema, jsonschema-cli all collect all errors | MEDIUM | Collect errors across all files, report at end |
| Show file path + JSON path + human message per error | Users need to know WHERE the error is to fix it; blog posts complain when validators lack this | MEDIUM | Format: `registry.json: $.registries[0].name: must be at least 3 characters` |
| Exit code 0 = valid, non-zero = invalid | Standard for all validation CLIs (ajv-cli, check-jsonschema, jtd-validate) | LOW | Exit 1 for validation errors, 2 for system errors (missing file, bad JSON) |
| `--quiet` / `-q` flag | CI: just need exit code, not the error text | LOW | |
| `--json` flag for structured error output | CI/editor integrations parse structured errors; jsonschema-cli and ajv-cli both support this | LOW | Array of `{file, path, message, severity}` objects |
| `--help` with examples | Universal | LOW | Show example invocations and example error output |
| Handle missing files gracefully | Users forget to create config.json; don't panic/stack trace | LOW | "config.json not found. Run from your registry repo root." |
| Success message when all valid | Silent-on-success is confusing; users wonder if it ran | LOW | Print "All files valid." (suppress with `--quiet`) |

#### Shared / Cross-cutting

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| `--version` flag | Universal CLI expectation | LOW | Already have `internal/version` package |
| Colored terminal output (errors in red, success in green) | Modern CLI baseline; auto-disable when not TTY | LOW | Respect `NO_COLOR` env var, `--no-color` flag, non-TTY detection |
| Errors to stderr, data to stdout | Unix convention; clig.dev mandates; enables piping `--json` output | LOW | Use `os.Stderr` for messages, `os.Stdout` for `--json` data |
| Work from repo root (detect `registry.json` in cwd) | Users expect to `cd my-registry && mcp-registry add ...` | LOW | Check cwd for registry.json; error if missing |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| `--dry-run` flag for `add` | Show what would be created/modified without writing; mcp-publisher supports this; clig.dev recommends for state changes | LOW | Print planned actions to stderr, exit 0 |
| Auto-detect transport type from command args | If user provides `npx @author/pkg`, infer stdio transport; if URL, infer remote. Reduces flags needed | MEDIUM | Pattern match on args: `npx`/`uvx` = stdio, `http(s)://` = remote |
| `validate --fix` for auto-fixable issues | jsonschema CLI has `lint` and `fmt`; auto-fixing missing `$schema` fields or formatting issues saves time | HIGH | Only for safe fixes (add missing `$schema` URL, fix formatting). Defer to v1.x |
| Suggest next command after `add` | "Run `mcp-registry validate` to verify" - guides workflow | LOW | Print to stderr after successful add |
| Schema version awareness | Fetch remote `$schema` URL from server.json and validate against it (like Python validator.py does) | MEDIUM | HTTP fetch with timeout + cache; depends on network |
| `validate --watch` for continuous validation | Re-validate on file change; useful during editing | HIGH | Needs fsnotify; defer to v1.x |
| Summary count in validate output | "3 files checked, 1 error in 1 file" at end of validation run | LOW | Quick overview before detailed errors |
| Idempotent `add` (re-running is safe) | Don't duplicate registry.json entries; don't error if server.json exists and matches | MEDIUM | Check if path already in registry.json before appending |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Interactive prompts for `add` | "npm init style" feels friendly | Blocks CI pipelines; survey library is archived; adds complexity for a tool that takes 3-4 args | Accept all input via flags/args. TTY detection + prompts can come in v2 if demanded |
| Auto-fetch and cache remote schemas on validate | Python validator.py does this | Network dependency in a local tool; fails offline; cache invalidation complexity; slow first run | Embed known schemas via `go:embed`; only fetch remote `$schema` URLs explicitly |
| Global config file (~/.mcp-registry/config) | "Remember my defaults" | Overcomplicates a tool that runs from a repo root; config precedence hell; XDG compliance burden | Use flags. Repo-level config comes free from registry.json |
| `add --template` with multiple server templates | Power users want templates for common patterns | Premature abstraction; the server.json format IS the template; templates need maintenance | Good defaults in `add` flags cover 90% of cases |
| Colorized JSON output | Looks nice in demos | Breaks `--json | jq` piping; interferes with machine parsing | Only colorize human output; `--json` is always plain |
| `validate --auto-fix` for destructive changes | "Just fix everything" | Silently rewriting user config is dangerous; unclear what changed; can't undo | `--fix` only for safe additive changes (missing $schema); show diff of proposed changes |
| Plugin/hook system | Extensibility | Way too early; adds API surface to maintain; nobody needs this yet | Direct code changes for now; plugins in v2+ if ecosystem grows |
| YAML config format support | "YAML is more readable" | MCP ecosystem uses JSON exclusively; supporting both doubles test surface; JSON Schema validates JSON natively | JSON only. Period. |

## Feature Dependencies

```
[Embedded schemas (config, registry, server)]
    +-- requires --> [validate command works at all]
    +-- requires --> [add command generates valid server.json]

[Parse author/name format]
    +-- requires --> [add command: create directory tree]
    +-- requires --> [add command: update registry.json]

[--json flag]
    +-- requires --> [stderr/stdout separation]
    +-- enhances --> [--quiet flag] (--json implies data-only stdout)

[--force flag]
    +-- requires --> [default no-overwrite behavior]

[--dry-run flag]
    +-- enhances --> [add command] (show planned changes)
    +-- enhances --> [validate --fix] (preview fixes)

[Colored output]
    +-- conflicts --> [--json flag] (never color JSON)
    +-- requires --> [NO_COLOR / TTY detection]

[Auto-detect transport type]
    +-- enhances --> [add command] (reduces required flags)
    +-- requires --> [Parse author/name format]

[Idempotent add]
    +-- requires --> [add command: update registry.json]
    +-- enhances --> [--force flag] (force + idempotent = safe re-run)

[Summary count in validate]
    +-- requires --> [Report ALL errors]

[validate --fix]
    +-- requires --> [validate command]
    +-- requires --> [--dry-run for preview]
```

### Dependency Notes

- **Embedded schemas required before anything works:** Both `add` (generating valid JSON) and `validate` (checking JSON) depend on having schemas available via `go:embed`. Server schema already embedded; need config and registry schemas.
- **`--json` requires stderr/stdout separation:** If messages and JSON share stdout, piping breaks. This is foundational.
- **`--dry-run` enhances `add` and `validate --fix`:** Same preview pattern applies to both; implement the pattern once.
- **Colored output conflicts with `--json`:** Must never apply ANSI codes to JSON output. TTY detection handles this naturally.
- **Idempotent `add` enhances `--force`:** With idempotency, `--force` becomes "overwrite server.json but don't duplicate registry entry" which is the safe, expected behavior.

## MVP Definition

### Launch With (v1)

Minimum viable product -- what's needed to replace the Python scripts.

- [ ] `add` subcommand with positional `author/name` arg -- core workflow
- [ ] `add` supports stdio (npx/uvx) and remote (SSE/streamable-http) transport types via flags
- [ ] `add` creates directory tree + server.json + updates registry.json
- [ ] `add` refuses overwrite without `--force`
- [ ] `validate` subcommand validates config.json, registry.json, and all server.json files
- [ ] `validate` reports ALL errors with file path + JSON path + message
- [ ] Both commands: `--quiet`, `--json`, `--help`, non-zero exit codes
- [ ] Errors to stderr, data to stdout
- [ ] Colored output with NO_COLOR/TTY detection
- [ ] `--version` flag
- [ ] Embedded schemas for config, registry, and server

### Add After Validation (v1.x)

Features to add once core is working.

- [ ] `--dry-run` for `add` -- add after users request it or before `validate --fix`
- [ ] Auto-detect transport type from command args -- reduces friction
- [ ] Idempotent `add` (skip duplicate registry.json entries) -- usability improvement
- [ ] Schema version awareness (fetch remote $schema URL) -- parity with Python validator
- [ ] Summary count at end of validate output -- polish
- [ ] `validate --fix` for safe auto-fixes (add missing $schema) -- only after --dry-run exists

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] `validate --watch` -- requires fsnotify, niche use case
- [ ] Interactive prompts for `add` -- only if non-technical users adopt the tool
- [ ] Plugin/hook system -- only if ecosystem grows significantly

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| `add` core (create files, update registry) | HIGH | MEDIUM | P1 |
| `validate` core (all files, all errors) | HIGH | MEDIUM | P1 |
| `--quiet` and `--json` flags | HIGH | LOW | P1 |
| Non-zero exit codes | HIGH | LOW | P1 |
| stderr/stdout separation | HIGH | LOW | P1 |
| `--force` flag | HIGH | LOW | P1 |
| `--help` with examples | HIGH | LOW | P1 |
| Colored output + NO_COLOR | MEDIUM | LOW | P1 |
| Embedded schemas (config + registry) | HIGH | LOW | P1 |
| `--dry-run` for `add` | MEDIUM | LOW | P2 |
| Auto-detect transport type | MEDIUM | MEDIUM | P2 |
| Idempotent `add` | MEDIUM | MEDIUM | P2 |
| Remote `$schema` fetch in validate | MEDIUM | MEDIUM | P2 |
| Validate summary count | LOW | LOW | P2 |
| `validate --fix` | MEDIUM | HIGH | P3 |
| `validate --watch` | LOW | HIGH | P3 |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | mcp-publisher CLI | Python adder.py/validator.py | sourcemeta/jsonschema CLI | Our Approach |
|---------|-------------------|------------------------------|---------------------------|--------------|
| Add/scaffold server config | `init` generates template | Builds full server.json from args | N/A (schema tool, not registry) | `add` with positional arg + flags; generate complete server.json |
| Validate against schema | `publish --dry-run` | Validates config, registry, server.json; fetches remote schemas | `validate` with all drafts; line-level errors | `validate` all three file types; embedded schemas; file+path+message errors |
| Output formats | Human only | `--quiet`, `--json` | `--verbose` | `--quiet`, `--json`, colored human output |
| Overwrite protection | N/A | No protection | N/A | `--force` flag; refuse by default |
| Error detail level | Basic | file + path + message | file + line number + path | file + JSON path + message (line numbers deferred; JSON path sufficient for JSON files) |
| CI integration | Basic | Exit codes + `--json` | Exit codes + `--verbose` | Exit codes + `--json` + `--quiet` + stderr/stdout split |
| Dry run | `--dry-run` on publish | No | N/A | `--dry-run` on `add` (v1.x) |
| Transport auto-detect | N/A | No (explicit type required) | N/A | Auto-detect from npx/uvx/URL patterns (v1.x) |

## Sources

- [Command Line Interface Guidelines (clig.dev)](https://clig.dev/) -- authoritative CLI UX guidelines; HIGH confidence
- [MCP Registry CLI Tool (mcp-publisher)](https://modelcontextprotocol.info/tools/registry/cli/) -- official MCP publisher CLI; HIGH confidence
- [sourcemeta/jsonschema CLI](https://github.com/sourcemeta/jsonschema) -- comprehensive JSON Schema CLI with validate, lint, fmt; HIGH confidence
- [check-jsonschema docs](https://check-jsonschema.readthedocs.io/en/latest/usage.html) -- Python JSON Schema validator CLI patterns; HIGH confidence
- [ajv-cli](https://ajv.js.org/packages/ajv-cli.html) -- JavaScript JSON Schema validator CLI; HIGH confidence
- [Cobra CLI framework](https://cobra.dev/) -- Go CLI framework best practices; HIGH confidence
- [charmbracelet/log](https://github.com/charmbracelet/log) -- Go colored terminal output library; HIGH confidence
- [JSON Schema validation line numbers blog post](https://blog.gripdev.xyz/2025/01/31/cli-json-schema-validation-line-numbers-detailed-error/) -- user expectations around error detail; MEDIUM confidence
- [Go 1.24 JSON output](https://www.bytesizego.com/blog/go-124-json-output) -- Go CLI JSON output patterns; HIGH confidence

---
*Feature research for: Go CLI tool for MCP server registry config management*
*Researched: 2026-02-18*
