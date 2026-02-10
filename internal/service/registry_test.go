package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
)

func TestRegistryService_FetchRegistry_Success(t *testing.T) {
	registry := model.RegistryFile{
		Registries: []model.RegistryEntry{
			{
				Name: "private",
				Type: "private",
				ServersRelativePath: []string{
					"servers/a/server.json",
					"servers/b/server.json",
				},
			},
		},
	}
	regJSON, _ := json.Marshal(registry)

	serverA := model.Server{Name: "test/a", Description: "Server A", Version: "1.0.0"}
	serverB := model.Server{Name: "test/b", Description: "Server B", Version: "2.0.0"}
	serverAJSON, _ := json.Marshal(serverA)
	serverBJSON, _ := json.Marshal(serverB)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var content []byte
		switch {
		case r.URL.Path == "/repos/org/repo/contents/registry.json":
			content = regJSON
		case r.URL.Path == "/repos/org/repo/contents/servers/a/server.json":
			content = serverAJSON
		case r.URL.Path == "/repos/org/repo/contents/servers/b/server.json":
			content = serverBJSON
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := GitHubContentsResponse{
			Content:  base64.StdEncoding.EncodeToString(content),
			Encoding: "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ghClient := NewGitHubClient(server.URL, "")
	regService := NewRegistryService(ghClient, nil, nil, nil)

	ref := model.RepoRef{Org: "org", Repo: "repo", Branch: "main"}
	servers, err := regService.FetchRegistry(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(servers))
	}

	if servers[0].Server.Name != "test/a" {
		t.Errorf("expected first server name 'test/a', got '%s'", servers[0].Server.Name)
	}
}

func TestRegistryService_FetchRegistry_MissingRegistryJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	ghClient := NewGitHubClient(server.URL, "")
	regService := NewRegistryService(ghClient, nil, nil, nil)

	ref := model.RepoRef{Org: "org", Repo: "repo", Branch: "main"}
	_, err := regService.FetchRegistry(context.Background(), ref)

	regErr, ok := err.(*RegistryError)
	if !ok {
		t.Fatalf("expected RegistryError, got %T", err)
	}
	if regErr.Code != 400 {
		t.Errorf("expected code 400, got %d", regErr.Code)
	}
}

func TestRegistryService_FetchRegistry_NoServersInRegistry(t *testing.T) {
	// Registry with no valid entries (public entry without servers field)
	registry := model.RegistryFile{
		Registries: []model.RegistryEntry{
			{Name: "public", URL: "https://example.com"}, // Missing servers field
		},
	}
	regJSON, _ := json.Marshal(registry)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := GitHubContentsResponse{
			Content:  base64.StdEncoding.EncodeToString(regJSON),
			Encoding: "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ghClient := NewGitHubClient(server.URL, "")
	regService := NewRegistryService(ghClient, nil, nil, nil)

	ref := model.RepoRef{Org: "org", Repo: "repo", Branch: "main"}
	_, err := regService.FetchRegistry(context.Background(), ref)

	regErr, ok := err.(*RegistryError)
	if !ok {
		t.Fatalf("expected RegistryError, got %T", err)
	}
	if regErr.Code != 400 {
		t.Errorf("expected code 400, got %d", regErr.Code)
	}
}

func TestRegistryService_FilterServers(t *testing.T) {
	wrappers := []model.ServerWrapper{
		{Server: model.Server{Name: "test/postgres", Description: "PostgreSQL", Version: "1.0.0"}},
		{Server: model.Server{Name: "test/mysql", Description: "MySQL", Version: "1.0.0"}},
		{Server: model.Server{Name: "test/postgres-backup", Description: "Backup", Version: "1.0.0"}},
	}

	svc := &RegistryService{}

	// Test search
	filtered := svc.FilterServers(wrappers, "postgres")
	if len(filtered) != 2 {
		t.Errorf("expected 2 matches for 'postgres', got %d", len(filtered))
	}

	// Test case insensitivity
	filtered = svc.FilterServers(wrappers, "POSTGRES")
	if len(filtered) != 2 {
		t.Errorf("expected 2 matches for 'POSTGRES', got %d", len(filtered))
	}

	// Test empty search
	filtered = svc.FilterServers(wrappers, "")
	if len(filtered) != 3 {
		t.Errorf("expected 3 servers for empty search, got %d", len(filtered))
	}
}

func TestRegistryService_PaginateServers(t *testing.T) {
	wrappers := make([]model.ServerWrapper, 10)
	for i := 0; i < 10; i++ {
		wrappers[i] = model.ServerWrapper{
			Server: model.Server{
				Name:        fmt.Sprintf("test/server%d", i),
				Description: "Test",
				Version:     "1.0.0",
			},
		}
	}

	svc := &RegistryService{}

	// First page
	page1, cursor := svc.PaginateServers(wrappers, "", 3)
	if len(page1) != 3 {
		t.Errorf("expected 3 servers, got %d", len(page1))
	}
	if cursor == nil {
		t.Error("expected cursor, got nil")
	}

	// Second page - cursor points to first item of next page
	// Cursor "test/server3:1.0.0" means page starts WITH server3
	page2, cursor := svc.PaginateServers(wrappers, *cursor, 3)
	if len(page2) != 3 {
		t.Errorf("expected 3 servers, got %d", len(page2))
	}
	if page2[0].Server.Name != "test/server3" {
		t.Errorf("expected first server on page 2 to be 'test/server3', got '%s'", page2[0].Server.Name)
	}

	// Last page - cursor test/server8 means start WITH server8, returns server8 and server9
	page4, cursor := svc.PaginateServers(wrappers, "test/server8:1.0.0", 3)
	if len(page4) != 2 {
		t.Errorf("expected 2 servers on last page, got %d", len(page4))
	}
	if cursor != nil {
		t.Errorf("expected nil cursor on last page, got %s", *cursor)
	}
}

func TestRegistryService_FindLatestVersion(t *testing.T) {
	wrappers := []model.ServerWrapper{
		{Server: model.Server{Name: "test/server", Version: "1.0.0"}},
		{Server: model.Server{Name: "test/server", Version: "2.0.0"}},
		{Server: model.Server{Name: "test/server", Version: "1.5.0"}},
		{Server: model.Server{Name: "test/server", Version: "2.0.0-beta"}},
		{Server: model.Server{Name: "other/server", Version: "3.0.0"}},
	}

	svc := &RegistryService{}
	latest := svc.FindLatestVersion(wrappers, "test/server")

	if latest == nil {
		t.Fatal("expected to find latest version")
	}
	if latest.Server.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", latest.Server.Version)
	}
}

func TestRegistryService_FindSpecificVersion(t *testing.T) {
	wrappers := []model.ServerWrapper{
		{Server: model.Server{Name: "test/server", Version: "1.0.0"}},
		{Server: model.Server{Name: "test/server", Version: "2.0.0"}},
	}

	svc := &RegistryService{}

	found := svc.FindSpecificVersion(wrappers, "test/server", "1.0.0")
	if found == nil {
		t.Fatal("expected to find version 1.0.0")
	}

	notFound := svc.FindSpecificVersion(wrappers, "test/server", "3.0.0")
	if notFound != nil {
		t.Error("expected nil for non-existent version")
	}
}

func TestRegistryService_FetchRegistry_PublicWithAllServers(t *testing.T) {
	// Mock public registry server
	publicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v0.1/servers" {
			resp := model.ServerListResponse{
				Servers: []model.ServerWrapper{
					{Server: model.Server{Name: "public/server1", Description: "Server 1", Version: "1.0.0"}},
					{Server: model.Server{Name: "public/server2", Description: "Server 2", Version: "2.0.0"}},
				},
				Metadata: model.ListMetadata{Count: 2, NextCursor: nil},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer publicServer.Close()

	// Registry with public entry using servers: "*"
	registry := model.RegistryFile{
		Registries: []model.RegistryEntry{
			{Name: "public", URL: publicServer.URL, Servers: "*"},
		},
	}
	regJSON, _ := json.Marshal(registry)

	// Mock GitHub server
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := GitHubContentsResponse{
			Content:  base64.StdEncoding.EncodeToString(regJSON),
			Encoding: "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ghServer.Close()

	ghClient := NewGitHubClient(ghServer.URL, "")
	httpClient := NewHTTPClient()
	regService := NewRegistryService(ghClient, httpClient, nil, nil)

	ref := model.RepoRef{Org: "org", Repo: "repo", Branch: "main"}
	servers, err := regService.FetchRegistry(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(servers))
	}

	if servers[0].Server.Name != "public/server1" {
		t.Errorf("expected first server name 'public/server1', got '%s'", servers[0].Server.Name)
	}
}

func TestRegistryService_FetchRegistry_PublicWithSpecificServers(t *testing.T) {
	// Mock public registry server
	publicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// URL path after server parses it: /v0.1/servers/ai.exa%2Fexa/versions/latest
		// Note: httptest server receives decoded path, so we check for decoded version
		if r.URL.Path == "/v0.1/servers/ai.exa%2Fexa/versions/latest" || r.URL.RawPath == "/v0.1/servers/ai.exa%2Fexa/versions/latest" {
			resp := model.VersionResponse{
				Server: model.Server{Name: "ai.exa/exa", Description: "Exa Server", Version: "1.0.0"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("404 for path: %s (raw: %s)", r.URL.Path, r.URL.RawPath)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer publicServer.Close()

	// Registry with public entry using specific servers map
	registry := model.RegistryFile{
		Registries: []model.RegistryEntry{
			{Name: "MCP Official", URL: publicServer.URL, Servers: map[string]interface{}{"ai.exa/exa": "latest"}},
		},
	}
	regJSON, _ := json.Marshal(registry)

	// Mock GitHub server
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := GitHubContentsResponse{
			Content:  base64.StdEncoding.EncodeToString(regJSON),
			Encoding: "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ghServer.Close()

	ghClient := NewGitHubClient(ghServer.URL, "")
	httpClient := NewHTTPClient()
	regService := NewRegistryService(ghClient, httpClient, nil, nil)

	ref := model.RepoRef{Org: "org", Repo: "repo", Branch: "main"}
	servers, err := regService.FetchRegistry(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(servers))
	}

	if servers[0].Server.Name != "ai.exa/exa" {
		t.Errorf("expected server name 'ai.exa/exa', got '%s'", servers[0].Server.Name)
	}
}

func TestRegistryService_FetchRegistry_MixedRegistries(t *testing.T) {
	// Mock public registry server
	publicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v0.1/servers/public/test/versions/latest" {
			resp := model.VersionResponse{
				Server: model.Server{Name: "public/test", Description: "Public Server", Version: "1.0.0"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer publicServer.Close()

	privateServer := model.Server{Name: "private/server", Description: "Private Server", Version: "1.0.0"}
	privateServerJSON, _ := json.Marshal(privateServer)

	// Registry with both private and public entries
	registry := model.RegistryFile{
		Registries: []model.RegistryEntry{
			{Name: "MCP Official", URL: publicServer.URL, Servers: map[string]interface{}{"public/test": "latest"}},
			{Name: "private", Type: "private", ServersRelativePath: []string{"servers/private/server.json"}},
		},
	}
	regJSON, _ := json.Marshal(registry)

	// Mock GitHub server
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var content []byte
		switch r.URL.Path {
		case "/repos/org/repo/contents/registry.json":
			content = regJSON
		case "/repos/org/repo/contents/servers/private/server.json":
			content = privateServerJSON
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := GitHubContentsResponse{
			Content:  base64.StdEncoding.EncodeToString(content),
			Encoding: "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ghServer.Close()

	ghClient := NewGitHubClient(ghServer.URL, "")
	httpClient := NewHTTPClient()
	regService := NewRegistryService(ghClient, httpClient, nil, nil)

	ref := model.RepoRef{Org: "org", Repo: "repo", Branch: "main"}
	servers, err := regService.FetchRegistry(context.Background(), ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(servers) != 2 {
		t.Errorf("expected 2 servers (1 public + 1 private), got %d", len(servers))
	}
}

func TestRegistryService_FetchRegistry_PublicRegistryError(t *testing.T) {
	// Mock public registry server that returns error
	publicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer publicServer.Close()

	registry := model.RegistryFile{
		Registries: []model.RegistryEntry{
			{Name: "failing", URL: publicServer.URL, Servers: "*"},
		},
	}
	regJSON, _ := json.Marshal(registry)

	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := GitHubContentsResponse{
			Content:  base64.StdEncoding.EncodeToString(regJSON),
			Encoding: "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ghServer.Close()

	ghClient := NewGitHubClient(ghServer.URL, "")
	httpClient := NewHTTPClient()
	regService := NewRegistryService(ghClient, httpClient, nil, nil)

	ref := model.RepoRef{Org: "org", Repo: "repo", Branch: "main"}
	_, err := regService.FetchRegistry(context.Background(), ref)

	regErr, ok := err.(*RegistryError)
	if !ok {
		t.Fatalf("expected RegistryError, got %T", err)
	}
	if regErr.Code != 502 {
		t.Errorf("expected code 502, got %d", regErr.Code)
	}
}
