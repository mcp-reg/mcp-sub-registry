package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
	"github.com/mcp-reg/mcp-sub-registry/internal/service"
	"github.com/go-chi/chi/v5"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	registry        *service.RegistryService
	browserCacheTTL time.Duration
}

// NewHandler creates a new handler
func NewHandler(registry *service.RegistryService, browserCacheTTL time.Duration) *Handler {
	return &Handler{
		registry:        registry,
		browserCacheTTL: browserCacheTTL,
	}
}

// extractRepoRef extracts org/repo/branch from URL params
func extractRepoRef(r *http.Request) model.RepoRef {
	return model.RepoRef{
		Org:    chi.URLParam(r, "org"),
		Repo:   chi.URLParam(r, "repo"),
		Branch: chi.URLParam(r, "branch"),
	}
}

// extractListParams extracts query parameters for list endpoint
func extractListParams(r *http.Request) model.ListParams {
	limit := model.DefaultLimit
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	return model.ListParams{
		Cursor:       r.URL.Query().Get("cursor"),
		Limit:        limit,
		Search:       r.URL.Query().Get("search"),
		UpdatedSince: r.URL.Query().Get("updated_since"),
		Version:      r.URL.Query().Get("version"),
	}
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeCachedJSON writes a JSON response with Cache-Control header for browser caching
func (h *Handler) writeCachedJSON(w http.ResponseWriter, status int, data interface{}) {
	if h.browserCacheTTL > 0 {
		maxAge := int(h.browserCacheTTL.Seconds())
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	}
	writeJSON(w, status, data)
}

// writeError writes an error response
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, model.ErrorResponse{
		Error: message,
	})
}

// handleGitHubError converts GitHub errors to HTTP responses
func (h *Handler) handleGitHubError(w http.ResponseWriter, err error, ref model.RepoRef) {
	if ghErr, ok := err.(*service.GitHubError); ok {
		switch {
		case ghErr.IsNotFound():
			h.writeError(w, http.StatusNotFound,
				"Repository "+ref.Org+"/"+ref.Repo+" not found")
		case ghErr.IsRateLimited():
			h.writeError(w, http.StatusTooManyRequests,
				"GitHub API rate limit exceeded")
		default:
			h.writeError(w, http.StatusBadGateway,
				"Failed to fetch from GitHub: "+ghErr.Message)
		}
		return
	}

	if regErr, ok := err.(*service.RegistryError); ok {
		h.writeError(w, regErr.Code, regErr.Message)
		return
	}

	slog.Error("unexpected error", "error", err)
	h.writeError(w, http.StatusServiceUnavailable, "Service temporarily unavailable")
}
