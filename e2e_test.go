//go:build e2e

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/handler"
	"github.com/mcp-reg/mcp-sub-registry/internal/model"
	"github.com/mcp-reg/mcp-sub-registry/internal/service"
	"github.com/mcp-reg/mcp-sub-registry/internal/validator"
)

func TestE2E_FullFlow(t *testing.T) {
	// Setup service with real GitHub repository
	const (
		org  = "mcp-reg"
		repo = "mcp-registry-template"
	)

	ghClient := service.NewGitHubClient("https://api.github.com", "")
	httpClient := service.NewHTTPClient()
	v, _ := validator.NewValidator()
	regService := service.NewRegistryService(ghClient, httpClient, v, nil)
	h := handler.NewHandler(regService, "https://docs.example.com", 5*time.Minute)
	router := handler.NewRouter(h)

	// Test cases
	t.Run("ListServers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/"+org+"/"+repo+"/main/v0.1/servers", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp model.ServerListResponse
		json.NewDecoder(rec.Body).Decode(&resp)

		if len(resp.Servers) == 0 {
			t.Error("expected at least 1 server, got 0")
		}
	})

	t.Run("SearchServers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/"+org+"/"+repo+"/main/v0.1/servers?search=hello", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var resp model.ServerListResponse
		json.NewDecoder(rec.Body).Decode(&resp)

		if len(resp.Servers) == 0 {
			t.Error("expected at least 1 server matching 'hello', got 0")
		}
	})

	t.Run("SearchServersWithSearchComplex", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/"+org+"/"+repo+"/main/v0.1/servers?search=example/hello-world", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var resp model.ServerListResponse
		json.NewDecoder(rec.Body).Decode(&resp)

		if len(resp.Servers) == 0 {
			t.Error("expected at least 1 server matching 'hello', got 0")
		}
	})

	t.Run("GetLatestVersion", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/"+org+"/"+repo+"/main/v0.1/servers/example%2Fhello-world/versions/latest", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var resp model.VersionResponse
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp.Server.Version != "0.1.0" {
			t.Errorf("expected version 0.1.0, got %s", resp.Server.Version)
		}
	})

	t.Run("GetSpecificVersion", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/"+org+"/"+repo+"/main/v0.1/servers/example%2Fhello-world/versions/0.1.0", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("ServerNotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/"+org+"/"+repo+"/main/v0.1/servers/nonexistent/versions/latest", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/"+org+"/"+repo+"/main/v0.1/servers?limit=1", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var resp model.ServerListResponse
		json.NewDecoder(rec.Body).Decode(&resp)

		if len(resp.Servers) != 1 {
			t.Errorf("expected 1 server with limit=1, got %d", len(resp.Servers))
		}
		if resp.Metadata.NextCursor == nil {
			t.Error("expected nextCursor for paginated response")
		}
	})
}
