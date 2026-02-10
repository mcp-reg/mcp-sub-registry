package service

import (
	"strings"
	"time"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
)

// TransformWrapper transforms a wrapper based on source URL/name
// Returns a new ServerWrapper (does not modify original)
func TransformWrapper(sourceURL, registryName string, wrapper model.ServerWrapper) model.ServerWrapper {
	url := strings.ToLower(sourceURL)
	name := strings.ToLower(registryName)

	var result model.ServerWrapper

	if strings.Contains(url, "api.mcp.github.com") || (strings.Contains(url, "github.com") && strings.Contains(url, "mcp")) {
		result = transformVSCode(wrapper)
	} else if strings.Contains(url, "acornlabs") || name == "obot" {
		result = transformObot(wrapper)
	} else {
		result = transformDefault(wrapper)
	}

	// Fallback: extract GithubInfo from Repository if still nil
	if result.Server.GithubInfo == nil && result.Server.Repository != nil {
		result.Server.GithubInfo = extractGitHubInfoFromRepository(result.Server.Repository)
	}

	// Always add source tracking
	return addSourceTracking(result, sourceURL, registryName)
}

// transformVSCode handles VSCode/GitHub Copilot registry data
func transformVSCode(wrapper model.ServerWrapper) model.ServerWrapper {
	result := copyWrapper(wrapper)

	// Extract github metadata from server._meta
	github := extractGitHubMeta(result.Server.Meta)
	if github == nil {
		return result
	}

	// Fill missing icons from preferredImage
	if len(result.Server.Icons) == 0 {
		if preferredImage, ok := github["preferredImage"].(string); ok && preferredImage != "" {
			result.Server.Icons = []model.Icon{{Src: preferredImage}}
		}
	}

	// Fill missing title from displayName
	if result.Server.Title == "" {
		if displayName, ok := github["displayName"].(string); ok {
			result.Server.Title = displayName
		}
	}

	// Extract GithubInfo from github metadata fields
	if result.Server.GithubInfo == nil {
		result.Server.GithubInfo = extractGitHubInfo(github)
	}

	// Extract readme if present and not already set
	if result.Server.Readme == "" {
		if readme, ok := github["readme"].(string); ok {
			result.Server.Readme = readme
		}
	}

	return result
}

// transformObot handles Obot registry data
func transformObot(wrapper model.ServerWrapper) model.ServerWrapper {
	result := copyWrapper(wrapper)

	// Merge ai.obot/server namespace from wrapper._meta if present
	if obotMeta, ok := wrapper.Meta["ai.obot/server"].(map[string]interface{}); ok {
		if result.Meta == nil {
			result.Meta = make(model.ServerMeta)
		}
		result.Meta["ai.obot/server"] = obotMeta
	}

	return result
}

// transformDefault handles unknown registries
func transformDefault(wrapper model.ServerWrapper) model.ServerWrapper {
	result := copyWrapper(wrapper)

	// Ensure official status exists
	if result.Meta == nil {
		result.Meta = make(model.ServerMeta)
	}
	if _, exists := result.Meta["io.modelcontextprotocol.registry/official"]; !exists {
		result.Meta["io.modelcontextprotocol.registry/official"] = map[string]interface{}{
			"status": "active",
		}
	}

	return result
}

// addSourceTracking adds source info to wrapper _meta
func addSourceTracking(wrapper model.ServerWrapper, sourceURL, registryName string) model.ServerWrapper {
	if wrapper.Meta == nil {
		wrapper.Meta = make(model.ServerMeta)
	}
	wrapper.Meta["io.mcpregistry/source"] = map[string]interface{}{
		"registry": registryName,
		"url":      sourceURL,
		"pulledAt": time.Now().UTC().Format(time.RFC3339),
	}
	return wrapper
}

// extractGitHubMeta extracts github object from server._meta
func extractGitHubMeta(meta model.ServerMeta) map[string]interface{} {
	if meta == nil {
		return nil
	}
	publisherProvided, ok := meta["io.modelcontextprotocol.registry/publisher-provided"].(map[string]interface{})
	if !ok {
		return nil
	}
	github, ok := publisherProvided["github"].(map[string]interface{})
	if !ok {
		return nil
	}
	return github
}

// extractGitHubInfo extracts GitHub repository info from github metadata
// Fields come directly from VSCode/GitHub Copilot registry format:
// nameWithOwner -> "owner/repo", name -> repo name, stargazerCount -> stars
func extractGitHubInfo(github map[string]interface{}) *model.GithubInfo {
	if github == nil {
		return nil
	}

	info := &model.GithubInfo{}

	// Extract owner and repo from nameWithOwner (e.g., "upstash/context7")
	if nameWithOwner, ok := github["nameWithOwner"].(string); ok && nameWithOwner != "" {
		parts := strings.SplitN(nameWithOwner, "/", 2)
		if len(parts) == 2 {
			info.Owner = parts[0]
			info.Repo = parts[1]
		}
	}

	// Name field
	if name, ok := github["name"].(string); ok {
		info.Name = name
	}

	// Build URL from owner/repo if we have them
	if info.Owner != "" && info.Repo != "" {
		info.URL = "https://github.com/" + info.Owner + "/" + info.Repo
	}

	// Stars (stargazerCount in VSCode format)
	if stars, ok := github["stargazerCount"].(float64); ok {
		info.Stars = int(stars)
	}

	// Path is not typically in VSCode format, but check anyway
	if path, ok := github["path"].(string); ok {
		info.Path = path
	}

	// Only return if we have meaningful data
	if info.Owner == "" && info.Repo == "" && info.Name == "" {
		return nil
	}

	return info
}

// extractGitHubInfoFromRepository extracts GitHub info from server.Repository
// Parses GitHub URLs like "https://github.com/owner/repo" to extract owner and repo
func extractGitHubInfoFromRepository(repo *model.Repository) *model.GithubInfo {
	if repo == nil || repo.URL == "" {
		return nil
	}

	// Only process GitHub URLs
	url := strings.ToLower(repo.URL)
	if !strings.Contains(url, "github.com") {
		return nil
	}

	// Parse URL: https://github.com/owner/repo or https://github.com/owner/repo.git
	// Remove trailing .git if present
	cleanURL := strings.TrimSuffix(repo.URL, ".git")

	// Extract path after github.com
	var path string
	if idx := strings.Index(strings.ToLower(cleanURL), "github.com/"); idx != -1 {
		path = cleanURL[idx+len("github.com/"):]
	} else {
		return nil
	}

	// Split into owner/repo
	parts := strings.SplitN(path, "/", 3) // 3 to handle any trailing path
	if len(parts) < 2 {
		return nil
	}

	owner := parts[0]
	repoName := parts[1]

	if owner == "" || repoName == "" {
		return nil
	}

	info := &model.GithubInfo{
		Owner: owner,
		Repo:  repoName,
		URL:   "https://github.com/" + owner + "/" + repoName,
	}

	// Set Path from subfolder if present
	if repo.Subfolder != "" {
		info.Path = repo.Subfolder
	}

	return info
}

// copyMeta creates a shallow copy of a ServerMeta map
func copyMeta(m model.ServerMeta) model.ServerMeta {
	if m == nil {
		return nil
	}
	newMeta := make(model.ServerMeta)
	for k, v := range m {
		newMeta[k] = v
	}
	return newMeta
}

// copyWrapper creates a copy of wrapper with new maps
func copyWrapper(w model.ServerWrapper) model.ServerWrapper {
	newMeta := make(model.ServerMeta)
	for k, v := range w.Meta {
		newMeta[k] = v
	}
	serverMeta := make(model.ServerMeta)
	for k, v := range w.Server.Meta {
		serverMeta[k] = v
	}
	return model.ServerWrapper{
		Server: model.Server{
			Schema:         w.Server.Schema,
			Name:           w.Server.Name,
			Description:    w.Server.Description,
			Version:        w.Server.Version,
			Title:          w.Server.Title,
			WebsiteURL:     w.Server.WebsiteURL,
			Repository:     w.Server.Repository,
			Icons:          w.Server.Icons,
			Packages:       w.Server.Packages,
			Remotes:        w.Server.Remotes,
			GithubInfo:     w.Server.GithubInfo,
			Readme:         w.Server.Readme,
			Meta:           serverMeta,
			OrigChildMeta:  w.Server.OrigChildMeta,  // preserve reference (immutable)
			OrigParentMeta: w.Server.OrigParentMeta, // preserve reference (immutable)
		},
		Meta: newMeta,
	}
}

// TransformAll transforms all wrappers
func TransformAll(sourceURL, registryName string, wrappers []model.ServerWrapper) []model.ServerWrapper {
	result := make([]model.ServerWrapper, len(wrappers))
	for i, w := range wrappers {
		result[i] = TransformWrapper(sourceURL, registryName, w)
	}
	return result
}
