// Package status returns whatever HTTP status the caller asks for via
// the URL path. Useful for exercising sentinel's handling of upstream
// errors (502, 429, etc.) without needing an upstream that actually
// fails.
package status

import (
	"net/http"
	"strconv"
)

// Handler is registered by main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(r.PathValue("code"))
	if err != nil || code < 100 || code > 599 {
		http.Error(w, "code must be a valid HTTP status (100-599)", http.StatusBadRequest)
		return
	}
	http.Error(w, http.StatusText(code), code)
}
