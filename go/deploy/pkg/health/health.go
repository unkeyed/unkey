// Package health provides a unified health check implementation for all services
package health

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response represents a standardized health check response
type Response struct {
	Status  string  `json:"status"`
	Service string  `json:"service"`
	Version string  `json:"version"`
	Uptime  float64 `json:"uptime_seconds"`
}

// Handler creates a standard health check handler
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

// SimpleHandler creates a basic health check that just returns OK
// This is useful for load balancers that don't need JSON
func SimpleHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}