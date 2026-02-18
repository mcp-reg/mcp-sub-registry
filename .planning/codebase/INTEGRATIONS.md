# External Integrations

**Analysis Date:** 2026-02-18

## APIs & External Services

**GitHub API:**
- Service: GitHub Contents API for file fetching
  - SDK/Client: Custom implementation in `internal/service/github.go`
  - Base URL: Configurable via `GITHUB_API_BASE` env var (default `https://api.github.com`)
  - Auth: Optional `GITHUB_TOKEN` env var for authenticated requests
  - Rate limiting: Handled with custom error type `GitHubError` tracking 429 status codes in `internal/service/github.go:99-116`
  - Endpoints used: `/repos/{org}/{repo}/contents/{path}?ref={branch}` - fetch individual files as base64-encoded content

**MCP Public Registry API:**
- Service: HTTP-based MCP server registries (pluggable)
  - SDK/Client: Custom `HTTPClient` implementation in `internal/service/http_client.go`
  - Base URL: Per-registry configurable via registry config URLs
  - Endpoints:
    - `GET /v0.1/servers` - List all servers with pagination (cursor-based)
    - `GET /v0.1/servers/{serverName}/versions/{version}` - Fetch specific server version
  - Pagination: Cursor-based via `nextCursor` in response metadata in `internal/service/http_client.go:81-84`
  - Timeout: 30 second request timeout in `internal/service/http_client.go:24`

## Data Storage

**Databases:**
- None - Stateless service, no persistent database

**File Storage:**
- Local Filesystem (in-memory primary, filesystem fallback)
  - Implementation: `internal/cache/filesystem_store.go` provides persistent cache
  - Cache location: `CACHE_BASE_DIR` env var (default `/tmp/mcp-registry-cache`)
  - Usage: Caching fetched MCP server specs to reduce GitHub/registry API calls

**Caching:**
- In-Memory Cache
  - Package: `github.com/patrickmn/go-cache` v2.1.0 (go-cache)
  - Wrapper: `github.com/eko/gocache/lib/v4` with gocache store implementation
  - TTL: Configurable via `CACHE_TTL` env var (default 1 hour)
  - Implementation: Chain pattern - memory first, filesystem fallback in `main.go:50-54`
  - Disabled: Set `CACHE_ENABLED=false` to skip caching entirely in `internal/config/config.go:50-53`

## Authentication & Identity

**Auth Provider:**
- Custom - GitHub token-based
  - Implementation: Bearer token in Authorization header to GitHub API in `internal/service/github.go:50`
  - Token source: `GITHUB_TOKEN` environment variable
  - Optional: Service works without token (lower rate limits) in `main.go:63-67`

**No User Authentication:**
- Service provides no user login/session management
- All APIs are read-only public endpoints
- HTTP endpoints in `internal/handler/` serve content without auth

## Monitoring & Observability

**Error Tracking:**
- None - Custom error handling only
- Errors logged via slog JSON logger in `main.go:25-26`

**Logs:**
- Structured logging with slog in JSON format
  - Handler: `slog.NewJSONHandler(os.Stdout, nil)` in `main.go:25`
  - Level: Info & Error messages throughout codebase
  - Examples: Cache initialization, server startup, version info in `main.go:27-56`

**Metrics:**
- Prometheus client pulled in as indirect dependency (via gocache)
- Not actively used in current implementation

## CI/CD & Deployment

**Hosting:**
- Self-hosted via standalone binary
- Stateless design allows container/serverless deployment
- Default runs on port 8080

**CI Pipeline:**
- GitHub Actions workflow: `.github/workflows/ci.yml`
  - Triggers: On push to main branch, all PRs, version tags (v*)
  - Go version: 1.25
  - Node version: 20
  - Steps: Frontend build, Go tests, linting, release binary builds on tags
  - Release job: Builds cross-platform binaries (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
  - Auto-generates GitHub release with release notes from changelog

**Build Process:**
- Makefile orchestration in `Makefile`
- Release targets: `make release-cli` builds all platform binaries with version info via ldflags
- Embedded frontend: Built with `-tags=embed` includes frontend dist in binary
- Version injection: Git tag, commit hash, build date injected via ldflags in `Makefile:50-53`

## Environment Configuration

**Required env vars:**
- `GITHUB_TOKEN` - Optional for authenticated GitHub API requests
- `PORT` - Server port (default 8080)
- `CACHE_ENABLED` - Toggle caching (default true)

**Optional env vars:**
- `GITHUB_API_BASE` - GitHub API endpoint (default `https://api.github.com`)
- `CACHE_TTL` - Cache expiration duration (default `1h`)
- `CACHE_BASE_DIR` - Filesystem cache location (default `/tmp/mcp-registry-cache`)
- `BROWSER_CACHE_TTL` - HTTP Cache-Control header TTL (default `5m`)

**Secrets location:**
- `.env` file - Example at `.env.example` shows expected variables
- Managed via environment variables only (no config files with secrets)

## Webhooks & Callbacks

**Incoming:**
- None - Service provides HTTP API only, no webhook endpoints

**Outgoing:**
- None - No external callbacks or webhooks

## Data Transformation

**Schema Validation:**
- JSON Schema validation: `github.com/santhosh-tekuri/jsonschema/v6` v6.0.2
- Schema location: `internal/validator/schemas/server.schema.json` (JSON schema for MCP server specs)
- Validator implementation: `internal/validator/` package validates registry/server configs

**Server Transformation:**
- MCP server wrapper transformation: `internal/service/transformer.go`
- Converts between registry formats and normalized server objects
- Includes metadata preservation (`OrigChildMeta`, `OrigParentMeta`) for nested metadata in `internal/service/http_client.go:73-74`

## Frontend-Backend Communication

**API Protocol:**
- JSON over HTTP
- Frontend proxy: `web/vite.config.ts:14-19` proxies API requests to backend during dev
- Pattern: Frontend requests Go backend at `/[org]/[repo]/[version]/v0.1/servers` endpoints

---

*Integration audit: 2026-02-18*
