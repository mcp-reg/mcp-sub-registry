# Architecture Research

**Domain:** CLI subcommand integration into existing Go HTTP server binary
**Researched:** 2026-02-18
**Confidence:** HIGH

## System Overview

```
                         main.go (dispatch)
                              |
               +--------------+--------------+
               |                             |
         [no subcommand]              [subcommand]
               |                             |
        startServer()              +---------+---------+
               |                   |                   |
        existing HTTP path    cmd/add.go          cmd/validate.go
        (config, cache,            |                   |
         service, handler,    internal/add/       internal/validator/
         router, listen)      builder.go          validator.go (extended)
                                   |                   |
                              internal/registry/   internal/registry/
                              editor.go            reader.go
                                   |                   |
                              Filesystem           Filesystem + HTTP
                              (read/write           (read local files,
                               registry.json,        fetch remote $schema)
                               write server.json)
```

### Component Responsibilities

| Component | Responsibility | Communicates With |
|-----------|----------------|-------------------|
| `main.go` (dispatch) | Detect subcommand vs server mode, route to appropriate entry point | `internal/cmd/*`, `startServer()` |
| `internal/cmd/add.go` | Parse `add` flags, orchestrate server.json creation and registry update | `internal/add`, `internal/registry` |
| `internal/cmd/validate.go` | Parse `validate` flags, orchestrate multi-file validation | `internal/validator` (extended), `internal/registry` |
| `internal/add/builder.go` | Build server.json structs for stdio/remote transports | `internal/model` |
| `internal/registry/editor.go` | Read/write registry.json, add paths to private entry | Filesystem |
| `internal/registry/reader.go` | Read and parse registry.json and config.json from disk | Filesystem |
| `internal/validator/` (extended) | Validate JSON files against schemas (server, config, registry) | `internal/service/http_client.go` (for remote $schema fetch) |
| `internal/model/` (existing) | Data structures for Server, RegistryFile, etc. | Nothing (pure data) |
| `internal/service/http_client.go` (existing) | HTTP client for fetching remote resources | Network |

## Recommended Project Structure

New files and packages to create:

```
.
├── main.go                          # MODIFY: add dispatch logic
├── internal/
│   ├── cmd/                         # NEW: CLI command entry points
│   │   ├── add.go                   #   add subcommand (flag parsing, orchestration)
│   │   └── validate.go              #   validate subcommand (flag parsing, orchestration)
│   ├── add/                         # NEW: server.json builder logic
│   │   ├── builder.go               #   build Server struct from CLI inputs
│   │   └── builder_test.go          #   unit tests
│   ├── registry/                    # NEW: local filesystem registry operations
│   │   ├── editor.go                #   read/modify/write registry.json
│   │   ├── reader.go                #   read config.json, registry.json, server.json from disk
│   │   └── editor_test.go           #   unit tests
│   ├── validator/                   # EXTEND: add config/registry/remote schema validation
│   │   ├── validator.go             #   extend with ValidateConfig, ValidateRegistry, ValidateAgainstURL
│   │   ├── validator_test.go        #   extend tests
│   │   └── schemas/
│   │       ├── server.schema.json   #   existing
│   │       ├── config.schema.json   #   NEW: embed config schema
│   │       └── registry.schema.json #   NEW: embed registry schema
│   └── ... (existing packages unchanged)
└── testdata/
    └── fixtures/                    # EXTEND with add/validate test fixtures
```

### Structure Rationale

- **`internal/cmd/`:** Separates CLI flag parsing and orchestration from business logic. Each file is a subcommand entry point that wires together lower-level packages. Not `cmd/` at project root because that Go convention is for multiple binaries; we have one binary with multiple modes.
- **`internal/add/`:** Isolates server.json construction logic. Pure functions that take inputs and return `model.Server`. Testable without filesystem.
- **`internal/registry/`:** Encapsulates all local filesystem operations on registry files. Both `add` and `validate` need to read registry.json; `add` also needs to write it. Shared package avoids duplication.
- **`internal/validator/` (extended):** Already compiles JSON schemas. Add new schemas (config, registry) to the embedded FS and add methods. For remote `$schema` validation, accept an HTTP-fetching function as a dependency rather than importing the HTTP client directly.

## Architectural Patterns

### Pattern 1: Dispatch-First in main.go

**What:** Check `os.Args` for subcommands before any server initialization. No heavy setup (config loading, cache init, service wiring) happens unless the server path is taken.

**When to use:** Always -- this is the entry point pattern.

**Trade-offs:** Simple, no library dependency for CLI parsing. Downside: manual flag parsing per subcommand. Acceptable given only 2 subcommands.

**Example:**
```go
func main() {
    if len(os.Args) > 1 {
        switch os.Args[1] {
        case "add":
            os.Exit(cmd.RunAdd(os.Args[2:]))
        case "validate":
            os.Exit(cmd.RunValidate(os.Args[2:]))
        case "version":
            fmt.Println(version.Info())
            return
        case "serve":
            // explicit serve mode, fall through to startServer
        default:
            // unknown subcommand or flags like --help
            printUsage()
            os.Exit(1)
        }
        return
    }
    // No subcommand: start HTTP server (backward compatible)
    startServer()
}
```

### Pattern 2: Flag Sets Per Subcommand

**What:** Each subcommand gets its own `flag.FlagSet` for independent flag parsing. Avoids global flag pollution.

**When to use:** For `add` and `validate` subcommands.

**Trade-offs:** Standard library only (no cobra/urfave dependency). Slight verbosity but keeps the dependency count at zero for CLI.

**Example:**
```go
// internal/cmd/add.go
func RunAdd(args []string) int {
    fs := flag.NewFlagSet("add", flag.ExitOnError)
    name := fs.String("name", "", "server name (author/name)")
    transport := fs.String("transport", "stdio", "transport type")
    dir := fs.String("dir", ".", "registry root directory")
    // ... more flags
    fs.Parse(args)
    // orchestrate
}
```

### Pattern 3: Dependency Injection for Testability

**What:** The `validate` command needs to fetch remote schemas via HTTP. Rather than importing `service.HTTPClient` directly into the validator, pass a `SchemaFetcher` function/interface.

**When to use:** When validator needs network access for remote `$schema` URLs.

**Trade-offs:** Slightly more wiring code, but makes validator unit-testable without network.

**Example:**
```go
// internal/validator/validator.go
type SchemaFetcher func(ctx context.Context, url string) ([]byte, error)

func (v *Validator) ValidateAgainstRemoteSchema(data []byte, schemaURL string, fetch SchemaFetcher) error {
    schemaData, err := fetch(context.Background(), schemaURL)
    // compile and validate
}
```

## Data Flow

### Add Command Flow

```
CLI: mcp-registry add --name author/server --transport stdio --runtime npx --package @scope/pkg
    |
    v
cmd/add.go: parse flags, validate inputs
    |
    v
add/builder.go: build model.Server struct from inputs
    |      - set $schema URL
    |      - set name, description, version
    |      - build packages[] or remotes[] based on transport
    |
    v
registry/editor.go: read registry.json from disk
    |      - find or create private registry entry
    |      - check for duplicate path
    |      - append mcps/author/name/server.json to servers_relative_path
    |
    v
Filesystem writes:
    1. mkdir -p mcps/author/name/
    2. write mcps/author/name/server.json (indented JSON)
    3. write registry.json (updated, indented JSON)
    |
    v
stdout: "Created mcps/author/name/server.json"
```

### Validate Command Flow

```
CLI: mcp-registry validate [--dir /path/to/registry]
    |
    v
cmd/validate.go: parse flags, resolve registry root dir
    |
    v
Step 1: Validate config.json against schemas/config.schema.json
    |      registry/reader.go reads config.json
    |      validator.ValidateConfig(data) using embedded schema
    |
    v
Step 2: Validate registry.json against schemas/registry.schema.json
    |      registry/reader.go reads registry.json
    |      validator.ValidateRegistry(data) using embedded schema
    |
    v
Step 3: For each private server in registry.json:
    |      registry/reader.go parses registry.json -> get private entry -> iterate servers_relative_path
    |      For each path:
    |        a. Read server.json from disk
    |        b. Extract $schema URL from the JSON
    |        c. Fetch schema from $schema URL (HTTP GET)
    |        d. Validate server.json against fetched schema
    |
    v
stdout: per-file pass/fail status
exit code: 0 if all pass, 1 if any fail
```

### Key Data Flows

1. **Add: Input to Filesystem** -- CLI flags are parsed into a `model.Server` struct by the builder, serialized to JSON, written to disk. Registry.json is read-modify-written.
2. **Validate: Filesystem to Verdict** -- Local files are read, schemas are loaded (embedded or fetched), validation results are collected and reported.
3. **Server mode (unchanged):** HTTP request -> handler -> service -> GitHub/cache -> response.

## Component Boundaries

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `cmd/*` -> `add/builder` | Function call (returns `model.Server`) | cmd parses flags, builder constructs structs |
| `cmd/*` -> `registry/*` | Function call (read/write files) | Pass registry root path, get/set data |
| `cmd/*` -> `validator` | Function call (validate bytes) | Pass file contents, get error or nil |
| `validator` -> `SchemaFetcher` | Callback/interface | Injected by cmd layer; decouples HTTP from validation |
| `add/builder` -> `model` | Direct struct construction | Uses existing `model.Server`, `model.Package`, `model.Remote` |
| `registry/*` -> `model` | JSON marshal/unmarshal | Uses existing `model.RegistryFile`, `model.RegistryEntry` |

### What Does NOT Communicate

- CLI commands do NOT touch `internal/service/registry.go` (that's for the HTTP server path)
- CLI commands do NOT touch `internal/handler/` (HTTP-only)
- CLI commands do NOT touch `internal/cache/` (server-only)
- CLI commands do NOT touch `internal/config/` (server-only env var config)
- The `add` package does NOT write files (it returns data; `cmd/add.go` or `registry/editor.go` writes)

## Build Order

Dependencies flow bottom-up. Build in this order:

### Phase 1: CLI Dispatch (foundation)

**What:** Modify `main.go` to detect subcommands. Extract existing server startup into `startServer()` function. Add `version` subcommand.

**Dependencies:** None (modifies existing file only).

**Validates:** Backward compatibility -- `mcp-registry` with no args still starts the server. `mcp-registry version` prints version info.

**Files:** `main.go`

### Phase 2: Registry Filesystem Operations

**What:** Create `internal/registry/` package. Implement `ReadRegistryFile()`, `ReadConfigFile()`, `ReadServerFile()`, `UpdateRegistryFile()` -- all operating on local filesystem paths.

**Dependencies:** `internal/model/` (existing).

**Validates:** Can read/parse registry.json, config.json. Can modify and write registry.json with new path entries.

**Files:** `internal/registry/editor.go`, `internal/registry/reader.go`, `internal/registry/editor_test.go`

### Phase 3: Extend Validator

**What:** Embed `config.schema.json` and `registry.schema.json`. Add `ValidateConfig()`, `ValidateRegistry()`, `ValidateAgainstRemoteSchema()` methods. The remote schema method accepts a `SchemaFetcher` function.

**Dependencies:** Existing `internal/validator/`, new schema files.

**Validates:** Config and registry files validate against their schemas. Remote schema fetch + validate works.

**Files:** `internal/validator/validator.go` (extend), `internal/validator/schemas/config.schema.json`, `internal/validator/schemas/registry.schema.json`, `internal/validator/validator_test.go` (extend)

### Phase 4: Validate Command

**What:** Create `internal/cmd/validate.go`. Wire together registry reader + extended validator + HTTP schema fetcher. Report results to stdout with exit code.

**Dependencies:** Phase 2 (registry reader), Phase 3 (extended validator), `internal/service/http_client.go` (existing, for schema fetching).

**Validates:** `mcp-registry validate` works end-to-end from a registry directory.

**Files:** `internal/cmd/validate.go`

### Phase 5: Add Command

**What:** Create `internal/add/builder.go` for server.json construction. Create `internal/cmd/add.go` for flag parsing and orchestration. Wire builder output to filesystem writes via registry editor.

**Dependencies:** Phase 2 (registry editor), `internal/model/` (existing).

**Validates:** `mcp-registry add --name author/server --transport stdio --runtime npx --package @scope/pkg` creates correct files.

**Files:** `internal/add/builder.go`, `internal/add/builder_test.go`, `internal/cmd/add.go`

### Phase 6: Integration Testing

**What:** Add testdata fixtures for add and validate. Write integration tests that exercise full command flows against fixture directories.

**Dependencies:** All previous phases.

**Files:** `testdata/fixtures/` (new fixtures), test files

## Reuse of Existing Packages

| Existing Package | How Reused | Extension Needed |
|------------------|-----------|------------------|
| `internal/validator/` | Core JSON schema validation engine | Add `ValidateConfig()`, `ValidateRegistry()`, `ValidateAgainstRemoteSchema()` methods; embed 2 new schema files |
| `internal/model/` | `Server`, `RegistryFile`, `RegistryEntry`, `Package`, `Remote` structs | None -- existing structs cover both add and validate needs |
| `internal/service/http_client.go` | HTTP GET for fetching remote `$schema` URLs during validation | Possibly add a simpler `FetchURL(url) ([]byte, error)` method, or wrap existing client behind `SchemaFetcher` interface |
| `internal/version/` | `version.Info()` for the `version` subcommand | None |

## Anti-Patterns

### Anti-Pattern 1: Cobra/Urfave for 2 Subcommands

**What people do:** Import a CLI framework for a handful of commands.
**Why it's wrong:** Adds a dependency, increases binary size, introduces framework concepts (persistent flags, hooks, completions) that are unnecessary here. The project already has zero CLI dependencies.
**Do this instead:** Use `os.Args` dispatch + `flag.FlagSet` per subcommand. If the CLI grows beyond 5 subcommands later, reconsider.

### Anti-Pattern 2: Sharing Server-Mode Dependencies with CLI Commands

**What people do:** Initialize cache, GitHub client, HTTP server config for CLI commands that don't need them.
**Why it's wrong:** Slow startup, confusing error messages if env vars are missing, unnecessary code paths.
**Do this instead:** The dispatch in main.go must branch BEFORE any server initialization. CLI commands construct only what they need.

### Anti-Pattern 3: Validator Directly Importing HTTP Client

**What people do:** Have the validator package import and create its own HTTP client.
**Why it's wrong:** Creates a hidden dependency, makes unit testing require network mocks, couples packages that should be independent.
**Do this instead:** Inject a `SchemaFetcher` function from the command layer. Validator stays pure: schema bytes in, validation result out.

### Anti-Pattern 4: Writing Files Inside Builder Logic

**What people do:** Have the builder function both construct JSON and write files.
**Why it's wrong:** Untestable without temp directories. Mixes concerns.
**Do this instead:** Builder returns `model.Server`. The command layer (or registry editor) handles serialization and file I/O.

## Schema Files Needed

The `validate` command requires two additional JSON schemas that must be embedded in the validator package:

1. **`config.schema.json`** -- Validates the registry's `config.json` file. Must match the schema used by the template repo (fetch from `schemas/config.schema.json` in the template).
2. **`registry.schema.json`** -- Validates `registry.json` structure. Must match the template repo's `schemas/registry.schema.json`.

These schemas should be copied from the [mcp-registry-template](https://github.com/mcp-reg/mcp-registry-template) repo and embedded alongside the existing `server.schema.json`.

## Sources

- Direct codebase analysis (HIGH confidence) -- all recommendations based on reading existing code structure
- Go standard library `flag` package documentation (HIGH confidence)
- Existing patterns in `internal/validator/` and `internal/model/` (HIGH confidence)

---
*Architecture research for: CLI subcommand integration into Go HTTP server binary*
*Researched: 2026-02-18*
