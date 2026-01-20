// Package worker provides the Restate workflow worker service.
//
// This package implements the Restate worker that handles asynchronous workflow
// execution for the unkey platform. It runs as a separate service from the main
// control plane, allowing independent scaling and deployment of workflow handlers.
//
// # Architecture
//
// The worker service consists of:
//   - Restate server for workflow handler execution
//   - Health check endpoint for orchestration
//   - Certificate renewal cron job for ACME management
//
// # Workflow Services
//
// The worker binds and executes the following Restate workflow services:
//   - [DeploymentService]: Orchestrates application deployment workflows
//   - [RoutingService]: Manages domain assignment and traffic routing
//   - [VersioningService]: Handles version management operations
//   - [CertificateService]: Processes ACME challenges and certificate lifecycle
//
// # ACME Integration
//
// The worker supports multiple ACME challenge providers:
//   - HTTP-01 challenges for regular domains
//   - DNS-01 challenges through Cloudflare for wildcard certificates
//   - DNS-01 challenges through AWS Route53 for wildcard certificates
//
// Certificate renewal is managed through a cron job that runs after the worker
// successfully registers with the Restate admin API.
//
// # Configuration
//
// The worker is configured through [Config] which includes:
//   - Database and vault configuration for persistence
//   - Build backend settings for container operations
//   - ACME provider configuration for certificate management
//   - Restate configuration for workflow registration
//
// # Usage
//
// Basic worker setup:
//
//	cfg := worker.Config{
//		InstanceID:      "worker-prod-001",
//		HttpPort:        7092,
//		DatabasePrimary: "user:pass@tcp(db:3306)/unkey",
//		VaultMasterKeys: []string{"master-key-1"},
//		VaultS3: worker.S3Config{
//			URL:             "https://s3.amazonaws.com",
//			Bucket:          "unkey-vault",
//			AccessKeyID:     "access-key",
//			AccessKeySecret: "secret-key",
//		},
//		BuildBackend: worker.BuildBackendDepot,
//		BuildPlatform: "linux/amd64",
//		Restate: worker.RestateConfig{
//			URL:        "http://restate:8080",
//			AdminURL:   "http://restate:9070",
//			HttpPort:   9080,
//			RegisterAs: "http://worker:9080",
//		},
//	}
//	err := worker.Run(context.Background(), cfg)
//
// The worker will:
//  1. Initialize all services (database, vault, build backend, etc.)
//  2. Start Restate server with workflow service bindings
//  3. Register with Restate admin API for service discovery
//  4. Bootstrap wildcard domain and start certificate renewal cron
//  5. Start health check endpoint on configured port
//  6. Handle graceful shutdown on context cancellation
//
// # Observability
//
// The worker integrates with OpenTelemetry for metrics, traces, and structured logging.
// It exposes health endpoints and Prometheus metrics for monitoring workflow execution.
package worker
