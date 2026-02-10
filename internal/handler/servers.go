package handler

import (
	"log/slog"
	"net/http"
)

// ListServers handles GET /{org}/{repo}/{branch}/v0.1/servers
func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	ref := extractRepoRef(r)
	params := extractListParams(r)

	slog.Info("list servers request",
		"org", ref.Org,
		"repo", ref.Repo,
		"branch", ref.Branch,
		"search", params.Search,
	)

	resp, err := h.registry.QueryServers(r.Context(), ref, params)
	if err != nil {
		h.handleGitHubError(w, err, ref)
		return
	}

	h.writeCachedJSON(w, http.StatusOK, resp)
}
