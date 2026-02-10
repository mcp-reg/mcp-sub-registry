package model

// ServerListResponse is the response for GET /v0.1/servers
type ServerListResponse struct {
	Servers  []ServerWrapper `json:"servers"`
	Metadata ListMetadata    `json:"metadata"`
}

// ServerWrapper wraps a server with metadata
type ServerWrapper struct {
	Server Server     `json:"server"`
	Meta   ServerMeta `json:"_meta,omitempty"`
}

// ServerMeta is a map of namespaced keys to metadata values
type ServerMeta map[string]interface{}

// ListMetadata contains pagination info
type ListMetadata struct {
	NextCursor *string `json:"nextCursor"`
	Count      int     `json:"count"`
}

// VersionResponse is the response for version endpoints
type VersionResponse struct {
	Server Server     `json:"server"`
	Meta   ServerMeta `json:"_meta,omitempty"`
}

// ErrorResponse is the standard error format
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse is the health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// RefreshResponse is the response for refresh endpoint
type RefreshResponse struct {
	Message   string `json:"message"`
	Refreshed bool   `json:"refreshed"`
}
