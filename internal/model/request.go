package model

// RepoRef identifies a specific repo and branch
type RepoRef struct {
	Org    string
	Repo   string
	Branch string
}

// CacheKey returns the cache key for this repo ref
func (r RepoRef) CacheKey() string {
	return r.Org + "/" + r.Repo + "/" + r.Branch
}

// ListParams contains query parameters for server listing
type ListParams struct {
	Cursor       string
	Limit        int
	Search       string
	UpdatedSince string
	Version      string
}

// DefaultLimit is the default page size
const DefaultLimit = 100

// MaxLimit is the maximum page size
const MaxLimit = 1000
