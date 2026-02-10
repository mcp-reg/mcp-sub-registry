package cache

import (
	"context"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
	gocache "github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
)

// Manager wraps gocache cache manager
type Manager struct {
	cache gocache.CacheInterface[[]model.ServerWrapper]
	ttl   time.Duration
}

// NewManager creates a new cache manager
func NewManager(cacheImpl gocache.CacheInterface[[]model.ServerWrapper], ttl time.Duration) *Manager {
	return &Manager{
		cache: cacheImpl,
		ttl:   ttl,
	}
}

// Get retrieves cached servers for a repo ref
func (m *Manager) Get(ctx context.Context, ref model.RepoRef) ([]model.ServerWrapper, error) {
	return m.cache.Get(ctx, ref.CacheKey())
}

// GetWithTTL retrieves cached servers for a repo ref with remaining TTL
// Falls back to Get if the cache doesn't support TTL (returns 0 duration)
func (m *Manager) GetWithTTL(ctx context.Context, ref model.RepoRef) ([]model.ServerWrapper, time.Duration, error) {
	// Try to use SetterCacheInterface if available (has GetWithTTL)
	if setterCache, ok := m.cache.(gocache.SetterCacheInterface[[]model.ServerWrapper]); ok {
		return setterCache.GetWithTTL(ctx, ref.CacheKey())
	}
	// Fallback to regular Get (returns 0 duration for TTL)
	result, err := m.cache.Get(ctx, ref.CacheKey())
	return result, 0, err
}

// Set stores servers in cache for a repo ref
func (m *Manager) Set(ctx context.Context, ref model.RepoRef, wrappers []model.ServerWrapper) error {
	return m.cache.Set(ctx, ref.CacheKey(), wrappers, store.WithExpiration(m.ttl))
}

// Delete removes cached servers for a repo ref
func (m *Manager) Delete(ctx context.Context, ref model.RepoRef) error {
	return m.cache.Delete(ctx, ref.CacheKey())
}
