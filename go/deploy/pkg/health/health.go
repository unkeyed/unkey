// Package health provides HTTP health check handlers.
package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response represents a health check response.
type Response struct {
	// Status is the health status, typically "ok".
	Status string `json:"status"`

	// Service is the service name.
	Service string `json:"service"`

	// Version is the service version.
	Version string `json:"version"`

	// Uptime is the service uptime in seconds.
	Uptime float64 `json:"uptime_seconds"`
}

// Handler returns an HTTP handler that responds with JSON health status.
// The handler calculates uptime from startTime and always returns 200 OK.
// If JSON encoding fails, it returns "OK" as plain text.
func Handler(serviceName, version string, startTime time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			Status:  "ok",
			Service: serviceName,
			Version: version,
			Uptime:  time.Since(startTime).Seconds(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			// If we can't encode the response, return a simple text response
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("OK"))
		}
	}
}

// SimpleHandler returns an HTTP handler that responds with "OK" as plain text.
// The handler always returns 200 OK with no JSON overhead.
func SimpleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
