package zen

import (
	"net/http"
	"strings"
)

// corsHandler wraps an http.Handler with CORS support. It runs before the
// mux so that OPTIONS preflight requests are handled even when only POST
// (or another method) is registered for a path.
type corsHandler struct {
	next      http.Handler
	originSet map[string]struct{}
}

func newCORSHandler(next http.Handler, allowedOrigins []string) *corsHandler {
	set := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		set[strings.TrimRight(o, "/")] = struct{}{}
	}
	return &corsHandler{next: next, originSet: set}
}

func (c *corsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	// No Origin header — not a CORS request.
	if origin == "" {
		c.next.ServeHTTP(w, r)
		return
	}

	// Check if origin is allowed. Normalize trailing slash to match config.
	normalizedOrigin := strings.TrimRight(origin, "/")
	if _, ok := c.originSet[normalizedOrigin]; !ok {
		c.next.ServeHTTP(w, r)
		return
	}

	h := w.Header()
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Access-Control-Allow-Credentials", "true")
	h.Set("Vary", "Origin")

	// Handle preflight.
	if r.Method == http.MethodOptions {
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		h.Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	c.next.ServeHTTP(w, r)
}
