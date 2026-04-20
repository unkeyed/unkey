// Package httpx provides the small response helpers shared across
// kitchensink probes. It exists purely to keep each probe's handler
// focused on what it's actually demonstrating — one helper call beats
// four lines of header plumbing.
//
// Kept under internal/ so the helpers stay out of kitchensink's
// "worked examples" surface: engineers reading a probe see stdlib
// calls plus a small local import, not a sprawling utility layer.
package httpx

import (
	"encoding/json"
	"net/http"
)

// JSON writes v as indented JSON with the given status code and the
// application/json Content-Type.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
