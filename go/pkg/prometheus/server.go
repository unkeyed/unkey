/*
Package prometheus provides utilities for exposing Prometheus metrics over HTTP.

This package makes it easy to integrate Prometheus metrics collection and exposure
into applications built with the zen framework. It handles the setup of a metrics
endpoint that Prometheus can scrape to collect runtime metrics from your application.

Common use cases include:
  - Creating a dedicated metrics server separate from your application server
  - Adding metrics endpoints to existing zen-based applications
  - Setting up consistent metrics across multiple microservices

This package is designed to work seamlessly with the zen framework while providing
all the functionality of the standard Prometheus client library.
*/
package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Config configures the Prometheus metrics server.
// It specifies dependencies needed for the server to function correctly.
type Config struct {
	// Logger is used for recording operational events from the metrics server.
	// This logger should be configured appropriately for your environment.
	Logger logging.Logger
}

// New creates a zen server that exposes Prometheus metrics at the /metrics endpoint.
// The server is configured to handle GET requests to the /metrics path using the
// standard Prometheus HTTP handler, which serves metrics in a format that Prometheus
// can scrape.
//
// New is used to create a standalone metrics server that can be started separately
// from your main application server, which is a common pattern for microservices
// architectures where concerns are separated.
//
// Parameters:
//   - config: Configuration for the server, including required dependencies.
//
// Returns:
//   - A configured zen.Server ready to be started.
//   - An error if server creation fails, typically due to invalid configuration.
//
// Example usage:
//
//	// Create a dedicated metrics server
//	logger := logging.NewLogger()
//	server, err := prometheus.New(prometheus.Config{
//	    Logger: logger,
//	})
//	if err != nil {
//	    log.Fatalf("Failed to create metrics server: %v", err)
//	}
//
//	// Start the metrics server on port 9090
//	go func() {
//	    if err := server.Listen(":9090"); err != nil {
//	        log.Fatalf("Metrics server failed: %v", err)
//	    }
//	}()
//
// When used with CLI commands, the server can be started with a command like:
//
//	myapp metrics --port=9090
//
// See [zen.New] for details on the underlying server creation.
// See [promhttp.Handler] for details on the Prometheus metrics handler.
func New(config Config) (*zen.Server, error) {
	z, err := zen.New(zen.Config{
		Logger: config.Logger,
		Flags:  nil,
	})

	if err != nil {
		return nil, err
	}

	h := promhttp.Handler()
	// Register the metrics endpoint with the zen server
	z.RegisterRoute([]zen.Middleware{}, zen.NewRoute("GET", "/metrics", func(ctx context.Context, s *zen.Session) error {
		h.ServeHTTP(s.ResponseWriter(), s.Request())
		return nil
	}))

	return z, nil
}
