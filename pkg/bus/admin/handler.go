// Package admin exposes a tiny, loopback-only HTTP surface for operating
// the bus during incidents. Endpoints flip the bus's pause flag (which
// stops Publish and gates dispatch without leaving the cluster) and report
// status for shell-driven diagnostics.
//
// Routes mount under /_unkey/internal/bus/* and are intended to be served
// on a 127.0.0.1 listener so the surface stays reachable from `kubectl
// exec` but never crosses the network namespace.
package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/bus"
)

// pathPrefix is the URL prefix every bus admin route mounts under.
const pathPrefix = "/_unkey/internal/bus"

// NewHandler returns an http.Handler with the pause/resume/status routes
// pre-registered. Bind it to a loopback listener with http.Server.Serve.
//
// The handler is plain net/http (no zen, no middleware) because this
// surface should never carry auth, validation, or observability — it is
// the kill switch and must work even when those layers are themselves
// misbehaving.
func NewHandler(b bus.Bus) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST "+pathPrefix+"/pause", func(w http.ResponseWriter, _ *http.Request) {
		b.Pause()
		writeJSON(w, statusBody{Paused: true, Members: len(b.Members())})
	})

	mux.HandleFunc("POST "+pathPrefix+"/resume", func(w http.ResponseWriter, _ *http.Request) {
		b.Resume()
		writeJSON(w, statusBody{Paused: false, Members: len(b.Members())})
	})

	mux.HandleFunc("GET "+pathPrefix+"/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, statusBody{Paused: b.IsPaused(), Members: len(b.Members())})
	})

	return mux
}

// NewServer wraps NewHandler with conservative timeouts. Bind to a 127.0.0.1
// listener so the surface is unreachable from outside the pod's network
// namespace.
func NewServer(b bus.Bus) *http.Server {
	return &http.Server{
		Handler:           NewHandler(b),
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
}

// statusBody is the JSON shape returned by every endpoint. Tiny on purpose.
type statusBody struct {
	Paused  bool `json:"paused"`
	Members int  `json:"members"`
}

func writeJSON(w http.ResponseWriter, body statusBody) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
