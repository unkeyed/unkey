// Package logs reads the request body and writes it to stdout at INFO.
// Useful for verifying the platform log aggregator picks up application
// logs.
//
// The package is named "logs" rather than "log" to avoid shadowing the
// stdlib log package in callers.
package logs

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/unkeyed/unkey/svc/kitchensink/internal/httpx"
)

// Handler is registered by main.go.
func Handler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	msg := string(body)
	slog.Info("kitchensink/log", "body", msg)
	httpx.JSON(w, http.StatusOK, map[string]string{"logged": msg})
}
