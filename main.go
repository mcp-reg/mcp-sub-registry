package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/cache"
	"github.com/mcp-reg/mcp-sub-registry/internal/config"
	"github.com/mcp-reg/mcp-sub-registry/internal/handler"
	"github.com/mcp-reg/mcp-sub-registry/internal/model"
	"github.com/mcp-reg/mcp-sub-registry/internal/service"
	"github.com/mcp-reg/mcp-sub-registry/internal/validator"
	"github.com/mcp-reg/mcp-sub-registry/internal/version"
	gocache "github.com/eko/gocache/lib/v4/cache"
	goCacheStore "github.com/eko/gocache/store/go_cache/v4"
	goCache "github.com/patrickmn/go-cache"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	slog.Info("starting", "version", version.Info())

	// Initialize validator
	v, err := validator.NewValidator()
	if err != nil {
		slog.Error("failed to initialize validator", "error", err)
		os.Exit(1)
	}

	// Initialize cache (if enabled)
	var cacheManager *cache.Manager
	if cfg.CacheEnabled {
		// In-memory store using go-cache
		goCacheClient := goCache.New(cfg.CacheTTL, 10*time.Minute)
		memoryStore := goCacheStore.NewGoCache(goCacheClient)

		// Filesystem store
		fsStore, err := cache.NewFilesystemStore(cfg.CacheBaseDir)
		if err != nil {
			slog.Warn("failed to initialize filesystem cache, using memory only", "error", err)
			memoryCache := gocache.New[[]model.ServerWrapper](memoryStore)
			cacheManager = cache.NewManager(memoryCache, cfg.CacheTTL)
		} else {
			// Chain stores: memory first, filesystem fallback
			memoryCache := gocache.New[[]model.ServerWrapper](memoryStore)
			fsCache := gocache.New[[]model.ServerWrapper](fsStore)
			chainCache := gocache.NewChain(memoryCache, fsCache)
			cacheManager = cache.NewManager(chainCache, cfg.CacheTTL)
		}
		slog.Info("cache initialized", "ttl", cfg.CacheTTL, "base_dir", cfg.CacheBaseDir)
	} else {
		slog.Info("cache disabled")
	}

	// Initialize services
	ghClient := service.NewGitHubClient(cfg.GitHubAPIBase, cfg.GitHubToken)
	if cfg.GitHubToken != "" {
		slog.Info("GitHub API authentication enabled")
	} else {
		slog.Info("GitHub API authentication disabled (set GITHUB_TOKEN to enable)")
	}
	httpClient := service.NewHTTPClient()
	regService := service.NewRegistryService(ghClient, httpClient, v, cacheManager)

	// Initialize handlers
	h := handler.NewHandler(regService, cfg.BrowserCacheTTL)
	router := handler.NewRouter(h)

	addr := fmt.Sprintf(":%d", cfg.Port)
	slog.Info("starting server", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
