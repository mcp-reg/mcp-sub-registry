package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
)

// GitHubClient fetches files from GitHub Contents API
type GitHubClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// GitHubContentsResponse is the GitHub API response structure
type GitHubContentsResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Message  string `json:"message,omitempty"` // Error message
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient(baseURL, token string) *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		token:      token,
	}
}

// FetchFile fetches a file from a GitHub repo
func (g *GitHubClient) FetchFile(ctx context.Context, ref model.RepoRef, path string) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s",
		g.baseURL, ref.Org, ref.Repo, path, ref.Branch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// Continue processing
	case http.StatusNotFound:
		return nil, &GitHubError{StatusCode: 404, Message: fmt.Sprintf("file not found: %s", path)}
	case http.StatusForbidden:
		// Check if rate limited
		if strings.Contains(string(body), "rate limit") {
			return nil, &GitHubError{StatusCode: 429, Message: "GitHub API rate limit exceeded"}
		}
		return nil, &GitHubError{StatusCode: 403, Message: "access forbidden"}
	default:
		return nil, &GitHubError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("GitHub API error: %s", string(body))}
	}

	var contents GitHubContentsResponse
	if err := json.Unmarshal(body, &contents); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if contents.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected encoding: %s", contents.Encoding)
	}

	// GitHub returns base64 with newlines, remove them
	cleanContent := strings.ReplaceAll(contents.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(cleanContent)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	return decoded, nil
}

// GitHubError represents a GitHub API error
type GitHubError struct {
	StatusCode int
	Message    string
}

func (e *GitHubError) Error() string {
	return e.Message
}

// IsNotFound returns true if error is 404
func (e *GitHubError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsRateLimited returns true if error is rate limit
func (e *GitHubError) IsRateLimited() bool {
	return e.StatusCode == 429
}
