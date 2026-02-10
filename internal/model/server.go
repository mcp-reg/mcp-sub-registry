package model

// Server represents a server.json file (MCP server definition)
type Server struct {
	Schema      string      `json:"$schema,omitempty"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Version     string      `json:"version"`
	Title       string      `json:"title,omitempty"`
	WebsiteURL  string      `json:"websiteUrl,omitempty"`
	Repository  *Repository `json:"repository,omitempty"`
	Icons       []Icon      `json:"icons,omitempty"`
	Packages    []Package   `json:"packages,omitempty"`
	Remotes     []Remote    `json:"remotes,omitempty"`
	GithubInfo  *GithubInfo `json:"githubInfo,omitempty"`
	Readme         string      `json:"readme,omitempty"`
	Meta           ServerMeta  `json:"_meta,omitempty"`
	OrigChildMeta  ServerMeta  `json:"-"` // original server._meta from upstream (not serialized)
	OrigParentMeta ServerMeta  `json:"-"` // original wrapper._meta from upstream (not serialized)
}

// GithubInfo contains GitHub repository metadata from upstream registries
type GithubInfo struct {
	Owner        string `json:"owner,omitempty"`
	Repo         string `json:"repo,omitempty"`
	URL          string `json:"url,omitempty"`
	Name         string `json:"name,omitempty"`
	Path         string `json:"path,omitempty"`
	Stars        int    `json:"stars,omitempty"`
	Contributors int    `json:"contributors,omitempty"`
}

type Repository struct {
	URL       string `json:"url"`
	Source    string `json:"source"`
	ID        string `json:"id,omitempty"`
	Subfolder string `json:"subfolder,omitempty"`
}

type Icon struct {
	Src      string   `json:"src"`
	MimeType string   `json:"mimeType,omitempty"`
	Sizes    []string `json:"sizes,omitempty"`
	Theme    string   `json:"theme,omitempty"`
}

type Package struct {
	RegistryType         string          `json:"registryType"`
	Identifier           string          `json:"identifier"`
	Version              string          `json:"version,omitempty"`
	FileSha256           string          `json:"fileSha256,omitempty"`
	RuntimeHint          string          `json:"runtimeHint,omitempty"`
	PackageArguments     []Argument      `json:"packageArguments,omitempty"`
	RuntimeArguments     []Argument      `json:"runtimeArguments,omitempty"`
	EnvironmentVariables []KeyValueInput `json:"environmentVariables,omitempty"`
	Transport            *Transport      `json:"transport,omitempty"`
}

type Remote struct {
	Type      string                 `json:"type"`
	URL       string                 `json:"url,omitempty"`
	Headers   []KeyValueInput        `json:"headers,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type Transport struct {
	Type string `json:"type"` // "stdio", "streamable-http", "sse"
}

type Argument struct {
	Type       string `json:"type"` // "positional" or "named"
	ValueHint  string `json:"valueHint,omitempty"`
	Value      string `json:"value,omitempty"`
	Name       string `json:"name,omitempty"`
	IsRepeated bool   `json:"isRepeated,omitempty"`
}

type KeyValueInput struct {
	Name        string   `json:"name"`
	Format      string   `json:"format,omitempty"`
	Description string   `json:"description,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Default     string   `json:"default,omitempty"`
	Value       string   `json:"value,omitempty"`
	Choices     []string `json:"choices,omitempty"`
	IsRequired  bool     `json:"isRequired,omitempty"`
	IsSecret    bool     `json:"isSecret,omitempty"`
}
