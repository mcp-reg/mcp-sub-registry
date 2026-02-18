# Architecture

**Analysis Date:** 2026-02-18

## Pattern Overview

**Overall:** Layered API + SPA architecture with caching and external service integration.

**Key Characteristics:**
- Go backend serves REST API with embedded React SPA frontend
- Domain-driven layering: handlers → services → external clients
- Multi-source data aggregation (public registries + private GitHub repos)
- Pluggable caching layer with memory + filesystem fallback
- Data transformation to normalize registries from different sources

## Layers

**HTTP Handler Layer:**
- Purpose: Parse requests, delegate to services, format responses
- Location: `internal/handler/`
- Contains: Request routing, parameter extraction, error handling, HTTP response writing
- Depends on: `service.RegistryService`, chi router, models
- Used by: HTTP clients (React frontend, IDE clients)
- Key files: `handler.go` (request/response utilities), `router.go` (route definitions), `servers.go`, `refresh.go`, `health.go`, `versions.go`

**Service Layer:**
- Purpose: Business logic for registry aggregation, transformation, and filtering
- Location: `internal/service/`
- Contains: Registry fetching, server filtering, pagination, data transformation
- Depends on: GitHub client, HTTP client, validator, cache manager, models
- Used by: Handlers
- Key files: `registry.go` (main business logic), `github.go` (GitHub API client), `transformer.go` (registry normalization), `http_client.go`

**Cache Layer:**
- Purpose: Multi-tier caching to reduce GitHub API calls
- Location: `internal/cache/`
- Contains: Cache manager, filesystem store, TTL management
- Depends on: eko/gocache library, models
- Used by: Registry service
- Key files: `cache.go` (cache manager), `filesystem_store.go` (persistent cache)

**Data Model Layer:**
- Purpose: Canonical representations of MCP concepts
- Location: `internal/model/`
- Contains: Server definitions, wrapper structures, request/response DTOs
- Depends on: Nothing (pure data structures)
- Used by: All layers
- Key files: `server.go` (MCP server definition), `response.go` (API responses), `request.go` (URL/query params), `registry.go` (registry.json structure)

**Configuration & Infrastructure:**
- Purpose: App startup, external service initialization
- Location: `internal/config/`, `internal/version/`, `internal/validator/`, `internal/frontend/`
- Contains: Environment variable loading, version info, JSON schema validation, embedded SPA serving
- Depends on: Standard library and santhosh-tekuri/jsonschema
- Used by: main.go

**Frontend Layer (React SPA):**
- Purpose: Interactive registry browsing and setup UI
- Location: `web/src/`
- Contains: Pages, components, API client, type definitions
- Depends on: React, React Router, Vite
- Used by: Browser clients via embedded frontend

## Data Flow

**Registry Fetch & Transform Flow:**

1. **Request Entry**: Handler receives GET `/{org}/{repo}/{branch}/v0.1/servers`
2. **Extract Parameters**: `extractRepoRef()` and `extractListParams()` parse URL/query
3. **Cache Check**: `FetchRegistry()` checks in-memory cache, then filesystem cache
4. **GitHub Fetch** (on cache miss/expired):
   - Fetch `registry.json` from GitHub repo
   - Parse registry entries (public/private registries)
   - For each entry: fetch server definitions from external registries or private GitHub paths
5. **Transform**: `TransformAll()` normalizes data based on source:
   - VSCode/GitHub Copilot registries: extract metadata from `_meta`, standardize icons/title
   - Obot registries: merge namespace metadata
   - Other registries: add standard official status metadata
6. **Cache Storage**: Store transformed servers with TTL
7. **Filter & Paginate**: `FilterServers()` (search) → `PaginateServers()` (cursor-based)
8. **Response**: Return `ServerListResponse` with paginated results + metadata

**Stale Cache Fallback**:
- If fetch fails and cache expired: try to return stale cache (lenient behavior)
- Allows service to continue if GitHub is temporarily unavailable

**State Management:**
- No stateful session/auth layer — all requests are stateless
- Cache state persisted to filesystem (if enabled)
- Request context flows through all layers for tracing/cancellation

## Key Abstractions

**RepoRef (model):**
- Purpose: Uniquely identify a GitHub registry repository + branch
- Location: `internal/model/request.go`
- Fields: `Org`, `Repo`, `Branch`
- Methods: `CacheKey()` generates cache lookup string

**ServerWrapper (model):**
- Purpose: Server definition + metadata container
- Location: `internal/model/response.go`
- Fields: `Server` (core definition), `Meta` (namespaced metadata)
- Used by: All registry responses
- Metadata pattern: allows registries to attach custom data under namespace keys (e.g., `io.modelcontextprotocol.registry/official`)

**RegistryService (service):**
- Purpose: Main orchestrator for registry operations
- Location: `internal/service/registry.go`
- Methods: `FetchRegistry()` (with caching), `QueryServers()` (fetch + filter + paginate), `InvalidateCache()`, `FilterServers()`, `PaginateServers()`
- Injection: Receives `GitHubClient`, `HTTPClient`, `Validator`, `CacheManager` at construction

**Cache Manager (cache):**
- Purpose: Abstract cache interface hiding gocache implementation
- Location: `internal/cache/cache.go`
- Methods: `Get()`, `GetWithTTL()`, `Set()`, `Delete()`
- Underlying: Chainable (memory → filesystem) gocache stores

**GitHubClient (service):**
- Purpose: GitHub Contents API wrapper
- Location: `internal/service/github.go`
- Methods: `FetchFile()` (with auth token, error handling)
- Error types: `GitHubError` (status-aware, rate limit detection)

## Entry Points

**HTTP Server:**
- Location: `main.go`
- Triggers: `go run` or binary execution
- Responsibilities:
  1. Load config from environment
  2. Initialize logger (structured JSON logs to stdout)
  3. Initialize validator with embedded schema
  4. Initialize cache (memory + filesystem stores)
  5. Initialize GitHub client with optional auth token
  6. Create registry service with all dependencies
  7. Create handler with service
  8. Start HTTP server on configured port

**Handler Routes:**
- `GET /health` → health check
- `GET /{org}/{repo}/{branch}/v0.1/servers` → list servers (with search/pagination)
- `GET /{org}/{repo}/{branch}/v0.1/servers/{serverName}/versions/latest` → get specific server version
- `GET /{org}/{repo}/{branch}/v0.1/servers/{serverName}/versions/{version}` → get exact version
- `GET /{org}/{repo}/{branch}/v0.1/servers/{serverName}/versions/` → list all versions
- `POST /{org}/{repo}/{branch}/v0.1/refresh` → invalidate and refresh cache
- `GET /*` → SPA fallback (serves embedded React frontend, or 404 if not built with `-tags=embed`)

**Frontend Entry:**
- Location: `web/src/main.tsx`
- Mounts React app at `#root` in `web/public/index.html`
- Routes managed by React Router: `/` (setup), `/:org/:repo` (registry view), `/:org/:repo/:serverName` (server detail)

## Error Handling

**Strategy:** Layered error type hierarchy with HTTP status mapping.

**Patterns:**
- `GitHubError`: Wraps GitHub API errors, provides `IsNotFound()`, `IsRateLimited()` methods for handler decisions
- `RegistryError`: Custom errors from registry logic (e.g., missing registry.json) with HTTP status codes
- `ValidationError`: Schema validation failures (wrapped by validator)
- Handler converts error types to HTTP responses: 404 (not found), 429 (rate limited), 400 (validation/bad request), 503 (service unavailable)
- Logging: All errors logged via `slog` with context (org, repo, branch, error message)

## Cross-Cutting Concerns

**Logging:**
- Framework: `log/slog` (Go 1.21+ structured logging)
- Output: JSON to stdout (production-ready)
- Patterns: Info logs for request handling, warnings for cache issues, errors for failures
- Examples: cache hits/misses, GitHub API calls, validation failures

**Validation:**
- Schema: JSON Schema embedded in `internal/validator/schemas/server.schema.json`
- Tool: santhosh-tekuri/jsonschema v6
- Usage: Validators applied when fetching external server definitions
- Errors: Formatted to readable messages before sending to clients

**Authentication:**
- GitHub token: Optional env var `GITHUB_TOKEN` for higher API rate limits
- No user authentication: API is open (intended for private deployment behind org auth)
- Token injection: Added to GitHub API requests as Bearer token

**Caching:**
- TTL: Configurable (default 1 hour) via `CACHE_TTL` env
- Stores: Memory (fast) + filesystem (persistent across restarts)
- Chain strategy: Try memory first, fallback to filesystem
- Invalidation: POST `/refresh` endpoint clears for specific repo/branch
- Stale fallback: If fetch fails, returns expired cache if available

**CORS:**
- Configured permissively: Allow all origins, GET/POST/OPTIONS methods, Content-Type header
- Max age: 300 seconds

---

*Architecture analysis: 2026-02-18*
