// Package echo returns the request body verbatim and preserves
// Content-Type. Useful for verifying body propagation and that proxies
// aren't silently rewriting payloads.
package echo

import (
	"io"
	"net/http"
)

// Handler is registered by main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	_, _ = io.Copy(w, r.Body)
}
