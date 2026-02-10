package model

// RegistryFile represents the root registry.json structure
type RegistryFile struct {
	Registries []RegistryEntry `json:"registries"`
}

// RegistryEntry represents a single registry in the registries array
type RegistryEntry struct {
	Name                string      `json:"name"`
	Type                string      `json:"type,omitempty"`
	URL                 string      `json:"url,omitempty"`
	Servers             interface{} `json:"servers,omitempty"` // "*" or map[string]string
	ServersRelativePath []string    `json:"servers_relative_path,omitempty"`
}

// IsPrivate returns true if this is a private registry entry
func (r RegistryEntry) IsPrivate() bool {
	return r.Type == "private"
}

// IsPublic returns true if this is a public registry entry (has URL and not private)
func (r RegistryEntry) IsPublic() bool {
	return r.Type != "private" && r.URL != ""
}

// ShouldFetchAllServers returns true if servers field is "*"
func (r RegistryEntry) ShouldFetchAllServers() bool {
	if str, ok := r.Servers.(string); ok {
		return str == "*"
	}
	return false
}

// GetSpecificServers returns map of server name -> version if servers is a map
func (r RegistryEntry) GetSpecificServers() map[string]string {
	if m, ok := r.Servers.(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range m {
			if vStr, ok := v.(string); ok {
				result[k] = vStr
			}
		}
		return result
	}
	return nil
}
