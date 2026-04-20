// Package sleep blocks for a caller-specified duration before returning
// 200. Useful for testing sentinel timeouts and slow-upstream behavior.
package sleep

import (
	"net/http"
	"time"
)

// Handler blocks for ?d=<duration> then returns 200. The duration is
// parsed with time.ParseDuration ("500ms", "2s", "1m30s", ...). Honors
// client cancellation so tests don't leak goroutines. Registered by
// main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	dStr := r.URL.Query().Get("d")
	if dStr == "" {
		http.Error(w, "d query param required, e.g. /sleep?d=500ms", http.StatusBadRequest)
		return
	}
	d, err := time.ParseDuration(dStr)
	if err != nil || d < 0 {
		http.Error(w, "d must be a valid duration (e.g. 500ms, 2s, 1m): "+dStr, http.StatusBadRequest)
		return
	}
	select {
	case <-time.After(d):
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("slept " + d.String() + "\n"))
	case <-r.Context().Done():
	}
}
