// Package ctrl provides the main control plane service for the unkey platform.
//
// This package implements the central control plane that orchestrates deployments,
// manages TLS certificates through ACME, handles build operations, and provides
// API services for the unkey ecosystem. It integrates with multiple backend
// services including Restate for workflow orchestration, vault for secrets management,
// and container registries for build operations.
//
// # Architecture
//
// The control plane consists of several integrated components:
//   - HTTP/2 Connect server for API endpoints
//   - Restate workflow engine for asynchronous operations
//   - Vault services for secrets and certificate encryption
//   - Database layer for persistent storage
//   - ACME providers for automatic TLS certificate management
//   - Build backends (Depot or Docker) for container image building
//
// # Key Services
//
// [Deployment Service]: Manages application deployment workflows through Restate
// [Certificate Service]: Handles ACME challenges and TLS certificate lifecycle
// [Build Service]: Orchestrates container image builds via Depot or Docker
// [Routing Service]: Manages domain assignment and traffic routing
// [Cluster Object Service]: Handles cluster metadata and bootstrapping
// [OpenAPI Service]: Provides OpenAPI specification and schema documentation
// [Ctrl Service]: Control plane health and management operations
//
// # ACME Integration
//
// The system supports ACME challenge providers:
//   - HTTP-01 challenges for regular domains
//   - DNS-01 challenges through AWS Route53 for wildcard certificates
//
// # Configuration
//
// The control plane is highly configurable through [Config] which includes:
//   - Database and vault configuration for persistence
//   - Registry and build backend settings for container operations
//   - ACME provider configuration for certificate management
//   - Restate integration for workflow orchestration
//   - TLS and authentication settings for secure operation
//
// # Usage
//
// Basic control plane setup:
//
//	cfg := ctrl.Config{
//		InstanceID:    "ctrl-prod-001",
//		Platform:       "aws",
//		Region:         "us-west-2",
//		HttpPort:       8080,
//		DatabasePrimary: "user:pass@tcp(db:3306)/unkey",
//		RegistryURL:    "registry.depot.dev",
//		RegistryUsername: "x-token",
//		RegistryPassword: "depot-token",
//		VaultMasterKeys: []string{"master-key-1"},
//		VaultS3: ctrl.S3Config{
//			URL:             "https://s3.amazonaws.com",
//			Bucket:          "unkey-vault",
//			AccessKeyID:     "access-key",
//			AccessKeySecret: "secret-key",
//		},
//		BuildBackend: ctrl.BuildBackendDepot,
//		BuildPlatform: "linux/amd64",
//		Acme: ctrl.AcmeConfig{
//			Enabled: true,
//			EmailDomain: "unkey.com",
//			Route53: ctrl.Route53Config{
//				Enabled:         true,
//				AccessKeyID:     "aws-key",
//				SecretAccessKey: "aws-secret",
//				Region:          "us-east-1",
//			},
//		},
//	}
//	err := ctrl.Run(context.Background(), cfg)
//
// The control plane will:
//  1. Initialize all services (database, vault, build backend, etc.)
//  2. Start Restate workflow engine with all service bindings
//  3. Register with Restate admin API for service discovery
//  4. Start HTTP/2 Connect server on configured port
//  5. Handle graceful shutdown on context cancellation
//
// # Observability
//
// The control plane integrates with OpenTelemetry for metrics, traces, and structured logging.
// It exposes health endpoints and provides comprehensive monitoring of all operations
// including deployment status, certificate renewal, and build progress.
package ctrl
