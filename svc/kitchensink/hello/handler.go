// Package hello is the simplest kitchensink probe: it returns a constant
// 200 OK response. Use this as a smoke test for the pipeline itself, and
// as the reference shape for every new probe you add to kitchensink.
package hello

import "net/http"

// Handler writes "hello, world" as text/plain. Registered by main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("hello, world\n"))
}
