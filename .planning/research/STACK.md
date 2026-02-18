# Stack Research: CLI Subcommand Library

**Domain:** Go CLI subcommand dispatch for existing HTTP server binary
**Researched:** 2026-02-18
**Confidence:** HIGH

## Recommendation: Cobra v1.10.2

Use `github.com/spf13/cobra` v1.10.2 (released Dec 4, 2024).

### Why Cobra for This Codebase

1. **Two subcommands, not twenty.** We're adding `add` and `validate`. Cobra handles this without ceremony. Kong's struct-tag approach shines when you have many flags/commands; for two commands it's needless indirection.

2. **Root command Run = server mode.** Cobra's `rootCmd.Run` fires when no subcommand is given. Put the existing HTTP server startup logic there. `mcp-registry` (no args) = server. `mcp-registry add ...` = CLI. Zero dispatch hacks needed.

3. **Existing ecosystem alignment.** The binary already uses chi, slog, go-cache -- all conventional Go libraries. Cobra is the conventional CLI library (kubectl, gh, hugo, helm). Kong is good but niche (~2,800 importers vs cobra's 100k+). For a project that ships cross-platform release binaries, community familiarity matters.

4. **pflag compatibility.** Cobra uses pflag (POSIX flags). The `add` command needs `--name`, `--transport`, and `--` separator for the command args. pflag handles `--` natively via `ArgsAfterDash`.

5. **No code generation required.** `cobra-cli` scaffolding is optional. We'll hand-write cmd/ files directly. Two commands don't warrant a generator.

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| spf13/cobra | v1.10.2 | CLI framework + subcommand dispatch | Industry standard for Go CLIs. Root command Run = default server mode. pflag handles `--` separator natively. |
| spf13/pflag | v1.0.9 | POSIX flag parsing (transitive dep of cobra) | Comes with cobra. Provides `ArgsAfterDash` for `-- npx -y @server` pattern. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| log/slog (stdlib) | Go 1.25 | Structured logging | Already used in project. Use in CLI commands too for consistency. |
| os/exec (stdlib) | Go 1.25 | Validate command existence | `add` command may want to verify the command in `--` args exists. Optional. |
| encoding/json (stdlib) | Go 1.25 | Read/write servers.json | CLI commands operate on local JSON files in user's registry repo. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| go test | Test CLI commands | Test command execution with cobra's `ExecuteC()` and buffer capture. |
| testify (if added) | Assertions | Project currently uses stdlib testing only. Keep it that way for CLI tests. |

## main.go Dispatch Pattern

### Current State

```go
func main() {
    cfg := config.Load()
    // ... setup validator, cache, services ...
    http.ListenAndServe(addr, router)
}
```

### Target State

```
main.go              -- calls cmd.Execute()
cmd/
  root.go            -- rootCmd with Run = startServer()
  add.go             -- addCmd: add server to local registry
  validate.go        -- validateCmd: validate local registry files
  server.go          -- startServer() extracted from current main.go
```

**root.go pattern:**

```go
var rootCmd = &cobra.Command{
    Use:   "mcp-registry",
    Short: "MCP server registry",
    Long:  "Run the MCP registry server, or use subcommands to manage a local registry.",
    Run: func(cmd *cobra.Command, args []string) {
        startServer() // existing main.go logic
    },
}

func init() {
    rootCmd.AddCommand(addCmd)
    rootCmd.AddCommand(validateCmd)
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

**Key dispatch behavior:**
- `mcp-registry` (no args) --> `rootCmd.Run` --> starts HTTP server
- `mcp-registry serve` --> optional explicit alias, same as no args
- `mcp-registry add --name foo/bar -- npx server` --> `addCmd.Run`
- `mcp-registry validate` --> `validateCmd.Run`
- `mcp-registry --help` --> auto-generated help listing subcommands

**main.go becomes:**

```go
func main() {
    cmd.Execute()
}
```

### The `--` Separator for Command Args

Cobra/pflag treats `--` as end-of-flags. Everything after `--` goes to `cmd.ArgsAfterDash()` or shows up in the `args` slice depending on cobra's `TraverseChildren` setting. For the `add` command:

```go
var addCmd = &cobra.Command{
    Use:   "add",
    Short: "Add an MCP server to the local registry",
    RunE: func(cmd *cobra.Command, args []string) error {
        // args after -- are in cmd.Flags().GetStringSlice or via ArgsAfterDash
        dashArgs := cmd.ArgsLenAtDash()
        if dashArgs >= 0 {
            serverCommand := args[dashArgs:]
            // serverCommand = ["npx", "-y", "@modelcontextprotocol/server-foo"]
        }
        return nil
    },
}
```

## Installation

```bash
go get github.com/spf13/cobra@v1.10.2
```

No other new dependencies needed. pflag comes transitively.

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| cobra v1.10.2 | alecthomas/kong v1.14.0 | If you prefer struct-tag-based CLI definition and have many nested commands. Kong's `default:"1"` tag handles default subcommand elegantly, but requires restructuring all commands as struct fields. Overkill for 2 commands. |
| cobra v1.10.2 | urfave/cli v3.6.2 | If you want a lighter library. Good for simple CLIs. But v3 was a recent major rewrite (API churn risk), and its subcommand model is less ergonomic than cobra for the root-command-as-server pattern. |
| cobra v1.10.2 | stdlib flag.FlagSet | If you want zero dependencies. Works for subcommands via manual dispatch (`os.Args[1]` switch). But no auto-help generation, no shell completion, no `--` handling, more boilerplate. Not worth the dep savings for a project that already has 7 direct dependencies. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| cobra-cli code generator | Generates scaffolding with opinions (license headers, viper integration) that bloat the codebase. Two commands don't need generation. | Hand-write cmd/ files directly. |
| spf13/viper | Cobra's companion config library. This project already has `internal/config` using env vars. Viper adds complexity (file watchers, remote config) not needed here. | Keep existing `config.Load()` for server mode. CLI commands read CWD files directly. |
| urfave/cli v2 | Deprecated in favor of v3. | If choosing urfave, use v3. But we recommend cobra. |
| kong | Not bad, just unnecessary. Struct-tag parsing is elegant for complex CLIs (10+ commands). For 2 commands, cobra's explicit `AddCommand` is clearer and more familiar to contributors. | cobra |
| flag stdlib | Manual dispatch, no help generation, no `--` handling, no completion. More code for worse UX. | cobra |

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| cobra v1.10.2 | Go 1.25 | cobra requires Go 1.16+. No issues with Go 1.25. |
| cobra v1.10.2 | pflag v1.0.9 | Automatically resolved. pflag is cobra's only real dependency. |
| cobra v1.10.2 | go-chi/chi v5 | No interaction. Cobra handles CLI parsing, chi handles HTTP routing. They coexist cleanly. |

## Confidence Assessment

| Decision | Confidence | Basis |
|----------|------------|-------|
| Use cobra | HIGH | Official pkg.go.dev docs, JetBrains 2025 Go ecosystem report, GitHub stars/importers, used by gh/kubectl/hugo |
| v1.10.2 specifically | HIGH | Verified on pkg.go.dev (published Dec 3, 2025), latest stable release |
| Root command = server pattern | HIGH | Documented cobra pattern: rootCmd.Run fires when no subcommand given. Standard in Go ecosystem. |
| Don't use kong | MEDIUM | Kong is good software. Recommendation against it is fit-based (2 commands, existing codebase style), not quality-based. |
| Don't use stdlib flag | HIGH | Project already has 7 deps, stdlib flag requires significantly more code for worse UX. |
| pflag handles `--` | HIGH | Documented pflag feature: `ArgsAfterDash` / `ArgsLenAtDash`. Verified in cobra docs. |

## Sources

- [spf13/cobra GitHub releases](https://github.com/spf13/cobra/releases) -- verified v1.10.2, Dec 4 2024
- [cobra pkg.go.dev](https://pkg.go.dev/github.com/spf13/cobra) -- published Dec 3 2025 (module cache date)
- [JetBrains Go Ecosystem 2025](https://blog.jetbrains.com/go/2025/11/10/go-language-trends-ecosystem-2025/) -- cobra listed as most widely used CLI library
- [alecthomas/kong pkg.go.dev](https://pkg.go.dev/github.com/alecthomas/kong) -- v1.14.0, Feb 6 2026, 2833 importers
- [urfave/cli v3 pkg.go.dev](https://pkg.go.dev/github.com/urfave/cli/v3) -- v3.6.2, Jan 18 2026
- [Kong default subcommand issue #41](https://github.com/alecthomas/kong/issues/41) -- default cmd limitations
- [Cobra default command issue #823](https://github.com/spf13/cobra/issues/823) -- rootCmd.Run as default
- [Go 1.25 release notes](https://go.dev/doc/go1.25) -- no flag package changes

---
*Stack research for: mcp-sub-registry CLI subcommands*
*Researched: 2026-02-18*
