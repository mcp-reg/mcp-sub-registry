package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
	"github.com/mcp-reg/mcp-sub-registry/internal/service"
	"github.com/go-chi/chi/v5"
)

// setupTestHandler creates a handler with mocked GitHub
func setupTestHandler(t *testing.T, mockResponses map[string]interface{}) (*Handler, *chi.Mux) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for path, resp := range mockResponses {
			if r.URL.Path == path {
				content, _ := json.Marshal(resp)
				ghResp := service.GitHubContentsResponse{
					Content:  base64.StdEncoding.EncodeToString(content),
					Encoding: "base64",
				}
				json.NewEncoder(w).Encode(ghResp)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(mockServer.Close)

	ghClient := service.NewGitHubClient(mockServer.URL, "")
	regService := service.NewRegistryService(ghClient, nil, nil, nil)
	h := NewHandler(regService, 5*time.Minute)

	return h, NewRouter(h)
}

func TestHealth(t *testing.T) {
	h := &Handler{}
	router := chi.NewRouter()
	router.Get("/health", h.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp model.HealthResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
}

func TestListServers(t *testing.T) {
	registry := model.RegistryFile{
		Registries: []model.RegistryEntry{
			{
				Name: "private",
				Type: "private",
				ServersRelativePath: []string{"server.json"},
			},
		},
	}
	server := model.Server{
		Name:        "test/server",
		Description: "Test server",
		Version:     "1.0.0",
	}

	_, router := setupTestHandler(t, map[string]interface{}{
		"/repos/org/repo/contents/registry.json": registry,
		"/repos/org/repo/contents/server.json":   server,
	})

	req := httptest.NewRequest("GET", "/org/repo/main/v0.1/servers", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp model.ServerListResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(resp.Servers))
	}
	if resp.Servers[0].Server.Name != "test/server" {
		t.Errorf("expected name 'test/server', got '%s'", resp.Servers[0].Server.Name)
	}
}

func TestListServers_NotFound(t *testing.T) {
	h := NewHandler(
		service.NewRegistryService(service.NewGitHubClient("http://invalid", ""), nil, nil, nil),
		5*time.Minute,
	)
	router := NewRouter(h)

	// Use a mock server that returns 404
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	ghClient := service.NewGitHubClient(mockServer.URL, "")
	h.registry = service.NewRegistryService(ghClient, nil, nil, nil)

	req := httptest.NewRequest("GET", "/org/repo/main/v0.1/servers", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestCORSHeaders(t *testing.T) {
	h := &Handler{}
	router := NewRouter(h)

	req := httptest.NewRequest("OPTIONS", "/org/repo/main/v0.1/servers", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS origin *, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}
