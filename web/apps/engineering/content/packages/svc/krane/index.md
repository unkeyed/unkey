---
title: krane
description: "provides a distributed container orchestration agent for managing"
---

Package krane provides a distributed container orchestration agent for managing deployments and sentinels across Kubernetes clusters via gRPC APIs.

Krane acts as a node-level agent that exposes gRPC endpoints for container orchestration operations. It manages two primary resource types: deployments (application workloads implemented as ReplicaSets) and sentinels (monitoring/gateway components implemented as Deployments). The agent handles authentication, secrets decryption, and resource lifecycle management through streaming APIs.

### Architecture

Krane uses a split control loop architecture with two independent controllers:

  - \[deployment.Controller]: Manages user workload ReplicaSets via the SyncDeployments stream. Has its own version cursor and circuit breaker.

  - \[sentinel.Controller]: Manages sentinel Deployments and Services via the SyncSentinels stream. Has its own version cursor and circuit breaker.

This separation provides failure isolation: if one controller experiences errors, the other continues operating independently. Each controller maintains its own connection to the control plane with separate version cursors for resumable streaming.

The system consists of these main components:

  - Control Plane: External service that streams state to krane agents
  - Krane Agents: Node-level agents with independent deployment/sentinel controllers
  - Kubernetes Cluster: Target infrastructure where containers are deployed

Each krane instance is identified by a unique InstanceID. The agent uses in-cluster Kubernetes configuration for direct cluster access.

### Key Services

\[kubernetes.Service]: Implements the SchedulerService gRPC interface for deployment and sentinel management operations. Handles ReplicaSet and Deployment creation, updates, deletion, and real-time status streaming.

\[secrets.Service]: Implements the SecretsService gRPC interface for decrypting environment variables and secrets. Integrates with vault service for secure secrets management and validates requests using Kubernetes service account tokens.

### Resource Types

Deployment: Application workloads implemented as Kubernetes ReplicaSets with specified container images, resource limits, and replica counts. Each deployment receives standardized environment variables and optional encrypted secrets.

Sentinel: Edge components implemented as Kubernetes Deployments for monitoring, routing, and security functions. Sentinels are managed separately from deployments and provide infrastructure services.

### Usage

Basic krane agent setup:

	cfg := krane.Config{
		InstanceID:    "krane-node-001",
		Region:        "us-west-2",
		RegistryURL:   "registry.depot.dev",
		RegistryUsername: "x-token",
		RegistryPassword: "depot-token",
		RPCPort:       8080,
		PrometheusPort: 9090,
		VaultMasterKeys: []string{"master-key-1"},
		VaultS3: krane.S3Config{
			URL:             "https://s3.amazonaws.com",
			Bucket:          "krane-vault",
			AccessKeyID:     "access-key",
			AccessKeySecret: "secret-key",
		},
	}
	err := krane.Run(context.Background(), cfg)

The agent will:

 1. Initialize Kubernetes client using in-cluster configuration
 2. Start gRPC server on configured RPCPort with SchedulerService handler
 3. Register SecretsService handler if vault configuration is provided
 4. Start Prometheus metrics server on configured PrometheusPort
 5. Handle graceful shutdown on context cancellation

### Authentication and Security

Krane uses Kubernetes service account tokens for authentication. The SecretsService validates that requests originate from pods belonging to the expected deployment by checking pod annotations and service account membership. All secrets are encrypted at rest using the vault service with configurable master keys.

### Label Management

All managed resources are labeled with standardized metadata for identification and selection. The labels package provides utilities for label management following Kubernetes conventions with the "unkey.com/" prefix for organization-specific labels and "app.kubernetes.io/" for standard Kubernetes labels.

### Observability

Krane exposes Prometheus metrics for monitoring gRPC API health, resource operations, and system performance. Structured logging provides detailed visibility into operations with correlation through InstanceID and Region fields.

## Functions

### func Run

```go
func Run(ctx context.Context, cfg Config) error
```

Run starts the krane agent server with the provided configuration.

This function initializes all required services including Kubernetes client, vault service for secrets management, gRPC servers for API endpoints, and Prometheus metrics server. It blocks until the context is cancelled or a fatal error occurs.

The function performs these steps in order: 1. Validates the configuration 2. Creates structured logger with instance metadata 3. Initializes vault service if master keys and S3 config are provided 4. Creates Kubernetes client using in-cluster configuration 5. Sets up gRPC server with SchedulerService handler 6. Registers SecretsService handler if vault is configured 7. Starts Prometheus metrics server if port is configured 8. Blocks until context cancellation or signal 9. Performs graceful shutdown of all services

Returns an error if configuration validation fails, service initialization fails, or during shutdown. Context cancellation results in clean shutdown with nil error.


## Types

### type Config

```go
type Config struct {
	// InstanceID is the unique identifier for this krane agent instance.
	// Used for distributed tracing, logging correlation, and cluster coordination.
	// Must be unique across all running krane instances in the same cluster.
	InstanceID string

	// Region identifies the geographic region where this node is deployed.
	// Used for observability, latency optimization, and compliance requirements.
	// Must match the region identifier used by the underlying cloud platform
	// and control plane configuration.
	Region string

	// RegistryURL is the URL of the container registry for pulling images.
	// Should include the protocol and registry domain, e.g., "registry.depot.dev"
	// or "https://registry.example.com". Used by all deployments unless overridden.
	RegistryURL string

	// RegistryUsername is the username for authenticating with the container registry.
	// Common values include "x-token" for token-based authentication or the
	// actual registry username. Must be paired with RegistryPassword.
	RegistryUsername string

	// RegistryPassword is the password or token for authenticating with the container registry.
	// Should be stored securely (e.g., environment variable or secret management system).
	// For token-based auth, this is the actual token value.
	RegistryPassword string

	// Clock provides time operations for testing and time zone handling.
	// Use clock.RealClock{} for production deployments and mock clocks for
	// deterministic testing. Enables time-based operations to be controlled in tests.
	Clock clock.Clock

	// PrometheusPort specifies the port for exposing Prometheus metrics.
	// Set to 0 to disable metrics exposure. When enabled, metrics are served
	// on all interfaces (0.0.0.0) on the specified port.
	PrometheusPort int

	// VaultURL is the URL of the remote vault service (e.g., http://vault:8080).
	// Required for decrypting environment variable secrets.
	VaultURL string

	// VaultToken is the authentication token for the vault service.
	// Used to authenticate requests to the vault API.
	VaultToken string

	// RPCPort specifies the port for the gRPC server that exposes krane APIs.
	// The SchedulerService and optionally SecretsService are served on this port.
	// Must be a valid port number (1-65535).
	RPCPort int

	ControlPlaneURL    string
	ControlPlaneBearer string

	// OtelEnabled enables OpenTelemetry instrumentation for tracing and metrics.
	// When true, InitGrafana will be called to set up OTEL exporters.
	OtelEnabled bool

	// OtelTraceSamplingRate controls the sampling rate for traces (0.0 to 1.0).
	// Only used when OtelEnabled is true.
	OtelTraceSamplingRate float64

	// LogSampleRate is the baseline probability (0.0-1.0) of emitting log events.
	LogSampleRate float64

	// LogSlowThreshold defines what duration qualifies as "slow" for sampling.
	LogSlowThreshold time.Duration
}
```

Config holds configuration for the krane agent server.

This configuration defines how the krane agent connects to Kubernetes, authenticates with container registries, handles secrets, and exposes metrics.

#### func (Config) Validate

```go
func (c Config) Validate() error
```

Validate checks the configuration for required fields and logical consistency.

Returns an error if required fields are missing or configuration values are invalid. This method should be called before starting the krane agent to ensure proper configuration and provide early feedback on configuration errors.

Currently, this method always returns nil as validation is not implemented. Future implementations will validate required fields such as RPCPort, RegistryURL, and consistency between VaultMasterKeys and VaultS3 configuration.

