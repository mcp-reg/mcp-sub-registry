package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port          int
	GitHubAPIBase string
	GitHubToken   string
	// Cache configuration
	CacheTTL     time.Duration
	CacheBaseDir string
	CacheEnabled bool

	// Browser cache configuration
	BrowserCacheTTL time.Duration
}

func Load() *Config {
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	githubAPI := "https://api.github.com"
	if g := os.Getenv("GITHUB_API_BASE"); g != "" {
		githubAPI = g
	}

	githubToken := os.Getenv("GITHUB_TOKEN")

	// Cache configuration
	cacheTTL := 1 * time.Hour
	if ttlStr := os.Getenv("CACHE_TTL"); ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			cacheTTL = parsed
		}
	}

	cacheBaseDir := "/tmp/mcp-registry-cache"
	if dir := os.Getenv("CACHE_BASE_DIR"); dir != "" {
		cacheBaseDir = dir
	}

	cacheEnabled := true
	if enabled := os.Getenv("CACHE_ENABLED"); enabled == "false" {
		cacheEnabled = false
	}

	// Browser cache configuration
	browserCacheTTL := 5 * time.Minute
	if ttlStr := os.Getenv("BROWSER_CACHE_TTL"); ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			browserCacheTTL = parsed
		}
	}

	return &Config{
		Port:            port,
		GitHubAPIBase:   githubAPI,
		GitHubToken:     githubToken,
		CacheTTL:        cacheTTL,
		CacheBaseDir:    cacheBaseDir,
		CacheEnabled:    cacheEnabled,
		BrowserCacheTTL: browserCacheTTL,
	}
}
