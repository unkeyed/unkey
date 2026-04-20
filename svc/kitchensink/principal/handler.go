// Package principal decodes the X-Unkey-Principal header that sentinel
// sets after a successful auth policy and echoes the decoded principal
// as JSON.
//
// Reaching this endpoint directly (not through sentinel) returns 401.
// A malformed header returns 502 — sentinel sent us garbage, not the
// caller.
package principal

import (
	"encoding/json"
	"net/http"

	"github.com/unkeyed/unkey/svc/kitchensink/internal/httpx"
)

// header is the header sentinel sets after a successful auth policy.
// Mirrors svc/sentinel/engine.PrincipalHeader — duplicated here to keep
// kitchensink stdlib-only and free of cross-service imports.
const header = "X-Unkey-Principal"

// Handler is registered by main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	raw := r.Header.Get(header)
	if raw == "" {
		http.Error(w, header+" header not set; reach this endpoint through sentinel with an auth policy", http.StatusUnauthorized)
		return
	}
	var p map[string]any
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		http.Error(w, "invalid principal JSON from sentinel: "+err.Error(), http.StatusBadGateway)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}
