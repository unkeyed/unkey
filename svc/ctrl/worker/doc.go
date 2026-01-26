// Package worker implements the Restate workflow worker for Unkey's control plane.
//
// The worker is the execution engine for long-running operations in Unkey's infrastructure,
// handling container builds, deployments, certificate management, and routing configuration
// through the Restate distributed workflow engine. It provides durable execution guarantees
// for operations that span multiple services and may take minutes to complete.
//
// # Architecture
//
// The worker acts as a Restate service host, binding multiple workflow services that handle
// distinct operational concerns. Each service is implemented as a separate sub-package:
//
//   - [deploy] handles container deployments across multiple regions
//   - [certificate] manages TLS certificates via ACME (Let's Encrypt)
//   - [routing] configures traffic routing for custom domains
//   - [versioning] manages application version lifecycle
//
// The worker maintains connections to several infrastructure components: the primary database
// for persistent state, two separate vault services (one for application secrets, one for
// ACME certificates), S3-compatible storage for build artifacts, and ClickHouse for analytics.
//
// # Configuration
//
// Configuration is provided through the [Config] struct, which validates all settings on startup.
// The worker supports two build backends ([BuildBackendDepot] for cloud builds and
// [BuildBackendDocker] for local builds), each with different requirements validated by
// [Config.Validate].
//
// # Usage
//
// The worker is started with [Run], which blocks until the context is cancelled or a fatal
// error occurs:
//
//	cfg := worker.Config{
//	    InstanceID:      "worker-1",
//	    HttpPort:        7092,
//	    DatabasePrimary: "mysql://...",
//	    BuildBackend:    worker.BuildBackendDepot,
//	    // ... additional configuration
//	}
//
//	if err := worker.Run(ctx, cfg); err != nil {
//	    log.Fatal(err)
//	}
//
// # Startup Sequence
//
// [Run] performs initialization in a specific order: configuration validation, vault services
// creation, database connection, build storage initialization, ACME provider setup, Restate
// server binding, admin registration with retry, wildcard certificate bootstrapping, health
// endpoint startup, and optional Prometheus metrics exposure.
//
// # Graceful Shutdown
//
// When the context passed to [Run] is cancelled, the worker performs graceful shutdown by
// stopping the health server, closing database connections, and allowing in-flight Restate
// workflows to complete. The shutdown sequence is managed through a shutdown handler that
// reverses the startup order.
//
// # ACME Certificate Management
//
// When ACME is enabled in configuration, the worker automatically manages TLS certificates
// using Let's Encrypt. It supports HTTP-01 challenges for regular domains and DNS-01
// challenges (via Route53) for wildcard certificates. On startup with a configured default
// domain, [Run] calls [bootstrapWildcardDomain] to ensure the platform's wildcard certificate
// can be automatically renewed.
package worker
