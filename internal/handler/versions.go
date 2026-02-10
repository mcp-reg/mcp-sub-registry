package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
	"github.com/go-chi/chi/v5"
)

// GetLatestVersion handles GET /{org}/{repo}/{branch}/v0.1/servers/{serverName}/versions/latest
func (h *Handler) GetLatestVersion(w http.ResponseWriter, r *http.Request) {
	ref := extractRepoRef(r)
	serverName, _ := url.PathUnescape(chi.URLParam(r, "serverName"))

	slog.Info("get latest version request",
		"org", ref.Org,
		"repo", ref.Repo,
		"branch", ref.Branch,
		"server", serverName,
	)

	wrappers, err := h.registry.FetchRegistry(r.Context(), ref)
	if err != nil {
		h.handleGitHubError(w, err, ref)
		return
	}

	latest := h.registry.FindLatestVersion(wrappers, serverName)
	if latest == nil {
		h.writeError(w, http.StatusNotFound,
			"Server '"+serverName+"' not found in registry")
		return
	}

	// Add isLatestVersion to meta
	addIsLatestVersion(latest)

	resp := model.VersionResponse{
		Server: latest.Server,
		Meta:   latest.Meta,
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetSpecificVersion handles GET /{org}/{repo}/{branch}/v0.1/servers/{serverName}/versions/{version}
func (h *Handler) GetSpecificVersion(w http.ResponseWriter, r *http.Request) {
	ref := extractRepoRef(r)
	serverName, _ := url.PathUnescape(chi.URLParam(r, "serverName"))
	version := chi.URLParam(r, "version")

	slog.Info("get specific version request",
		"org", ref.Org,
		"repo", ref.Repo,
		"branch", ref.Branch,
		"server", serverName,
		"version", version,
	)

	wrappers, err := h.registry.FetchRegistry(r.Context(), ref)
	if err != nil {
		h.handleGitHubError(w, err, ref)
		return
	}

	wrapper := h.registry.FindSpecificVersion(wrappers, serverName, version)
	if wrapper == nil {
		// Check if server exists at all
		if h.registry.FindServer(wrappers, serverName) == nil {
			h.writeError(w, http.StatusNotFound,
				"Server '"+serverName+"' not found in registry")
			return
		}
		h.writeError(w, http.StatusNotFound,
			"Version '"+version+"' not found for server '"+serverName+"'")
		return
	}

	// Check if this is the latest version
	latest := h.registry.FindLatestVersion(wrappers, serverName)
	isLatest := latest != nil && latest.Server.Version == wrapper.Server.Version

	if isLatest {
		addIsLatestVersion(wrapper)
	}

	resp := model.VersionResponse{
		Server: wrapper.Server,
		Meta:   wrapper.Meta,
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListVersions handles GET /{org}/{repo}/{branch}/v0.1/servers/{serverName}/versions
func (h *Handler) ListVersions(w http.ResponseWriter, r *http.Request) {
	ref := extractRepoRef(r)
	serverName, _ := url.PathUnescape(chi.URLParam(r, "serverName"))

	slog.Info("list versions request",
		"org", ref.Org,
		"repo", ref.Repo,
		"branch", ref.Branch,
		"server", serverName,
	)

	wrappers, err := h.registry.FetchRegistry(r.Context(), ref)
	if err != nil {
		h.handleGitHubError(w, err, ref)
		return
	}

	versions := h.registry.FindServersByName(wrappers, serverName)
	if len(versions) == 0 {
		h.writeError(w, http.StatusNotFound,
			"Server '"+serverName+"' not found in registry")
		return
	}

	// Sort by semver descending (latest first)
	h.registry.SortByVersionDescending(versions)

	// Mark the first one (latest) with isLatest flag
	addIsLatestVersion(&versions[0])

	resp := model.ServerListResponse{
		Servers: versions,
		Metadata: model.ListMetadata{
			Count: len(versions),
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

// addIsLatestVersion adds isLatestVersion flag to official meta
func addIsLatestVersion(wrapper *model.ServerWrapper) {
	if wrapper.Meta == nil {
		wrapper.Meta = make(model.ServerMeta)
	}
	official, ok := wrapper.Meta["io.modelcontextprotocol.registry/official"].(map[string]interface{})
	if !ok {
		official = map[string]interface{}{"status": "active"}
	}
	official["isLatestVersion"] = true
	wrapper.Meta["io.modelcontextprotocol.registry/official"] = official
}
