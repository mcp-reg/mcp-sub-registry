package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/mcp-reg/mcp-sub-registry/internal/cache"
	"github.com/mcp-reg/mcp-sub-registry/internal/model"
	"github.com/mcp-reg/mcp-sub-registry/internal/validator"
)

// RegistryService handles registry operations
type RegistryService struct {
	github    *GitHubClient
	http      *HTTPClient
	validator *validator.Validator
	cache     *cache.Manager
}

// NewRegistryService creates a new registry service
func NewRegistryService(github *GitHubClient, http *HTTPClient, v *validator.Validator, cacheMgr *cache.Manager) *RegistryService {
	return &RegistryService{
		github:    github,
		http:      http,
		validator: v,
		cache:     cacheMgr,
	}
}

// FetchRegistry fetches and compiles all servers from a registry with caching
func (s *RegistryService) FetchRegistry(ctx context.Context, ref model.RepoRef) ([]model.ServerWrapper, error) {
	// If cache disabled, fetch and transform directly
	if s.cache == nil {
		return s.fetchAndTransform(ctx, ref)
	}

	// Try cache first with TTL check to distinguish miss from expired
	cached, ttl, err := s.cache.GetWithTTL(ctx, ref)
	if err == nil {
		if ttl > 0 {
			slog.Info("cache hit",
				"org", ref.Org,
				"repo", ref.Repo,
				"branch", ref.Branch,
				"cache_key", ref.CacheKey(),
				"ttl_remaining", ttl,
			)
		} else {
			slog.Info("cache hit (TTL info not available)",
				"org", ref.Org,
				"repo", ref.Repo,
				"branch", ref.Branch,
				"cache_key", ref.CacheKey(),
			)
		}
		return cached, nil
	}

	// Cache miss or expired - fetch fresh
	// Note: GetWithTTL will return NotFound error for both miss and expired,
	// so we log as "cache miss or expired" since we can't distinguish reliably
	slog.Info("cache miss or expired, reloading",
		"org", ref.Org,
		"repo", ref.Repo,
		"branch", ref.Branch,
		"cache_key", ref.CacheKey(),
	)
	wrappers, fetchErr := s.fetchAndTransform(ctx, ref)

	if fetchErr != nil {
		// Fetch failed - try to return expired cache if available
		// Note: eko/gocache may return expired entries, but if not, we return error
		// This is lenient fallback behavior (Option B)
		expiredCached, expiredErr := s.cache.Get(ctx, ref)
		if expiredErr == nil {
			slog.Warn("fetch failed, returning stale cache",
				"org", ref.Org,
				"repo", ref.Repo,
				"branch", ref.Branch,
				"cache_key", ref.CacheKey(),
			)
			return expiredCached, nil // Return stale cache on fetch failure
		}
		return nil, fetchErr
	}

	// Store in cache (best effort - ignore errors)
	_ = s.cache.Set(ctx, ref, wrappers)
	slog.Info("cache updated",
		"org", ref.Org,
		"repo", ref.Repo,
		"branch", ref.Branch,
		"cache_key", ref.CacheKey(),
	)
	return wrappers, nil
}

// QueryServers fetches servers and applies filtering + pagination
func (s *RegistryService) QueryServers(ctx context.Context, ref model.RepoRef, params model.ListParams) (*model.ServerListResponse, error) {
	// 1. Fetch all (cached)
	wrappers, err := s.FetchRegistry(ctx, ref)
	if err != nil {
		return nil, err
	}

	// 2. Filter
	filtered := s.FilterServers(wrappers, params.Search)

	// 3. Paginate
	page, nextCursor := s.PaginateServers(filtered, params.Cursor, params.Limit)

	return &model.ServerListResponse{
		Servers: page,
		Metadata: model.ListMetadata{
			NextCursor: nextCursor,
			Count:      len(page),
		},
	}, nil
}

// fetchAndTransform fetches raw servers, applies transformation, returns result
func (s *RegistryService) fetchAndTransform(ctx context.Context, ref model.RepoRef) ([]model.ServerWrapper, error) {
	// Fetch registry.json
	regData, err := s.github.FetchFile(ctx, ref, "registry.json")
	if err != nil {
		if ghErr, ok := err.(*GitHubError); ok && ghErr.IsNotFound() {
			return nil, &RegistryError{
				Code:    400,
				Message: "Repository is not a valid MCP registry (missing registry.json)",
			}
		}
		return nil, fmt.Errorf("fetch registry.json: %w", err)
	}

	// Parse registry.json
	var regFile model.RegistryFile
	if err := json.Unmarshal(regData, &regFile); err != nil {
		return nil, &RegistryError{
			Code:    400,
			Message: fmt.Sprintf("Invalid registry.json: %v", err),
		}
	}

	// Process registry entries with transformation
	allWrappers := make([]model.ServerWrapper, 0)

	for _, entry := range regFile.Registries {
		var wrappers []model.ServerWrapper
		var err error

		if entry.IsPrivate() {
			wrappers, err = s.fetchPrivateServers(ctx, ref, entry)
		} else if entry.IsPublic() {
			if entry.ShouldFetchAllServers() {
				wrappers, err = s.fetchPublicAllServers(ctx, entry)
			} else if specificServers := entry.GetSpecificServers(); specificServers != nil {
				wrappers, err = s.fetchPublicSpecificServers(ctx, entry)
			} else {
				continue
			}
		} else {
			continue
		}

		if err != nil {
			return nil, err
		}

		// Apply transformer
		transformed := TransformAll(entry.URL, entry.Name, wrappers)
		allWrappers = append(allWrappers, transformed...)
	}

	if len(allWrappers) == 0 {
		return nil, &RegistryError{
			Code:    400,
			Message: "No servers found in any registry",
		}
	}

	return allWrappers, nil
}

// fetchPrivateServers fetches servers from a private GitHub registry
func (s *RegistryService) fetchPrivateServers(ctx context.Context, ref model.RepoRef, entry model.RegistryEntry) ([]model.ServerWrapper, error) {
	wrappers := make([]model.ServerWrapper, 0, len(entry.ServersRelativePath))

	for _, path := range entry.ServersRelativePath {
		serverData, err := s.github.FetchFile(ctx, ref, path)
		if err != nil {
			if ghErr, ok := err.(*GitHubError); ok && ghErr.IsNotFound() {
				return nil, &RegistryError{
					Code:    400,
					Message: fmt.Sprintf("Server definition not found: %s", path),
				}
			}
			return nil, fmt.Errorf("fetch %s: %w", path, err)
		}

		// Validate against schema if validator is configured
		if s.validator != nil {
			if err := s.validator.ValidateServer(serverData); err != nil {
				if ve, ok := err.(*validator.ValidationError); ok {
					return nil, &RegistryError{
						Code:    400,
						Message: fmt.Sprintf("Invalid schema at %s: %s", path, ve.Details),
					}
				}
				return nil, fmt.Errorf("validate %s: %w", path, err)
			}
		}

		var server model.Server
		if err := json.Unmarshal(serverData, &server); err != nil {
			return nil, &RegistryError{
				Code:    400,
				Message: fmt.Sprintf("Invalid server.json at %s: %v", path, err),
			}
		}

		wrappers = append(wrappers, model.ServerWrapper{Server: server})
	}

	return wrappers, nil
}

// fetchPublicAllServers fetches all servers from a public registry
func (s *RegistryService) fetchPublicAllServers(ctx context.Context, entry model.RegistryEntry) ([]model.ServerWrapper, error) {
	if s.http == nil {
		return nil, &RegistryError{
			Code:    500,
			Message: "HTTP client not configured for public registries",
		}
	}

	wrappers, err := s.http.FetchPublicServers(ctx, entry.URL)
	if err != nil {
		return nil, &RegistryError{
			Code:    502,
			Message: fmt.Sprintf("Failed to fetch from public registry %s: %v", entry.Name, err),
		}
	}

	return wrappers, nil
}

// fetchPublicSpecificServers fetches specific servers from a public registry
func (s *RegistryService) fetchPublicSpecificServers(ctx context.Context, entry model.RegistryEntry) ([]model.ServerWrapper, error) {
	if s.http == nil {
		return nil, &RegistryError{
			Code:    500,
			Message: "HTTP client not configured for public registries",
		}
	}

	specificServers := entry.GetSpecificServers()
	wrappers := make([]model.ServerWrapper, 0, len(specificServers))

	for name, version := range specificServers {
		wrapper, err := s.http.FetchPublicServerVersion(ctx, entry.URL, name, version)
		if err != nil {
			return nil, &RegistryError{
				Code:    502,
				Message: fmt.Sprintf("Failed to fetch %s:%s from public registry %s: %v", name, version, entry.Name, err),
			}
		}
		wrappers = append(wrappers, *wrapper)
	}

	return wrappers, nil
}

// FilterServers filters servers by search term (name substring match)
func (s *RegistryService) FilterServers(wrappers []model.ServerWrapper, search string) []model.ServerWrapper {
	if search == "" {
		return wrappers
	}

	search = strings.ToLower(search)
	filtered := make([]model.ServerWrapper, 0)
	for _, w := range wrappers {
		if strings.Contains(strings.ToLower(w.Server.Name), search) {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

// PaginateServers applies cursor-based pagination
// Cursor format: "serverName:version" (matches official API)
func (s *RegistryService) PaginateServers(wrappers []model.ServerWrapper, cursor string, limit int) ([]model.ServerWrapper, *string) {
	if limit <= 0 {
		limit = model.DefaultLimit
	}
	if limit > model.MaxLimit {
		limit = model.MaxLimit
	}

	// Find start index based on cursor
	startIdx := 0
	if cursor != "" {
		for i, w := range wrappers {
			cursorKey := fmt.Sprintf("%s:%s", w.Server.Name, w.Server.Version)
			if cursorKey == cursor {
				startIdx = i // FIX: Was i + 1, now starts WITH cursor item
				break
			}
		}
	}

	// Slice wrappers
	endIdx := startIdx + limit
	if endIdx > len(wrappers) {
		endIdx = len(wrappers)
	}

	result := wrappers[startIdx:endIdx]

	// Calculate next cursor
	var nextCursor *string
	if endIdx < len(wrappers) {
		next := fmt.Sprintf("%s:%s", wrappers[endIdx].Server.Name, wrappers[endIdx].Server.Version)
		nextCursor = &next
	}

	return result, nextCursor
}

// FindServer finds a server wrapper by name
func (s *RegistryService) FindServer(wrappers []model.ServerWrapper, name string) *model.ServerWrapper {
	for i := range wrappers {
		if wrappers[i].Server.Name == name {
			return &wrappers[i]
		}
	}
	return nil
}

// FindServersByName finds all versions of a server by name
func (s *RegistryService) FindServersByName(wrappers []model.ServerWrapper, name string) []model.ServerWrapper {
	var matches []model.ServerWrapper
	for _, w := range wrappers {
		if w.Server.Name == name {
			matches = append(matches, w)
		}
	}
	return matches
}

// SortByVersionDescending sorts server wrappers by semver version descending (latest first)
func (s *RegistryService) SortByVersionDescending(wrappers []model.ServerWrapper) {
	sort.Slice(wrappers, func(i, j int) bool {
		vi, erri := semver.NewVersion(wrappers[i].Server.Version)
		vj, errj := semver.NewVersion(wrappers[j].Server.Version)
		if erri != nil || errj != nil {
			return wrappers[i].Server.Version > wrappers[j].Server.Version
		}
		return vi.GreaterThan(vj)
	})
}

// FindLatestVersion finds the latest semver version among servers with same name
func (s *RegistryService) FindLatestVersion(wrappers []model.ServerWrapper, name string) *model.ServerWrapper {
	matches := s.FindServersByName(wrappers, name)
	if len(matches) == 0 {
		return nil
	}

	// Sort by semver descending
	sort.Slice(matches, func(i, j int) bool {
		vi, erri := semver.NewVersion(matches[i].Server.Version)
		vj, errj := semver.NewVersion(matches[j].Server.Version)
		if erri != nil || errj != nil {
			// Fallback to string comparison if not valid semver
			return matches[i].Server.Version > matches[j].Server.Version
		}
		return vi.GreaterThan(vj)
	})

	return &matches[0]
}

// FindSpecificVersion finds a specific version of a server
func (s *RegistryService) FindSpecificVersion(wrappers []model.ServerWrapper, name, version string) *model.ServerWrapper {
	for i := range wrappers {
		if wrappers[i].Server.Name == name && wrappers[i].Server.Version == version {
			return &wrappers[i]
		}
	}
	return nil
}

// InvalidateCache invalidates the cache for a repo ref
func (s *RegistryService) InvalidateCache(ctx context.Context, ref model.RepoRef) error {
	if s.cache == nil {
		return nil // No-op if cache disabled
	}
	return s.cache.Delete(ctx, ref)
}

// RegistryError represents a registry-level error
type RegistryError struct {
	Code    int
	Message string
}

func (e *RegistryError) Error() string {
	return e.Message
}
