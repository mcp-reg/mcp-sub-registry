# Coding Conventions

**Analysis Date:** 2026-02-18

## Naming Patterns

**Files:**
- Go: `snake_case.go` (e.g., `config.go`, `http_client.go`, `model_test.go`)
- TypeScript/React: `PascalCase.tsx` for components (e.g., `ServerCard.tsx`, `App.tsx`), `camelCase.ts` for utilities/types (e.g., `registry.ts`, `utils.ts`)
- Tests: Suffix with `_test.go` for Go tests (e.g., `transformer_test.go`, `github_test.go`)
- E2E tests: Named `e2e_test.go` with build tag `//go:build e2e` at top of file

**Functions:**
- Go: `PascalCase` for exported functions (e.g., `NewRegistryService()`, `FetchRegistry()`), `camelCase` for unexported (e.g., `extractGitHubMeta()`, `copyWrapper()`)
- TypeScript: `camelCase` for all functions (e.g., `formatStars()`, `fetchServers()`, `refreshCache()`)
- Test functions: Prefix with `Test` for Go (e.g., `TestTransformWrapper_VSCode()`, `TestGitHubClient_FetchFile_Success()`)

**Variables:**
- Go: `camelCase` for all (e.g., `ref`, `cacheManager`, `ghErr`)
- TypeScript: `camelCase` for all (e.g., `iconSrc`, `imgError`, `encodedName`)
- Constants: Go uses `PascalCase` (e.g., `DefaultLimit`), TypeScript uses `camelCase` with `const`

**Types:**
- Go structs: `PascalCase` (e.g., `Server`, `ServerWrapper`, `GithubInfo`, `RegistryService`)
- Go struct fields: `PascalCase` with JSON tags using `snake_case` (e.g., `WebsiteURL` → `json:"websiteUrl"`, `GithubInfo` → `json:"githubInfo"`)
- TypeScript interfaces: `PascalCase` (e.g., `Server`, `ServerWrapper`, `ServersResponse`)

## Code Style

**Formatting:**
- Go: Standard Go formatting enforced by `go fmt` and `gofmt`
- TypeScript: ESLint with flat config (`eslint.config.js`)
- No dedicated Prettier config - ESLint rules govern style

**Linting:**
- Go: `go vet` and `golangci-lint run` (see `Makefile` line 25-26)
- TypeScript: ESLint with `@eslint/js`, `typescript-eslint`, `react-hooks`, `react-refresh` plugins
- No strict style config for semicolons/quotes in TS (inherit from base configs)

## Import Organization

**Go Order:**
1. Standard library imports (`fmt`, `encoding/json`, `net/http`, `os`, `log/slog`)
2. Third-party imports (`github.com/...`, `github.com/go-chi/...`)
3. Internal imports (`github.com/mcp-reg/mcp-sub-registry/internal/...`)

Example from `main.go`:
```go
import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/cache"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/go-chi/chi/v5"
)
```

**TypeScript Order:**
1. React/framework imports
2. Third-party library imports
3. Type imports with `type` keyword
4. Internal imports (relative or aliased)

Example from `ServerCard.tsx`:
```typescript
import { useState } from "react"
import { useNavigate } from "react-router-dom"
import type { Server } from "@/types/server"
```

**Path Aliases:**
- TypeScript: `@/*` maps to `./src/*` (defined in `tsconfig.app.json`, lines 28-31)
- Use `@/` prefix for all internal imports in web code

## Error Handling

**Go Patterns:**
- Return error as last return value (e.g., `func FetchFile(...) ([]byte, error)`)
- Check errors with `if err != nil` followed by immediate handling
- Custom error types like `*GitHubError` and `*RegistryError` with methods like `IsNotFound()`, `IsRateLimited()`
- Use `slog.Error()` for unexpected errors in handlers after returning response to client
- Errors wrapped with context where needed (e.g., `"failed to initialize validator"`)

Example from `handler.go` lines 81-104:
```go
func (h *Handler) handleGitHubError(w http.ResponseWriter, err error, ref model.RepoRef) {
	if ghErr, ok := err.(*service.GitHubError); ok {
		switch {
		case ghErr.IsNotFound():
			h.writeError(w, http.StatusNotFound, "Repository not found")
		case ghErr.IsRateLimited():
			h.writeError(w, http.StatusTooManyRequests, "GitHub API rate limit exceeded")
		default:
			h.writeError(w, http.StatusBadGateway, "Failed to fetch from GitHub: "+ghErr.Message)
		}
		return
	}
	slog.Error("unexpected error", "error", err)
	h.writeError(w, http.StatusServiceUnavailable, "Service temporarily unavailable")
}
```

**TypeScript Patterns:**
- Use try/catch in async functions
- Generic error fallback with `.catch(() => ({ error: "..." }))`
- Throw `new Error()` with descriptive messages

Example from `registry.ts` lines 10-21:
```typescript
if (!response.ok) {
  const error: ErrorResponse = await response.json().catch(() => ({
    error: "Failed to fetch servers"
  }))
  throw new Error(error.error)
}
```

## Logging

**Framework:** Go uses `log/slog` (structured logging with JSON output)

**Patterns:**
- Initialize once in `main.go`: `slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))`
- Use `slog.Info()` for startup messages and normal operations
- Use `slog.Warn()` for recoverable issues (e.g., cache fallback)
- Use `slog.Error()` for unexpected errors (paired with client response)
- Structured fields as key-value pairs: `slog.Info("message", "key", value, "key2", value2)`

Example from `main.go` lines 25-27:
```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
slog.SetDefault(logger)
slog.Info("starting", "version", version.Info())
```

## Comments

**When to Comment:**
- Document exported functions/types with doc comments (e.g., `// Handler holds dependencies for HTTP handlers`)
- Explain non-obvious logic (e.g., why stale cache is returned on fetch failure)
- Clarify intent for complex operations (e.g., metadata transformation rules)

**JSDoc/TSDoc:**
- Not used in TypeScript code - minimal/no JSDoc comments observed
- Go doc comments use simple English sentences

Example from `transformer.go` line 10:
```go
// TransformWrapper transforms a wrapper based on source URL/name
// Returns a new ServerWrapper (does not modify original)
func TransformWrapper(sourceURL, registryName string, wrapper model.ServerWrapper) model.ServerWrapper {
```

## Function Design

**Size:** Functions typically 20-100 lines. Longer functions broken into smaller private helpers.

**Parameters:**
- Go: Max 3-4 parameters before considering a struct
- TypeScript: Props passed as single typed object in React components

**Return Values:**
- Go: Last return value is always `error` (if applicable)
- Multiple return values acceptable in Go (e.g., `Get(ctx, key)` → `(value, ttl, error)`)
- TypeScript: Return single value or typed object; async functions return Promise

Example of reasonable function size from `transformer.go` lines 135-180:
```go
func extractGitHubInfo(github map[string]interface{}) *model.GithubInfo {
	if github == nil {
		return nil
	}
	info := &model.GithubInfo{}
	// ... 40 lines of field extraction
	return info
}
```

## Module Design

**Exports:**
- Go: Exported via `PascalCase` naming; unexported via `camelCase`
- TypeScript: Use `export` keyword; organize in logical groups

**Barrel Files:**
- Not used in this codebase
- Direct imports from source files (e.g., `import type { Server } from "@/types/server"`)

**Package Organization:**
- Go: Single-responsibility packages (`handler/`, `service/`, `model/`, `config/`, `cache/`, `validator/`)
- TypeScript: Feature-based organization (`src/api/`, `src/types/`, `src/components/`, `src/pages/`, `src/lib/`)

## Struct/Interface Design

**Go Structs:**
- Include JSON struct tags for serialization (e.g., `json:"name"`, `json:"_meta,omitempty"`)
- Use pointers for optional fields (e.g., `Repository  *Repository` for optional repository)
- Constructor functions are `New<Type>()` pattern (e.g., `NewHandler()`, `NewValidator()`)

Example from `server.go` lines 4-20:
```go
type Server struct {
	Schema      string      `json:"$schema,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Version     string      `json:"version"`
	Title       string      `json:"title,omitempty"`
	GithubInfo  *GithubInfo `json:"githubInfo,omitempty"` // pointer for optional
	Meta        ServerMeta  `json:"_meta,omitempty"`
}
```

**TypeScript Interfaces:**
- Use `?` for optional fields
- Include descriptive names

Example from `server.ts` lines 23-32:
```typescript
export interface Server {
  name: string
  title?: string
  description: string
  version: string
  icons?: Icon[]
  githubInfo?: GithubInfo
  readme?: string
}
```

## Testing Conventions

- Test names include the function being tested and the scenario: `TestTransformWrapper_VSCode`, `TestGitHubClient_FetchFile_Success`
- Subtests use `t.Run()` with descriptive names: `t.Run("api.mcp.github.com", func(t *testing.T) { ... })`
- Table-driven tests for multiple scenarios (see `transformer_test.go` lines 148-187)
- Use `httptest.NewServer()` for mocking HTTP endpoints
- Create test fixtures inline for simplicity

---

*Convention analysis: 2026-02-18*
