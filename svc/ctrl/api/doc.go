// Package api provides the control plane HTTP/2 server for Unkey's distributed infrastructure.
//
// The control plane coordinates deployment workflows, certificate management, and cluster
// operations across the Unkey platform. It exposes Connect RPC services over HTTP/2 and
// integrates with Restate for durable workflow execution.
//
// # Architecture
//
// The control plane sits at the center of Unkey's infrastructure. It coordinates
// sentinel instances that run customer workloads, Restate for durable workflow
// execution, build artifact storage, and ACME providers for automatic TLS
// certificates. Connect RPC services are exposed for core control plane
// operations, deployment workflows, ACME management, OpenAPI specs, and cluster
// coordination.
//
// # Usage
//
// Configure and start the control plane:
//
//	cfg := api.Config{
//	    InstanceID:      "ctrl-1",
//	    HttpPort:        8080,
//	    DatabasePrimary: "postgres://...",
//	    Restate: api.RestateConfig{
//	        URL:      "http://restate:8080",
//	        AdminURL: "http://restate:9070",
//	    },
//	}
//	if err := api.Run(ctx, cfg); err != nil {
//	    log.Fatal(err)
//	}
//
// The server supports both HTTP/2 cleartext (h2c) for development and TLS for production.
// When [Config.TLSConfig] is set, the server uses HTTPS; otherwise it uses h2c to allow
// HTTP/2 without TLS.
//
// # Shutdown
//
// The [Run] function handles graceful shutdown when the provided context is cancelled.
// All active connections are drained, database connections closed, and telemetry flushed
// before the function returns.
package api
