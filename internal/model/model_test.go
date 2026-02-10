package model

import (
	"encoding/json"
	"testing"
)

func TestRegistryFileParsing(t *testing.T) {
	data := `{
		"registries": [
			{"name": "public", "url": "https://example.com"},
			{"name": "private", "type": "private", "servers_relative_path": ["a/server.json"]}
		]
	}`

	var rf RegistryFile
	if err := json.Unmarshal([]byte(data), &rf); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(rf.Registries) != 2 {
		t.Errorf("expected 2 registries, got %d", len(rf.Registries))
	}

	if !rf.Registries[1].IsPrivate() {
		t.Error("expected second registry to be private")
	}

	if len(rf.Registries[1].ServersRelativePath) != 1 {
		t.Errorf("expected 1 server path, got %d", len(rf.Registries[1].ServersRelativePath))
	}
}

func TestServerListResponseMarshal(t *testing.T) {
	resp := ServerListResponse{
		Servers: []ServerWrapper{
			{
				Server: Server{
					Name:        "test/server",
					Description: "A test server",
					Version:     "1.0.0",
				},
				Meta: ServerMeta{"io.modelcontextprotocol.registry/official": map[string]interface{}{"status": "active"}},
			},
		},
		Metadata: ListMetadata{
			NextCursor: nil,
			Count:      1,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Verify it can be unmarshaled back
	var parsed ServerListResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed.Servers[0].Server.Name != "test/server" {
		t.Errorf("expected name 'test/server', got '%s'", parsed.Servers[0].Server.Name)
	}
}

func TestErrorResponseMarshal(t *testing.T) {
	resp := ErrorResponse{
		Error: "Not found",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	expected := `{"error":"Not found"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}
