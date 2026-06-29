// Package krane provides a distributed container orchestration agent for managing
// deployments across Kubernetes clusters via gRPC APIs.
//
// Krane acts as a node-level agent that exposes gRPC endpoints for container
// orchestration operations. It manages deployments (application workloads
// implemented as ReplicaSets) along with their supporting Kubernetes resources.
// The agent handles authentication, secrets decryption, and resource lifecycle
// management through streaming APIs.
//
// # Architecture
//
// Krane consumes a single unified WatchDeploymentChanges stream from the control
// plane. A [watcher.Watcher] dispatches each change to the controllers that
// reconcile the corresponding Kubernetes resources:
//
//   - [deployment.Controller]: Manages user workload ReplicaSets, their
//     HorizontalPodAutoscalers, and per-deployment CiliumNetworkPolicies. Reports
//     observed pod state back to the control plane.
//
// The stream carries a version cursor so it can resume after disconnects, and a
// resync loop periodically reconciles drift as a safety net.
//
// The system consists of these main components:
//   - Control Plane: External service that streams state to krane agents
//   - Krane Agents: Node-level agents running the watcher and deployment controller
//   - Kubernetes Cluster: Target infrastructure where containers are deployed
//
// Each krane instance is identified by a unique InstanceID. The agent uses in-cluster
// Kubernetes configuration for direct cluster access.
//
// # Key Services
//
// [kubernetes.Service]: Implements the SchedulerService gRPC interface for
// deployment management operations. Handles ReplicaSet creation, updates,
// deletion, and real-time status streaming.
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
// # Usage
//
// Basic krane agent setup:
//
//	cfg, err := config.Load[krane.Config]("/etc/unkey/krane.toml")
//	if err != nil { ... }
//	cfg.Clock = clock.New()
//	err = krane.Run(context.Background(), cfg)
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
