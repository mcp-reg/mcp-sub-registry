package service

import (
	"strings"
	"testing"

	"github.com/mcp-reg/mcp-sub-registry/internal/model"
)

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

	result := TransformWrapper("https://api.mcp.github.com/registry", "vscode", wrapper)

	// Should fill icons from preferredImage
	if len(result.Server.Icons) != 1 {
		t.Errorf("expected 1 icon, got %d", len(result.Server.Icons))
	}
	if len(result.Server.Icons) > 0 && result.Server.Icons[0].Src != "https://example.com/icon.png" {
		t.Errorf("expected icon src 'https://example.com/icon.png', got '%s'", result.Server.Icons[0].Src)
	}

	// Should fill title from displayName
	if result.Server.Title != "Test Display Name" {
		t.Errorf("expected title 'Test Display Name', got '%s'", result.Server.Title)
	}

	// Should have source tracking
	source, ok := result.Meta["io.mcpregistry/source"].(map[string]interface{})
	if !ok {
		t.Fatal("expected source tracking in _meta")
	}
	if source["registry"] != "vscode" {
		t.Errorf("expected registry 'vscode', got '%v'", source["registry"])
	}
}

func TestTransformWrapper_VSCode_NoOverwrite(t *testing.T) {
	// Test that existing icons/title are not overwritten
	wrapper := model.ServerWrapper{
		Server: model.Server{
			Name:        "test/server",
			Description: "Test server",
			Version:     "1.0.0",
			Title:       "Existing Title",
			Icons:       []model.Icon{{Src: "https://existing.com/icon.png"}},
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

	result := TransformWrapper("https://api.mcp.github.com/registry", "vscode", wrapper)

	// Should NOT overwrite existing icons
	if result.Server.Icons[0].Src != "https://existing.com/icon.png" {
		t.Errorf("expected existing icon to be preserved, got '%s'", result.Server.Icons[0].Src)
	}

	// Should NOT overwrite existing title
	if result.Server.Title != "Existing Title" {
		t.Errorf("expected existing title to be preserved, got '%s'", result.Server.Title)
	}
}

func TestTransformWrapper_Obot(t *testing.T) {
	wrapper := model.ServerWrapper{
		Server: model.Server{
			Name:        "test/server",
			Description: "Test server",
			Version:     "1.0.0",
		},
		Meta: model.ServerMeta{
			"ai.obot/server": map[string]interface{}{
				"custom": "data",
			},
		},
	}

	result := TransformWrapper("https://acornlabs.com/registry", "obot", wrapper)

	// Should preserve obot metadata
	obotMeta, ok := result.Meta["ai.obot/server"].(map[string]interface{})
	if !ok {
		t.Fatal("expected ai.obot/server in _meta")
	}
	if obotMeta["custom"] != "data" {
		t.Errorf("expected custom data to be preserved")
	}

	// Should have source tracking
	_, ok = result.Meta["io.mcpregistry/source"].(map[string]interface{})
	if !ok {
		t.Fatal("expected source tracking in _meta")
	}
}

func TestTransformWrapper_Default(t *testing.T) {
	wrapper := model.ServerWrapper{
		Server: model.Server{
			Name:        "test/server",
			Description: "Test server",
			Version:     "1.0.0",
		},
	}

	result := TransformWrapper("https://unknown-registry.com", "unknown", wrapper)

	// Should add official status
	official, ok := result.Meta["io.modelcontextprotocol.registry/official"].(map[string]interface{})
	if !ok {
		t.Fatal("expected official status in _meta")
	}
	if official["status"] != "active" {
		t.Errorf("expected status 'active', got '%v'", official["status"])
	}

	// Should have source tracking
	source, ok := result.Meta["io.mcpregistry/source"].(map[string]interface{})
	if !ok {
		t.Fatal("expected source tracking in _meta")
	}
	if source["registry"] != "unknown" {
		t.Errorf("expected registry 'unknown', got '%v'", source["registry"])
	}
}

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

			source, ok := result.Meta["io.mcpregistry/source"].(map[string]interface{})
			if !ok {
				t.Fatal("expected source tracking in _meta")
			}

			if source["registry"] != tc.registryName {
				t.Errorf("expected registry '%s', got '%v'", tc.registryName, source["registry"])
			}

			if source["url"] != tc.sourceURL {
				t.Errorf("expected url '%s', got '%v'", tc.sourceURL, source["url"])
			}

			pulledAt, ok := source["pulledAt"].(string)
			if !ok || pulledAt == "" {
				t.Error("expected pulledAt timestamp")
			}
		})
	}
}

func TestTransformWrapper_DoesNotModifyOriginal(t *testing.T) {
	original := model.ServerWrapper{
		Server: model.Server{
			Name:        "test/server",
			Description: "Test server",
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

func TestTransformAll(t *testing.T) {
	wrappers := []model.ServerWrapper{
		{Server: model.Server{Name: "test/server1", Version: "1.0.0"}},
		{Server: model.Server{Name: "test/server2", Version: "2.0.0"}},
	}

	results := TransformAll("https://example.com", "test", wrappers)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for i, result := range results {
		source, ok := result.Meta["io.mcpregistry/source"].(map[string]interface{})
		if !ok {
			t.Errorf("result %d: expected source tracking", i)
		}
		if source["registry"] != "test" {
			t.Errorf("result %d: expected registry 'test'", i)
		}
	}
}

func TestTransformWrapper_URLPatternMatching(t *testing.T) {
	testCases := []struct {
		name          string
		sourceURL     string
		registryName  string
		expectedMatch string // "vscode", "obot", or "default"
	}{
		{"api.mcp.github.com", "https://api.mcp.github.com/v0.1/servers", "vscode", "vscode"},
		{"github.com with mcp", "https://github.com/modelcontextprotocol/mcp-server", "gh", "vscode"},
		{"acornlabs", "https://acornlabs.io/registry", "acorn", "obot"},
		{"obot name", "https://example.com", "obot", "obot"},
		{"unknown", "https://random.com", "random", "default"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrapper := model.ServerWrapper{
				Server: model.Server{Name: "test/server", Version: "1.0.0"},
			}

			result := TransformWrapper(tc.sourceURL, tc.registryName, wrapper)

			// Check which transformer was applied based on the result
			switch tc.expectedMatch {
			case "default":
				// Default transformer adds official status
				if _, exists := result.Meta["io.modelcontextprotocol.registry/official"]; !exists {
					t.Errorf("expected default transformer to add official status")
				}
			}

			// All should have source tracking
			source := result.Meta["io.mcpregistry/source"].(map[string]interface{})
			if !strings.EqualFold(source["registry"].(string), tc.registryName) {
				t.Errorf("expected registry '%s'", tc.registryName)
			}
		})
	}
}
