// Package prometheus provides HTTP server infrastructure for exposing Prometheus metrics.
//
// This package is the entry point for running a metrics server that exposes the
// /metrics endpoint for Prometheus scraping. The actual metric collectors are defined
// in the [metrics] subpackage.
//
// # Usage
//
// Metrics register in the default registry at package init (promauto). Each
// service binary links only its own metrics packages. Serve them on a
// dedicated listener. Serve blocks, so run it in a goroutine and shut it down
// on context cancellation:
//
//	prom, err := prometheus.New()
//	if err != nil {
//	    return err
//	}
//	ln, err := net.Listen("tcp", ":9090")
//	if err != nil {
//	    return err
//	}
//	go func() {
//	    if err := prom.Serve(ctx, ln); err != nil && !errors.Is(err, context.Canceled) {
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
