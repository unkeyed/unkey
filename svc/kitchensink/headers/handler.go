// Package headers echoes the incoming request headers as JSON. Useful
// for debugging header propagation through sentinel, load balancers, or
// any proxy in the path.
package headers

import (
	"net/http"

	"github.com/unkeyed/unkey/svc/kitchensink/internal/httpx"
)

// Handler is registered by main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, r.Header)
}
