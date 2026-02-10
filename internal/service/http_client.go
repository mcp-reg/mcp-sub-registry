package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
)

// HTTPClient handles HTTP requests to public MCP registries
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTP client with default settings
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchPublicServers fetches all servers from a public registry with pagination
func (c *HTTPClient) FetchPublicServers(ctx context.Context, baseURL string) ([]model.ServerWrapper, error) {
	var allWrappers []model.ServerWrapper
	var cursor string

	for {
		endpoint := fmt.Sprintf("%s/v0.1/servers", baseURL)
		if cursor != "" {
			endpoint = fmt.Sprintf("%s?cursor=%s", endpoint, url.QueryEscape(cursor))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch servers: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, &PublicRegistryError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("public registry returned status %d", resp.StatusCode),
			}
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}

		var listResp model.ServerListResponse
		if err := json.Unmarshal(body, &listResp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}

		// Store original meta before adding to result
		for i := range listResp.Servers {
			w := &listResp.Servers[i]
			w.Server.OrigChildMeta = copyMeta(w.Server.Meta)
			w.Server.OrigParentMeta = copyMeta(w.Meta)
		}

		// Keep full wrappers
		allWrappers = append(allWrappers, listResp.Servers...)

		// Check for next page
		if listResp.Metadata.NextCursor == nil || *listResp.Metadata.NextCursor == "" {
			break
		}
		cursor = *listResp.Metadata.NextCursor
	}

	return allWrappers, nil
}

// FetchPublicServerVersion fetches a specific server version from public registry
func (c *HTTPClient) FetchPublicServerVersion(ctx context.Context, baseURL, serverName, version string) (*model.ServerWrapper, error) {
	// URL-escape server name (e.g., "ai.exa/exa" -> "ai.exa%2Fexa")
	escapedName := url.PathEscape(serverName)
	endpoint := fmt.Sprintf("%s/v0.1/servers/%s/versions/%s", baseURL, escapedName, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch server version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &PublicRegistryError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("server %s version %s not found", serverName, version),
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &PublicRegistryError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("public registry returned status %d", resp.StatusCode),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var versionResp model.VersionResponse
	if err := json.Unmarshal(body, &versionResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Store original meta before returning
	versionResp.Server.OrigChildMeta = copyMeta(versionResp.Server.Meta)
	versionResp.Server.OrigParentMeta = copyMeta(versionResp.Meta)

	return &model.ServerWrapper{
		Server: versionResp.Server,
		Meta:   versionResp.Meta,
	}, nil
}

// PublicRegistryError represents an error from a public registry
type PublicRegistryError struct {
	StatusCode int
	Message    string
}

func (e *PublicRegistryError) Error() string {
	return e.Message
}
