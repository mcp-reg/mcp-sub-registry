//go:build !embed

package frontend

import "net/http"

// Handler returns nil when embedding is disabled.
func Handler() http.Handler {
	return nil
}

// Enabled returns false when embedding is disabled.
func Enabled() bool {
	return false
}
