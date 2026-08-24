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
//	    if err := prometheus.Serve(":9090"); err != nil {
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
// The server uses net/http directly so metrics-only services do not link an
// application HTTP framework and its validation dependencies.
package prometheus
