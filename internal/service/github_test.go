package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
)

func TestGitHubClient_FetchFile_Success(t *testing.T) {
	expectedContent := `{"name": "test"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(expectedContent))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		expectedPath := "/repos/testorg/testrepo/contents/registry.json"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.URL.Query().Get("ref") != "main" {
			t.Errorf("expected ref=main, got %s", r.URL.Query().Get("ref"))
		}

		resp := GitHubContentsResponse{
			Content:  encoded,
			Encoding: "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "")
	ref := model.RepoRef{Org: "testorg", Repo: "testrepo", Branch: "main"}

	content, err := client.FetchFile(context.Background(), ref, "registry.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("expected %s, got %s", expectedContent, string(content))
	}
}

func TestGitHubClient_FetchFile_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "")
	ref := model.RepoRef{Org: "testorg", Repo: "testrepo", Branch: "main"}

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

func TestGitHubClient_FetchFile_RateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "API rate limit exceeded"}`))
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "")
	ref := model.RepoRef{Org: "testorg", Repo: "testrepo", Branch: "main"}

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

func TestGitHubClient_FetchFile_Base64WithNewlines(t *testing.T) {
	expectedContent := `{"name": "test with longer content that spans multiple lines when base64 encoded"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(expectedContent))
	// GitHub adds newlines every 60 chars
	encodedWithNewlines := ""
	for i := 0; i < len(encoded); i += 60 {
		end := i + 60
		if end > len(encoded) {
			end = len(encoded)
		}
		encodedWithNewlines += encoded[i:end] + "\n"
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := GitHubContentsResponse{
			Content:  encodedWithNewlines,
			Encoding: "base64",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "")
	ref := model.RepoRef{Org: "testorg", Repo: "testrepo", Branch: "main"}

	content, err := client.FetchFile(context.Background(), ref, "registry.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("expected %s, got %s", expectedContent, string(content))
	}
}
