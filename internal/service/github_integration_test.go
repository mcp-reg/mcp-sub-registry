//go:build integration

package service

import (
	"context"
	"testing"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
)

func TestGitHubClient_RealRepo(t *testing.T) {
	client := NewGitHubClient("https://api.github.com", "")
	ref := model.RepoRef{
		Org:    "modelcontextprotocol",
		Repo:   "registry",
		Branch: "main",
	}

	// Fetch a known file from the MCP registry repo
	content, err := client.FetchFile(context.Background(), ref, "README.md")
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty content")
	}

	t.Logf("fetched %d bytes from real GitHub", len(content))
}
