//go:build embed

package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves the embedded SPA.
// Returns nil if embedding is disabled (default build).
func Handler() http.Handler {
	build, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("frontend: failed to create sub filesystem: " + err.Error())
	}

	httpFS := http.FS(build)
	fileServer := http.FileServer(httpFS)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Set cache headers for hashed assets (immutable)
		if strings.HasPrefix(path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		// Try to serve the file
		f, err := httpFS.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// Enabled returns true if frontend embedding is enabled.
func Enabled() bool {
	return true
}
