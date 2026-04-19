// Package env exposes process environment variables over HTTP. Useful
// for verifying deployment env injection works end-to-end.
package env

import (
	"net/http"
	"os"
	"strings"

	"github.com/unkeyed/unkey/svc/kitchensink/internal/httpx"
)

// Handler returns process env as JSON. ?prefix= filters by key prefix.
// Registered by main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	out := map[string]string{}
	for _, kv := range os.Environ() {
		k, v, _ := strings.Cut(kv, "=")
		if prefix != "" && !strings.HasPrefix(k, prefix) {
			continue
		}
		out[k] = v
	}
	httpx.JSON(w, http.StatusOK, out)
}
