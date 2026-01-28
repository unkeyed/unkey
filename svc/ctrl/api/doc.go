// Package api provides the control plane HTTP/2 server for Unkey's distributed infrastructure.
//
// The control plane coordinates deployment workflows, certificate management, and cluster
// operations across the Unkey platform. It exposes Connect RPC services over HTTP/2 and
// integrates with Restate for durable workflow execution.
//
// # Architecture
//
// The control plane sits at the center of Unkey's infrastructure, coordinating between:
//   - Sentinel instances that run customer workloads
//   - Restate for durable async workflow execution
//   - S3-compatible storage for build artifacts
//   - ACME providers for automatic TLS certificates
//
// # Services
//
// The server exposes several Connect RPC services:
//
//   - [ctrl.Ctrl] - Core control plane operations
//   - [deployment.Deployment] - Application deployment workflows
//   - [acme.Acme] - ACME certificate management and HTTP-01 challenges
//   - [openapi.OpenApi] - OpenAPI specification management
//   - [cluster.Cluster] - Cluster coordination and sentinel management
//
// # Usage
//
// Configure and start the control plane:
//
//	cfg := api.Config{
//	    InstanceID:      "ctrl-1",
//	    HttpPort:        8080,
//	    DatabasePrimary: "postgres://...",
//	    Clock:           clock.RealClock{},
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
