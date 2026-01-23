// Package prometheus provides HTTP server infrastructure for exposing Prometheus metrics.
//
// This package is the entry point for running a metrics server that exposes the
// /metrics endpoint for Prometheus scraping. The actual metric collectors are defined
// in the [metrics] subpackage.
//
// # Usage
//
// Start the metrics server on a dedicated port, typically in a goroutine since
// [Serve] blocks until the server stops or encounters an error:
//
//	go func() {
//	    if err := prometheus.Serve(ctx, ":9090"); err != nil {
//	        log.Printf("metrics server error: %v", err)
//	    }
//	}()
//
// # Architecture
//
// The package is split into two parts:
//   - This package: HTTP server that exposes metrics at GET /metrics
//   - [metrics] subpackage: All metric collectors organized by subsystem
//
// This separation allows services to import only the metrics they need without
// pulling in HTTP server dependencies.
package prometheus
