// Package prometheus provides HTTP server infrastructure for exposing Prometheus metrics.
//
// This package is the entry point for running a metrics server that exposes the
// /metrics endpoint for Prometheus scraping. The actual metric collectors are defined
// in the [metrics] subpackage.
//
// # Usage
//
// Create a per-service registry, register standard Go and process collectors,
// then create a server with [NewWithRegistry]:
//
//	reg := promclient.NewRegistry()
//	reg.MustRegister(collectors.NewGoCollector())
//	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
//	prom, err := prometheus.NewWithRegistry(reg)
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
