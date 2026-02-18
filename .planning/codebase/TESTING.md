# Testing Patterns

**Analysis Date:** 2026-02-18

## Test Framework

**Runner:**
- Go: Built-in `testing` package (standard library)
- Config: Tests identified by `_test.go` suffix or `//go:build e2e` tag
- Frontend: No JavaScript testing framework configured (ESLint only)

**Assertion Library:**
- Go: Simple comparison assertions; custom error messages with `t.Errorf()`, `t.Fatalf()`

**Run Commands:**
```bash
make test              # Run all unit tests (go test -v -race ./...)
make test-e2e          # Run e2e tests (go test -v -tags=e2e ./...)
make lint              # Go linting (go vet + golangci-lint run)
cd web && npm run lint # ESLint check (frontend only)
```

## Test File Organization

**Location:**
- Go: Co-located with source code (same package, `_test.go` suffix)
  - `internal/service/transformer.go` → `internal/service/transformer_test.go`
  - `internal/model/model.go` → `internal/model/model_test.go`
- Frontend: No dedicated test directory (no test framework configured)

**Naming:**
- Unit tests: `<source>_test.go` (e.g., `github_test.go`, `transformer_test.go`)
- E2E tests: Single file `e2e_test.go` at root with build tag
- Test functions: `Test<FunctionName>_<Scenario>` (e.g., `TestTransformWrapper_VSCode`)

**Structure:**
```
internal/
├── service/
│   ├── transformer.go
│   ├── transformer_test.go
│   ├── github.go
│   └── github_test.go
├── model/
│   ├── server.go
│   ├── model.go
│   └── model_test.go
```

## Test Structure

**Suite Organization:**
Go tests do not use formal test suites. Instead:
- Each test function is independent (no setup/teardown across tests)
- Subtests group related scenarios under single test using `t.Run()`

Example from `transformer_test.go` lines 147-187 (table-driven):
```go
func TestTransformWrapper_AddsSourceTracking(t *testing.T) {
	testCases := []struct {
		name         string
		sourceURL    string
		registryName string
	}{
		{"vscode", "https://api.mcp.github.com/registry", "vscode"},
		{"obot", "https://acornlabs.com/registry", "obot"},
		{"default", "https://example.com", "custom"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrapper := model.ServerWrapper{
				Server: model.Server{
					Name:    "test/server",
					Version: "1.0.0",
				},
			}
			result := TransformWrapper(tc.sourceURL, tc.registryName, wrapper)
			// assertions...
		})
	}
}
```

**Patterns:**

Setup (inline, no shared fixtures):
```go
// Create test data directly in test
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// mock response
}))
defer server.Close()
```

Teardown (cleanup via defer):
```go
defer server.Close()
```

Assertions (direct comparison):
```go
if len(result) != 2 {
	t.Errorf("expected 2 results, got %d", len(result))
}
if result.Server.Name != "test/server" {
	t.Errorf("expected name 'test/server', got '%s'", result.Server.Name)
}
```

## Mocking

**Framework:** Manual mocking using `httptest.NewServer()` for HTTP endpoints

**Patterns:**

Example from `github_test.go` lines 14-47:
```go
func TestGitHubClient_FetchFile_Success(t *testing.T) {
	expectedContent := `{"name": "test"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(expectedContent))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		expectedPath := "/repos/testorg/testrepo/contents/registry.json"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.URL.Query().Get("ref") != "main" {
			t.Errorf("expected ref=main, got %s", r.URL.Query().Get("ref"))
		}

		// Return response
		resp := GitHubContentsResponse{
			Content:  encoded,
			Encoding: "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "")
	content, err := client.FetchFile(context.Background(), ref, "registry.json")
	// assertions...
}
```

**What to Mock:**
- HTTP endpoints (via `httptest.NewServer()`)
- External APIs (replace with test server URL)
- Responses that require specific error conditions (404, 429, etc.)

**What NOT to Mock:**
- Core service logic (test actual functions)
- Data structures (create real instances)
- Standard library functions

## Fixtures and Factories

**Test Data:**
Created inline, not in separate fixtures. Example from `transformer_test.go` lines 11-25:
```go
func TestTransformWrapper_VSCode(t *testing.T) {
	wrapper := model.ServerWrapper{
		Server: model.Server{
			Name:        "test/server",
			Description: "Test server",
			Version:     "1.0.0",
			Meta: model.ServerMeta{
				"io.modelcontextprotocol.registry/publisher-provided": map[string]interface{}{
					"github": map[string]interface{}{
						"preferredImage": "https://example.com/icon.png",
						"displayName":    "Test Display Name",
					},
				},
			},
		},
	}
	// test logic
}
```

**Location:**
- Inline within test functions (no separate fixture files)
- Test data created with default values, modified as needed per test

## Coverage

**Requirements:** No enforced coverage targets (none configured)

**View Coverage:**
```bash
go test -cover ./...    # Show coverage summary
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Coverage tools not integrated into CI/CD pipeline.

## Test Types

**Unit Tests:**
- Scope: Individual functions (e.g., `TransformWrapper()`, `extractGitHubInfo()`)
- Approach: Direct function calls with controlled inputs, verify outputs
- Location: `<module>_test.go` files throughout `internal/`
- Examples: `transformer_test.go`, `model_test.go`, `github_test.go`

**Integration Tests:**
- Scope: Service interactions (e.g., `registry_test.go`, `github_integration_test.go`)
- Approach: Mock external HTTP, test service layer orchestration
- Location: Same package as integration point (e.g., `internal/service/`)
- Example from `github_test.go` lines 98-130 (handles GitHub's base64 format with newlines):
```go
func TestGitHubClient_FetchFile_Base64WithNewlines(t *testing.T) {
	expectedContent := `{"name": "test with longer content..."}`
	encoded := base64.StdEncoding.EncodeToString([]byte(expectedContent))
	// GitHub adds newlines every 60 chars
	encodedWithNewlines := ""
	for i := 0; i < len(encoded); i += 60 {
		// ... add newlines
	}
	// test GitHub's actual response format
}
```

**E2E Tests:**
- Framework: Go `testing` package with `//go:build e2e` tag
- Config: File `e2e_test.go` at package `main` (root of project)
- Trigger: `make test-e2e` compiles with `-tags=e2e` flag
- Scope: Full HTTP handler flow against real registry data
- Approach: Use `httptest.NewRecorder()` for HTTP testing, hit real handlers

Example from `e2e_test.go` lines 18-48:
```go
func TestE2E_FullFlow(t *testing.T) {
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

	t.Run("ListServers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/"+org+"/"+repo+"/main/v0.1/servers", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		// ... assertions
	})
}
```

## Common Patterns

**Async Testing:**
Not applicable in Go unit tests (sync). E2E tests use `context.Background()` for API calls:
```go
content, err := client.FetchFile(context.Background(), ref, "registry.json")
```

**Error Testing:**
Test error conditions by mocking error responses:

Example from `github_test.go` lines 49-71 (404 error):
```go
func TestGitHubClient_FetchFile_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "")
	_, err := client.FetchFile(context.Background(), ref, "missing.json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	ghErr, ok := err.(*GitHubError)
	if !ok {
		t.Fatalf("expected GitHubError, got %T", err)
	}
	if !ghErr.IsNotFound() {
		t.Errorf("expected not found error, got %d", ghErr.StatusCode)
	}
}
```

Example from `github_test.go` lines 73-95 (rate limit):
```go
func TestGitHubClient_FetchFile_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "API rate limit exceeded"}`))
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "")
	_, err := client.FetchFile(context.Background(), ref, "registry.json")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	ghErr, ok := err.(*GitHubError)
	if !ok {
		t.Fatalf("expected GitHubError, got %T", err)
	}
	if !ghErr.IsRateLimited() {
		t.Errorf("expected rate limit error, got %d", ghErr.StatusCode)
	}
}
```

**Immutability Testing:**
Test that functions don't modify input (important for data transformation):

Example from `transformer_test.go` lines 190-218:
```go
func TestTransformWrapper_DoesNotModifyOriginal(t *testing.T) {
	original := model.ServerWrapper{
		Server: model.Server{
			Name:        "test/server",
			Version:     "1.0.0",
		},
		Meta: model.ServerMeta{
			"existing": "data",
		},
	}

	result := TransformWrapper("https://example.com", "test", original)

	// Verify original is not modified
	if _, exists := original.Meta["io.mcpregistry/source"]; exists {
		t.Error("original should not be modified")
	}

	// Verify result has new metadata
	if _, exists := result.Meta["io.mcpregistry/source"]; !exists {
		t.Error("result should have source tracking")
	}

	// Verify original data is preserved in result
	if result.Meta["existing"] != "data" {
		t.Error("original metadata should be preserved in result")
	}
}
```

---

*Testing analysis: 2026-02-18*
