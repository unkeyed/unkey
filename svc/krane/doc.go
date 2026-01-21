// Package krane provides a distributed container orchestration agent for managing
// deployments and sentinels across Kubernetes clusters via gRPC APIs.
//
// Krane acts as a node-level agent that exposes gRPC endpoints for container
// orchestration operations. It manages two primary resource types: deployments
// (application workloads implemented as ReplicaSets) and sentinels (monitoring/gateway
// components implemented as Deployments). The agent handles authentication,
// secrets decryption, and resource lifecycle management through streaming APIs.
//
// # Architecture
//
// Krane uses a split control loop architecture with two independent controllers:
//
//   - [deployment.Controller]: Manages user workload ReplicaSets via the
//     SyncDeployments stream. Has its own version cursor and circuit breaker.
//
//   - [sentinel.Controller]: Manages sentinel Deployments and Services via the
//     SyncSentinels stream. Has its own version cursor and circuit breaker.
//
// This separation provides failure isolation: if one controller experiences errors,
// the other continues operating independently. Each controller maintains its own
// connection to the control plane with separate version cursors for resumable
// streaming.
//
// The system consists of these main components:
//   - Control Plane: External service that streams state to krane agents
//   - Krane Agents: Node-level agents with independent deployment/sentinel controllers
//   - Kubernetes Cluster: Target infrastructure where containers are deployed
//
// Each krane instance is identified by a unique InstanceID. The agent uses in-cluster
// Kubernetes configuration for direct cluster access.
//
// # Key Services
//
// [kubernetes.Service]: Implements the SchedulerService gRPC interface for
// deployment and sentinel management operations. Handles ReplicaSet and Deployment
// creation, updates, deletion, and real-time status streaming.
//
// [secrets.Service]: Implements the SecretsService gRPC interface for decrypting
// environment variables and secrets. Integrates with vault service for secure
// secrets management and validates requests using Kubernetes service account tokens.
//
// # Resource Types
//
// Deployment: Application workloads implemented as Kubernetes ReplicaSets with
// specified container images, resource limits, and replica counts. Each deployment
// receives standardized environment variables and optional encrypted secrets.
//
// Sentinel: Edge components implemented as Kubernetes Deployments for monitoring,
// routing, and security functions. Sentinels are managed separately from deployments
// and provide infrastructure services.
//
// # Usage
//
// Basic krane agent setup:
//
//	cfg := krane.Config{
//		InstanceID:    "krane-node-001",
//		Region:        "us-west-2",
//		RegistryURL:   "registry.depot.dev",
//		RegistryUsername: "x-token",
//		RegistryPassword: "depot-token",
//		RPCPort:       8080,
//		PrometheusPort: 9090,
//		VaultMasterKeys: []string{"master-key-1"},
//		VaultS3: krane.S3Config{
//			URL:             "https://s3.amazonaws.com",
//			Bucket:          "krane-vault",
//			AccessKeyID:     "access-key",
//			AccessKeySecret: "secret-key",
//		},
//	}
//	err := krane.Run(context.Background(), cfg)
//
// The agent will:
//  1. Initialize Kubernetes client using in-cluster configuration
//  2. Start gRPC server on configured RPCPort with SchedulerService handler
//  3. Register SecretsService handler if vault configuration is provided
//  4. Start Prometheus metrics server on configured PrometheusPort
//  5. Handle graceful shutdown on context cancellation
//
// # Authentication and Security
//
// Krane uses Kubernetes service account tokens for authentication. The SecretsService
// validates that requests originate from pods belonging to the expected deployment
// by checking pod annotations and service account membership. All secrets are
// encrypted at rest using the vault service with configurable master keys.
//
// # Label Management
//
// All managed resources are labeled with standardized metadata for identification
// and selection. The labels package provides utilities for label management following
// Kubernetes conventions with the "unkey.com/" prefix for organization-specific
// labels and "app.kubernetes.io/" for standard Kubernetes labels.
//
// # Observability
//
// Krane exposes Prometheus metrics for monitoring gRPC API health, resource
// operations, and system performance. Structured logging provides detailed visibility
// into operations with correlation through InstanceID and Region fields.
package krane
