# Pitfalls Research

**Domain:** Adding CLI subcommands to existing Go HTTP server binary (porting Python registry tooling to Go)
**Researched:** 2026-02-18
**Confidence:** HIGH (based on direct codebase analysis + verified Go ecosystem patterns)

## Critical Pitfalls

### Pitfall 1: Breaking HTTP Server Default Dispatch

**What goes wrong:**
Current `main.go` calls `config.Load()` then immediately boots `http.ListenAndServe`. Adding Cobra/subcommand dispatch without a backward-compatible default means existing deployments (Docker, systemd, `make run`) break because they pass zero args and expect the server to start.

**Why it happens:**
Cobra requires an explicit subcommand by default. If `serve` becomes a subcommand, bare `./mcp-registry` with no args shows help text instead of starting the server. Every existing deployment workflow breaks silently.

**How to avoid:**
Make the root Cobra command's `RunE` the HTTP server (current `main.go` logic). Subcommands like `add`, `validate`, `compile` are children. Bare invocation = serve. Alternatively, detect `len(os.Args) == 1` and inject `["serve"]` before Cobra parses, but the root-command approach is cleaner.

Pattern:
```go
rootCmd := &cobra.Command{
    Use: "mcp-registry",
    RunE: func(cmd *cobra.Command, args []string) error {
        return runServer(cmd.Context()) // existing main.go logic
    },
}
rootCmd.AddCommand(addCmd, validateCmd, compileCmd)
```

**Warning signs:**
- `make run` or `go run .` stops working after refactor
- Docker containers exit immediately with exit code 0
- CI health checks fail

**Phase to address:**
Phase 1 (CLI scaffold). Must be the first thing verified before any subcommand work begins. Write a smoke test: `go run . &` still binds port 8080.

---

### Pitfall 2: Non-Atomic registry.json Writes Causing Corruption

**What goes wrong:**
Python `adder.py` does `json.load` then `json.dump` on registry.json (lines 133-159). A crash or concurrent invocation between read and write leaves a truncated or empty file. Go's `os.Create` + `json.NewEncoder.Encode` has the same problem but is worse because Go truncates the file on `os.Create` before writing.

**Why it happens:**
`os.Create` opens with `O_TRUNC` -- the file is zeroed before the first byte is written. If the process crashes mid-write, the file is empty or partial. The Python version has the same theoretical risk, but Go makes it more likely because `os.Create` truncates eagerly.

**How to avoid:**
Write-to-temp-then-rename pattern:
```go
func atomicWriteJSON(path string, data any) error {
    tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
    if err != nil { return err }
    defer os.Remove(tmp.Name()) // cleanup on error

    enc := json.NewEncoder(tmp)
    enc.SetIndent("", "    ")
    if err := enc.Encode(data); err != nil {
        tmp.Close()
        return err
    }
    // Write trailing newline (Encode already adds one, see Pitfall 3)
    if err := tmp.Close(); err != nil { return err }
    return os.Rename(tmp.Name(), path)
}
```

Note: `os.Rename` is atomic on Linux/macOS but NOT on Windows. Since this project builds for Windows (see Makefile `windows-amd64`), consider using `github.com/google/renameio` for cross-platform safety, or document the Windows limitation.

**Warning signs:**
- Empty or zero-byte registry.json after interrupted `add` command
- JSON parse errors on next read after a failed write
- CI tests passing but prod registry.json occasionally corrupted

**Phase to address:**
Phase 2 (add command implementation). Implement atomic writes from the start -- retrofitting is trivial but forgetting is catastrophic.

---

### Pitfall 3: JSON Output Format Mismatch (Indent + Trailing Newline)

**What goes wrong:**
Python uses `json.dump(data, f, indent=4)` followed by `f.write("\n")` (adder.py lines 157-159, 195-197). Go's `json.MarshalIndent` does NOT add a trailing newline, but `json.Encoder.Encode` DOES add one. Using the wrong Go API produces files that diff against Python-generated files, breaking git workflows and making PRs noisy.

**Why it happens:**
Go has two JSON serialization paths with different newline behavior:
- `json.MarshalIndent` -- no trailing newline
- `json.NewEncoder(w).Encode` -- adds `\n` after output

Developers pick one without checking the other. Additionally, `json.MarshalIndent` uses `"    "` (4 spaces) by default only if you pass that; the default is `""` (no indent).

**How to avoid:**
Use `json.MarshalIndent(data, "", "    ")` then append `"\n"` manually. Or use `json.Encoder` with `SetIndent("", "    ")` which auto-appends newline via `Encode`. Either works, but be consistent across all write sites (server.json, registry.json, compiled output).

Verify with: `diff <(python3 -c "import json; json.dump({'a':1}, open('/dev/stdout'), indent=4); print()") <(go run ./cmd/test-json)`.

**Warning signs:**
- `git diff` shows trailing newline changes on every file the Go CLI touches
- Files opened in editors show "No newline at end of file" warnings
- Mixed indent styles (2-space vs 4-space) between files

**Phase to address:**
Phase 2. Define a single `writeJSON` helper used by all commands. Test output byte-for-byte against Python-generated fixtures.

---

### Pitfall 4: Remote Schema Fetch as Network Dependency in CLI

**What goes wrong:**
Python `validator.py` fetches `$schema` URL from each server.json at validation time (line 141-146, using `requests.get`). Porting this directly to Go means `validate` and `add` commands require network access. Users running offline, in CI without egress, or behind proxies get cryptic failures.

**Why it happens:**
The existing Go codebase already has an embedded schema (`internal/validator/schemas/server.schema.json` via `//go:embed`). But the Python validator fetches the remote schema URL declared in each `$schema` field, which may differ from the embedded one. Developers might port the Python behavior literally without realizing the Go server already solved this with embedding.

**How to avoid:**
Reuse the existing `internal/validator.Validator` for CLI commands. It uses the embedded schema, no network needed. If the CLI must also validate against the declared `$schema` URL (for spec compliance), make network validation opt-in via `--remote-schema` flag, with embedded as default.

For the `add` command: the `$schema` URL is hardcoded in `build_remote_server` and `build_stdio_server` (adder.py lines 50, 69). The Go version should use the same hardcoded URL but validate against the embedded schema, not fetch it.

**Warning signs:**
- `validate` command hanging for 10 seconds (HTTP timeout on unreachable schema host)
- CI jobs failing with "connection refused" when no egress allowed
- Tests requiring network mocking for what should be a local operation

**Phase to address:**
Phase 2 (validate command). Design decision needed early: embedded-first with optional remote validation.

---

### Pitfall 5: `--` Separator for Command Passthrough (argparse.REMAINDER Equivalent)

**What goes wrong:**
Python CLI uses `nargs=argparse.REMAINDER` for the `add` command's stdio command list (registry.py line 199-201), with manual `--` stripping (line 131). Cobra's flag parsing does not natively support REMAINDER semantics. The `--` separator is a POSIX convention that Cobra respects (stops flag parsing), but the remaining args go into `args []string` for the command -- developers must manually handle this, and edge cases abound.

**Why it happens:**
Cobra's `ArgsFunction` and `Args` validators don't have an equivalent of argparse.REMAINDER. `cmd.Flags().ArgsLenAtDash()` returns the index of `--` in args, but only if `TraverseChildren` is set correctly. If `--` is not present, all remaining args are treated as command args normally. But flags like `-y` in `npx -y @some/package` get parsed as Cobra flags unless `--` is used or `DisableFlagParsing` is set on the subcommand.

Example failure: `mcp-registry add -t stdio author/name npx -y @pkg` -- Cobra tries to parse `-y` as a flag of the `add` command.

**How to avoid:**
Two approaches:
1. **Require `--` before command:** `mcp-registry add -t stdio author/name -- npx -y @pkg`. Use `cmd.ArgsLenAtDash()` to split. Clear, explicit, matches Python behavior.
2. **Use positional args with `--command` flag:** `mcp-registry add -t stdio --command "npx,-y,@pkg" author/name`. Avoids REMAINDER issue entirely but worse UX.

Option 1 is better. Document it clearly. Test edge cases: no `--`, `--` with no command after, command containing flags.

**Warning signs:**
- "unknown flag" errors when passing commands like `npx -y`
- Commands silently losing arguments after flags
- User confusion about when `--` is required

**Phase to address:**
Phase 2 (add command). Define arg parsing contract in the design doc before implementation.

---

### Pitfall 6: Init-Heavy main.go Slowing CLI Commands

**What goes wrong:**
Current `main.go` initializes validator, cache (go-cache + filesystem store), GitHub client, HTTP client, and router before doing anything. If CLI dispatch happens after all this init, commands like `validate` or `add` (which need none of it) pay a startup penalty: filesystem cache init, network config, etc.

**Why it happens:**
Naive refactoring moves all existing init into the root command's `PersistentPreRun`, making every subcommand pay the cost. Or worse, init stays in `main()` before `rootCmd.Execute()`.

**How to avoid:**
Lazy initialization. Each subcommand initializes only what it needs:
- `serve`: full init (validator, cache, GitHub, HTTP, router) -- current behavior
- `add`: filesystem only (no cache, no network clients)
- `validate`: validator + filesystem (no cache, no HTTP)
- `compile`: validator + filesystem + HTTP client (for fetching public registries)

Pattern: pass a config struct to each command constructor, let RunE initialize its own dependencies.

**Warning signs:**
- CLI commands taking >100ms to start (should be <10ms for local-only operations)
- `add` command failing because `GITHUB_TOKEN` is not set (it shouldn't need it)
- Cache directory creation errors blocking non-server commands

**Phase to address:**
Phase 1 (CLI scaffold). Architecture decision: dependency injection per command, not global init.

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Sharing `internal/validator` between server and CLI without interface | Less code | Can't mock validation in CLI tests; server schema changes break CLI | MVP only -- extract interface before Phase 3 |
| Hardcoding `$schema` URL in Go adder | Matches Python behavior | Schema URL changes require code change + release | Acceptable if URL is a const, not buried in function body |
| Using `os.ReadFile` + `os.WriteFile` instead of atomic write | Simpler code | Registry corruption on crash | Never -- atomic writes are trivial to add |
| Single `main.go` with all command definitions | Quick to build | 500+ line main.go, hard to test | Only in Phase 1 scaffold; split into `cmd/` by Phase 2 |
| Skipping Windows path testing | Faster CI | `filepath.Join` vs hardcoded `/` breaks on Windows | Acceptable if Windows support is best-effort (but the Makefile builds Windows binaries) |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Remote `$schema` URLs (modelcontextprotocol.io) | Fetching on every validation call; no timeout; no cache | Use embedded schema for CLI; cache remote schema if fetched; 5s timeout with clear error message |
| registry.json read-modify-write | Reading into `map[string]interface{}` loses field order | Use `json.Decoder` with `UseNumber()` + `json.RawMessage` for fields you don't modify, or accept that Go's `encoding/json` does not preserve key order (it sorts alphabetically). Python preserves insertion order (dict is ordered since 3.7). This WILL cause diffs. |
| `mcps/author/name/server.json` path construction | Using `path.Join` (URL paths) instead of `filepath.Join` (OS paths) for local files | Always `filepath.Join` for local filesystem; always `path.Join` for URL paths and registry.json relative paths (which use forward slashes by design) |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Loading entire registry.json into memory for every `add` | Slow for large registries | Only matters at 10K+ servers; not a real concern for private registries | Unlikely to be a problem |
| Fetching remote schemas on every `validate` invocation | Slow validation, network-dependent | Embed schemas; cache fetched schemas per-process | Immediately in offline/CI environments |
| Cobra init overhead (reflection, flag registration) | Measurable only with 50+ subcommands | Not a concern for 4-5 commands | Never for this project |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Writing server.json with user-supplied `$schema` URL without validation | Schema URL could point to attacker-controlled server during validation | Allowlist schema URLs to `static.modelcontextprotocol.io` domain only |
| Passing unsanitized `--command` args to a shell | Command injection if add command ever executes the command | Never execute the command; only store it in server.json. The Python version also never executes it. |
| Storing env var defaults (from `--env KEY=secret`) in plaintext server.json | Secrets in git | Warn user if `--env` default value looks like a secret (starts with `sk-`, `ghp_`, etc.). Python doesn't do this either, but it's a good improvement. |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No `--dry-run` for `add` command | Users can't preview what will be created | Add `--dry-run` that prints the JSON and path without writing |
| Error messages that say "failed to add server: open mcps/foo/bar/server.json: permission denied" | User doesn't know what to fix | Wrap errors with context: "Cannot create server file at mcps/foo/bar/server.json -- check directory permissions" |
| `validate` failing on first error and stopping | User fixes one error, reruns, finds another | Collect all errors and report them together (Python validator already does this; port that behavior) |
| Silent success on `add` when `--quiet` is default | User doesn't know if it worked | Make non-quiet the default (print path created + registry updated). Python does this correctly. |
| Version command not available | Users can't report which binary version they're running | Add `mcp-registry version` subcommand using existing `internal/version` package |

## "Looks Done But Isn't" Checklist

- [ ] **Dispatch:** Bare `./mcp-registry` (no args) starts HTTP server -- verify in integration test
- [ ] **JSON format:** Output files match Python's `indent=4` + trailing newline -- byte-compare test
- [ ] **Atomic writes:** `add` command uses write-to-temp-then-rename -- verify with `kill -9` during write (or test with error injection)
- [ ] **Path separators:** registry.json always uses `/` in `servers_relative_path` regardless of OS -- verify on Windows CI or with `filepath.ToSlash`
- [ ] **Error codes:** CLI commands return non-zero exit code on failure -- test with `mcp-registry validate` on invalid registry
- [ ] **Schema URL:** Generated server.json contains correct `$schema` URL -- compare against Python-generated files
- [ ] **Existing tests pass:** `make test` and `make test-e2e` still pass after refactoring main.go
- [ ] **`--` handling:** `mcp-registry add -t stdio name -- npx -y @pkg` correctly captures `["npx", "-y", "@pkg"]` as the command

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Broke HTTP server dispatch | LOW | Revert main.go to pre-Cobra version; server logic is self-contained |
| Corrupted registry.json | MEDIUM | Restore from git (it's tracked); add atomic writes to prevent recurrence |
| JSON format mismatch (indent/newline) | LOW | Fix helper function; run `add` again to regenerate; git diff to verify |
| Wrong `$schema` URL in generated files | LOW | Find-and-replace in generated server.json files; update const |
| `--` args not parsed correctly | LOW | Fix Cobra arg parsing; no data corruption since args only affect what gets written |
| Init-heavy CLI startup | MEDIUM | Requires refactoring command init pattern; harder to retrofit than to design correctly |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| HTTP server dispatch break | Phase 1 (CLI scaffold) | Smoke test: `go run . &` binds port 8080; `go run . add --help` prints help |
| Non-atomic registry.json writes | Phase 2 (add command) | Test: kill process mid-write; file is either old or new, never corrupt |
| JSON format mismatch | Phase 2 (add command) | Golden file tests comparing Go output to Python-generated fixtures |
| Remote schema network dependency | Phase 2 (validate command) | Test: validate works with no network (use embedded schema) |
| `--` separator handling | Phase 2 (add command) | Test matrix: with `--`, without `--`, command with flags, empty command |
| Init-heavy startup | Phase 1 (CLI scaffold) | Benchmark: `time mcp-registry add --help` < 50ms |
| registry.json key ordering | Phase 2 (add command) | Accept that Go reorders keys; document in CONTRIBUTING.md |

## Sources

- [Go `encoding/json` Indent trailing newline issue](https://github.com/golang/go/issues/13520) -- confirms `json.MarshalIndent` has no trailing newline
- [Go `json.Encoder` adds newline](https://github.com/golang/go/issues/37083) -- confirms `Encoder.Encode` appends `\n`
- [Atomically writing files in Go](https://michael.stapelberg.ch/posts/2017-01-28-golang_atomically_writing/) -- write-to-temp-then-rename pattern
- [`github.com/google/renameio`](https://pkg.go.dev/github.com/google/renameio) -- cross-platform atomic rename library
- [Cobra user guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md) -- command structure, flag parsing
- [Cobra `DisableFlagParsing` issues](https://github.com/spf13/cobra/issues/1160) -- gotchas with pass-through args
- Direct analysis of: `mcp-registry-template/scripts/adder.py`, `validator.py`, `registry.py`, `compiler.py`, `fetcher.py`
- Direct analysis of: `mcp-sub-registry/main.go`, `internal/validator/validator.go`, `internal/config/config.go`, `internal/model/registry.go`

---
*Pitfalls research for: CLI subcommand addition to Go MCP sub-registry*
*Researched: 2026-02-18*
