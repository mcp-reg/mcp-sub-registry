package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
	lib_store "github.com/eko/gocache/lib/v4/store"
)

// FilesystemStore implements eko/gocache StoreInterface for filesystem persistence
type FilesystemStore struct {
	baseDir string
}

// NewFilesystemStore creates a new filesystem cache store
func NewFilesystemStore(baseDir string) (*FilesystemStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &FilesystemStore{baseDir: baseDir}, nil
}

// Get retrieves value from filesystem
func (s *FilesystemStore) Get(ctx context.Context, key any) (any, error) {
	keyStr, ok := key.(string)
	if !ok {
		return nil, lib_store.NotFoundWithCause(errors.New("key must be string"))
	}

	path := s.keyToPath(keyStr)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, lib_store.NotFoundWithCause(errors.New("value not found in filesystem store"))
		}
		return nil, err
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		_ = os.Remove(path) // Best effort cleanup
		return nil, lib_store.NotFoundWithCause(errors.New("value not found in filesystem store"))
	}

	return entry.Value, nil
}

// GetWithTTL retrieves value from filesystem with remaining TTL
func (s *FilesystemStore) GetWithTTL(ctx context.Context, key any) (any, time.Duration, error) {
	keyStr, ok := key.(string)
	if !ok {
		return nil, 0, lib_store.NotFoundWithCause(errors.New("key must be string"))
	}

	path := s.keyToPath(keyStr)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, lib_store.NotFoundWithCause(errors.New("value not found in filesystem store"))
		}
		return nil, 0, err
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, 0, err
	}

	// Check expiration
	now := time.Now()
	if now.After(entry.ExpiresAt) {
		_ = os.Remove(path) // Best effort cleanup
		return nil, 0, lib_store.NotFoundWithCause(errors.New("value not found in filesystem store"))
	}

	remainingTTL := entry.ExpiresAt.Sub(now)
	return entry.Value, remainingTTL, nil
}

// Set stores value to filesystem
func (s *FilesystemStore) Set(ctx context.Context, key any, value any, options ...lib_store.Option) error {
	keyStr, ok := key.(string)
	if !ok {
		return errors.New("key must be string")
	}

	opts := lib_store.ApplyOptions(options...)
	ttl := opts.Expiration
	if ttl == 0 {
		ttl = 1 * time.Hour // Default
	}

	// Type assert value to []model.ServerWrapper for storage
	wrappers, ok := value.([]model.ServerWrapper)
	if !ok {
		return errors.New("value must be []model.ServerWrapper")
	}

	entry := cacheEntry{
		Value:     wrappers,
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	path := s.keyToPath(keyStr)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Delete removes value from filesystem
func (s *FilesystemStore) Delete(ctx context.Context, key any) error {
	keyStr, ok := key.(string)
	if !ok {
		return nil // Ignore invalid keys
	}

	path := s.keyToPath(keyStr)
	return os.Remove(path)
}

// Invalidate invalidates entries based on options (not fully implemented - clears all for simplicity)
func (s *FilesystemStore) Invalidate(ctx context.Context, options ...lib_store.InvalidateOption) error {
	// For simplicity, clear all entries when invalidate is called
	// A more sophisticated implementation could filter by tags if needed
	return s.Clear(ctx)
}

// Clear removes all cached entries
func (s *FilesystemStore) Clear(ctx context.Context) error {
	return os.RemoveAll(s.baseDir)
}

// GetType returns store type
func (s *FilesystemStore) GetType() string {
	return "filesystem"
}

// keyToPath converts cache key to filesystem path
func (s *FilesystemStore) keyToPath(key string) string {
	// Sanitize key for filesystem (replace / with _)
	safeKey := strings.ReplaceAll(key, "/", "_")
	safeKey = strings.ReplaceAll(safeKey, "\\", "_")
	safeKey = filepath.Base(safeKey) // Prevent directory traversal
	// Remove any remaining invalid characters
	safeKey = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, safeKey)
	return filepath.Join(s.baseDir, safeKey+".json")
}

type cacheEntry struct {
	Value     []model.ServerWrapper `json:"value"`
	ExpiresAt time.Time             `json:"expires_at"`
}
