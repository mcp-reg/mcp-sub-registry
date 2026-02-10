package handler

import (
	"log/slog"
	"net/http"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
)

// RefreshCache handles POST /{org}/{repo}/{branch}/refresh
func (h *Handler) RefreshCache(w http.ResponseWriter, r *http.Request) {
	ref := extractRepoRef(r)

	slog.Info("refresh cache request",
		"org", ref.Org,
		"repo", ref.Repo,
		"branch", ref.Branch,
	)

	// Invalidate cache first, then re-fetch
	if h.registry != nil {
		// Delete existing cache entry
		if err := h.registry.InvalidateCache(r.Context(), ref); err != nil {
			slog.Warn("failed to invalidate cache",
				"org", ref.Org,
				"repo", ref.Repo,
				"branch", ref.Branch,
				"error", err,
			)
		}

		// Fetch fresh data (will populate cache)
		_, err := h.registry.FetchRegistry(r.Context(), ref)
		if err != nil {
			h.handleGitHubError(w, err, ref)
			return
		}
	}

	writeJSON(w, http.StatusOK, model.RefreshResponse{
		Message:   "Cache invalidated and refreshed",
		Refreshed: true,
	})
}
