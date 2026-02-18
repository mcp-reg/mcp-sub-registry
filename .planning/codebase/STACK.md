# Technology Stack

**Analysis Date:** 2026-02-18

## Languages

**Primary:**
- Go 1.25 - Backend server & CLI binary
- TypeScript 5.9 - Frontend web app

**Secondary:**
- JavaScript (ES2022) - Build tooling, Vite config
- Bash - Makefile scripts, setup automation

## Runtime

**Environment:**
- Go 1.25 runtime (cross-compiled Linux amd64/arm64, macOS amd64/arm64, Windows amd64)
- Node.js 20 (frontend development)

**Package Manager:**
- Go modules - `go.mod` lockfile present at `go.mod`
- npm - `package.json` at `web/package.json`, dependencies installed to `web/node_modules/`

## Frameworks

**Backend:**
- Chi v5.2.3 - HTTP router & middleware (`github.com/go-chi/chi/v5`) in `main.go:10`
- chi CORS middleware v1.2.2 - Cross-origin request handling

**Frontend:**
- React 19.2.0 - UI framework
- React Router DOM 7.12.0 - Client-side routing
- Vite 7.2.4 - Build tool & dev server at `web/vite.config.ts`
- Tailwind CSS 4.1.18 - Utility-first CSS
- Tailwind CSS Typography 0.5.19 - Markdown/prose styling

**Testing:**
- Go test - Built-in, run with `make test`
- go test with race detector - `go test -v -race ./...`
- E2E tests with build tags - `go test -tags=e2e ./...`

**Build/Dev:**
- Make - Build orchestration `Makefile`
- TypeScript 5.9.3 - Frontend type checking
- Vite TypeScript support - `@vitejs/plugin-react` v5.1.1

## Key Dependencies

**Critical Backend:**
- `github.com/eko/gocache/lib/v4` v4.2.3 - In-memory & filesystem caching layer
- `github.com/eko/gocache/store/go_cache/v4` v4.2.4 - go-cache store implementation
- `github.com/patrickmn/go-cache` v2.1.0 - Underlying cache engine
- `github.com/santhosh-tekuri/jsonschema/v6` v6.0.2 - JSON schema validation for MCP server specs

**Version Management:**
- `github.com/Masterminds/semver/v3` v3.4.0 - Semantic versioning parsing

**Frontend UI Components:**
- `@radix-ui/react-slot` v1.2.4 - Composition primitive for components
- `lucide-react` v0.562.0 - Icon library
- `class-variance-authority` v0.7.1 - Component variant system
- `clsx` v2.1.1 - Class name utility
- `tailwind-merge` v3.4.0 - Merge Tailwind class conflicts

**Frontend Content:**
- `react-markdown` v10.1.0 - Render markdown in React
- `remark-gfm` v4.0.1 - GitHub Flavored Markdown support
- `rehype-raw` v7.0.0 - Raw HTML in markdown

**Indirect Dependencies:**
- Prometheus client_golang v1.23.2 - Metrics (pulled in by gocache)
- protobuf v1.36.10 - Serialization (pulled in by metrics)

## Configuration

**Backend Environment:**
- Port configuration - `PORT` env var (default 8080) in `internal/config/config.go:23-28`
- GitHub API - `GITHUB_API_BASE` (default https://api.github.com) in `internal/config/config.go:30-33`
- Cache TTL - `CACHE_TTL` (default 1h, parsed as duration) in `internal/config/config.go:38-43`
- Cache directory - `CACHE_BASE_DIR` (default /tmp/mcp-registry-cache) in `internal/config/config.go:45-48`
- Cache enable flag - `CACHE_ENABLED` (default true, set to "false" to disable) in `internal/config/config.go:50-53`
- Browser cache - `BROWSER_CACHE_TTL` (default 5m, HTTP Cache-Control) in `internal/config/config.go:56-61`
- GitHub token - `GITHUB_TOKEN` (optional, enables authenticated API requests) in `internal/config/config.go:35`

**Frontend Build:**
- TypeScript config: `web/tsconfig.json`, `web/tsconfig.app.json` (strict mode enabled)
- Vite config: `web/vite.config.ts` - React plugin, Tailwind CSS, path alias `@/*`
- ESLint config: `web/eslint.config.js` - JavaScript v9.39.1 config
- Tailwind config: Inline in Vite plugin, no separate `tailwind.config.js`

**Deployment:**
- Embedded frontend - Go binary built with `-tags=embed` includes frontend dist files
- Build output: Frontend built to `web/dist/`, copied to `internal/frontend/dist/` via `postbuild` script in `web/package.json:9`

## Platform Requirements

**Development:**
- Go 1.25+
- Node.js 20
- npm
- Make
- golangci-lint (for linting)
- Cross-compile toolchain for release builds

**Production:**
- Standalone binary (Linux amd64/arm64, macOS amd64/arm64, Windows amd64 supported)
- No external dependencies - all functionality built into single binary
- Writable filesystem for cache directory (default /tmp/mcp-registry-cache, configurable)

**Network:**
- Outbound HTTP to GitHub API (api.github.com) for file fetching
- Outbound HTTP to MCP public registry endpoints (configurable base URLs)
- HTTP server listening on configured port (default 8080)

---

*Stack analysis: 2026-02-18*
