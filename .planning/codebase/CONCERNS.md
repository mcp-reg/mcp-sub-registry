# Codebase Concerns

**Analysis Date:** 2026-02-18

## Tech Debt

**Shallow Copy in Metadata Transformation:**
- Issue: `transformer.go` uses shallow copies for `ServerMeta` maps via `copyMeta()` and wrapper copies via `copyWrapper()`. Nested maps/slices reference the same underlying data as source.
- Files: `internal/service/transformer.go` (lines 235-276)
- Impact: Modifications to nested metadata (e.g., `wrapper.Meta["io.modelcontextprotocol.registry/official"]`) can affect other instances sharing the same reference. Risk when metadata is enriched in handlers.
- Fix approach: Deep copy metadata maps recursively, especially for mutable maps like `map[string]interface{}`. Consider using JSON marshal/unmarshal pattern or explicit recursive copy function.

**Best-Effort Cache Writes Without Validation:**
- Issue: Cache writes ignore errors silently with comment "best effort" in `registry.go` line 92-93.
- Files: `internal/service/registry.go` (line 93: `_ = s.cache.Set(ctx, ref, wrappers)`)
- Impact: Silent failures mean cache never gets populated on filesystem I/O errors. Users won't know data isn't being cached. Subsequent requests may hit slow paths repeatedly.
- Fix approach: Log cache errors explicitly; consider making cache failure non-fatal but observable via logging/metrics.

**Cursor Pagination Off-By-One Behavior:**
- Issue: Pagination logic in `registry.go` line 310 contains inline comment "FIX: Was i + 1, now starts WITH cursor item" - cursor iteration may include duplicate when cursor moves to next page.
- Files: `internal/service/registry.go` (lines 304-314)
- Impact: Clients using pagination cursors may see duplicate server entries when paginating through large lists.
- Fix approach: Clarify intended behavior (exclusive vs inclusive cursor semantics) and test with large datasets to verify no duplicates.

**Frontend API Error Handling Silence:**
- Issue: Frontend error handling in `registry.ts` catches JSON parse errors but doesn't differentiate between parse failures and actual error responses.
- Files: `web/src/api/registry.ts` (lines 14-16, 45-47)
- Impact: Network errors, malformed responses, and genuine API errors all result in same generic message to user. Harder to debug issues.
- Fix approach: Add error type detection; log raw response when parse fails; provide specific error messages for different failure modes.

## Security Considerations

**Cache File Permissions:**
- Risk: Cache files written with `0644` permissions (readable by any local user on shared systems). Cached registry data may contain sensitive metadata or tokens in URLs.
- Files: `internal/cache/filesystem_store.go` (line 125: `os.WriteFile(path, data, 0644)`)
- Current mitigation: Filesystem permissions depend on umask of process. No explicit protection for sensitive data in cache.
- Recommendations: (1) Use restrictive permissions `0600` for cache files; (2) Document that cache directory should be in secure location with proper parent directory permissions; (3) Consider encrypting cache at rest if sensitive URLs are stored.

**Cache Key Sanitization May Be Bypassed:**
- Risk: `keyToPath()` sanitization removes special characters but `filepath.Base()` is called after replacement. An attacker-controlled repo ref could craft keys that bypass sanitization.
- Files: `internal/cache/filesystem_store.go` (lines 157-169)
- Current mitigation: Multiple sanitization passes (replaceAll for `/\`, filepath.Base, rune map filter). But order matters.
- Recommendations: (1) Use `url.QueryEscape()` or similar standard escaping for cache keys; (2) Add tests with adversarial repo refs (e.g., `../`, `..\\`, unicode chars); (3) Whitelist allowed characters instead of blacklist.

**GitHub Token Handling:**
- Risk: GitHub API token passed as Bearer token without explicit validation. Token could be logged in error messages or debug output.
- Files: `internal/service/github.go` (line 50: `"Authorization", "Bearer "+g.token`)
- Current mitigation: Token from env var only, not logged at startup besides boolean flag (line 64 in `main.go`).
- Recommendations: (1) Add sanitization for error messages containing URLs with tokens; (2) Never log full error bodies from GitHub API that might echo auth headers; (3) Consider token validation/rotation strategy for long-lived registries.

## Performance Bottlenecks

**Stale Cache Fallback on Every Fetch Failure:**
- Problem: When fetch fails, service attempts `cache.Get()` again (ignoring TTL) to return stale data. This causes additional filesystem I/O on network errors.
- Files: `internal/service/registry.go` (lines 75-89)
- Cause: Network error triggers fallback check instead of returning immediately. Adds latency to error path.
- Improvement path: (1) Cache the stale entry reference separately when storing; (2) Reduce fallback to in-memory check only; (3) Implement circuit breaker pattern to skip fallback after repeated failures.

**Pagination Requires Full Registry Load:**
- Problem: `QueryServers()` fetches entire registry even for paginated requests. With large registries (1000+ servers), memory and processing time scales poorly.
- Files: `internal/service/registry.go` (lines 104-124)
- Cause: No streaming or partial fetch; service loads all wrappers into memory before filtering/pagination.
- Improvement path: (1) Implement cursor-aware registry fetching to avoid full load; (2) Cache filtered results by query pattern; (3) Add database or indexed store for large registries instead of file-based.

**HTTP Client Timeout Fixed at 30 Seconds:**
- Problem: All HTTP operations (GitHub, public registries) use fixed 30s timeout. No configurable timeout per request type.
- Files: `internal/service/github.go` (line 33), `internal/service/http_client.go` (line 24)
- Cause: Hard-coded timeout prevents tuning for slow registries or network conditions.
- Improvement path: (1) Make timeout configurable via env var; (2) Implement different timeouts for different registry types; (3) Add exponential backoff instead of hard fail.

## Fragile Areas

**Type Assertions on Unvalidated Metadata:**
- Files: `internal/service/transformer.go` (multiple), `internal/handler/versions.go` (line 143)
- Why fragile: Registry metadata comes from untrusted external sources (GitHub, public registries). Type assertions like `.(map[string]interface{})` will silently fail and return nil if structure doesn't match expected format. Leads to missing metadata without error.
- Safe modification: (1) Add schema validation before type assertions; (2) Log mismatches for debugging; (3) Use structured metadata types with JSON validation instead of `interface{}`.
- Test coverage: Transformer tests in `internal/service/transformer_test.go` test known good formats but don't test malformed/unexpected metadata structures.

**Cache Manager Nil Check Pattern:**
- Files: `internal/service/registry.go` (lines 38, 234, 254, 400), `main.go` (line 48)
- Why fragile: Multiple conditional nil checks for cache manager throughout code. If cache is accidentally nil when expected or vice versa, behavior degrades silently. No panic or explicit error.
- Safe modification: (1) Use optional/wrapper type instead of nil sentinel; (2) Create separate "NoCacheManager" implementation that handles cache calls as no-ops; (3) Validate cache initialization at startup.
- Test coverage: Unit tests may not cover all cache=nil code paths.

**String Concatenation for Error Messages:**
- Files: `internal/handler/versions.go` (lines 33, 73, 77, 118), `internal/handler/handler.go` (lines 86, 92)
- Why fragile: URL parameters concatenated directly into error strings. If org/repo contain quotes or special chars, error message formatting breaks.
- Safe modification: (1) Use struct with fields instead of string concat; (2) URL-encode or escape user inputs in error messages; (3) Use fmt with %q for safe string inclusion.
- Test coverage: Happy path tests; limited error path testing with special characters.

## Error Handling

**Strategy:** Service layer returns custom error types (`GitHubError`, `RegistryError`, `PublicRegistryError`). Handler layer converts these to HTTP responses. Frontend layer throws generic errors.

**Gaps:**
- No structured error codes for client differentiation (e.g., "RATE_LIMITED" vs "INVALID_REGISTRY")
- GitHub API 5xx errors caught but not retried; client gets immediate failure
- Validation errors from server JSON unmarshal not distinguished from schema validation errors
- Frontend doesn't handle 502/503 differently from 404

## Test Coverage Gaps

**Backend:**
- No test for cache invalidation after fetch success
- No test for concurrent requests with same cache key
- No test for malformed registry.json with missing required fields
- No test for transformer with invalid metadata structures
- No e2e test for pagination cursor across multiple pages
- GitHub client tests don't cover edge cases: 403 without "rate limit" string, redirects, timeouts

**Frontend:**
- No test for error states in OrgRepoPage/ServerDetailPage
- No test for failed API calls with UI recovery
- No test for icon loading failures with fallback rendering
- No test for concurrent requests (race conditions)

## Missing Critical Features

**Missing pagination safeguard:**
- Issue: Client requests `limit=1000000` with no enforced max. Service has `MaxLimit` of 100 but no server-side hard cap on in-memory processing.
- Impact: Large arbitrary limits could cause OOM or slow down service
- Files: `internal/service/registry.go` (lines 298-302)
- Fix: Enforce maximum limit hard cap before loading data.

**No authentication/authorization:**
- Private registries exposed to any request without verification
- No rate limiting per client/IP
- No audit trail for who accessed what

## Scaling Limits

**Filesystem Cache Bottleneck:**
- Current capacity: Single directory with sanitized filenames. Each repo/branch = 1 JSON file.
- Limit: Filesystem limits (max files per directory, inode exhaustion). On many repos/branches, filesystem operations slow down.
- Scaling path: (1) Use key-value store (Redis, Memcached); (2) Implement multi-level directory structure; (3) Add cache eviction policy by LRU/TTL.

**Memory Usage on Large Registries:**
- Problem: All servers loaded into memory as `[]model.ServerWrapper`. Large registries (10k+ servers) consume proportional memory per request.
- Limit: Memory exhaustion under concurrent load if registries are large.
- Scaling path: (1) Stream responses instead of buffering; (2) Implement database indexing; (3) Cache aggregated responses.

---

*Concerns audit: 2026-02-18*
