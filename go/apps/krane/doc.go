// Package krane provides a distributed container orchestration system for managing
// deployments and sentinels across Kubernetes clusters.
//
// Krane acts as a node-level agent that synchronizes desired state from a central
// control plane with the actual state in Kubernetes. It manages two primary resource
// types: deployments (application workloads) and sentinels (monitoring/gateway
// components).
//
// # Architecture
//
// The system consists of three main components:
//   - Control Plane: Central authority that maintains desired state
//   - Krane Agents: Node-level agents that execute deployment operations
//   - Kubernetes Cluster: Target infrastructure where containers run
//
// Each krane instance is identified by a unique InstanceID and operates within
// a specific Region and Shard for distributed coordination.
//
// # Key Components
//
// [SyncEngine]: Core component that maintains bidirectional synchronization with
// the control plane via streaming gRPC connections.
//
// [deploymentreflector]: Manages Deployment custom resources using the
// Kubernetes controller-runtime pattern. Handles lifecycle operations for
// application workloads.
//
// [SentinelController]: Manages Sentinel custom resources for monitoring and
// gateway components. Provides observability and traffic routing capabilities.
//
// # Resource Types
//
// Deployment: Represents application deployments with specifications for
// container images, resource requirements, and replica counts. Each deployment
// is scoped to a workspace, project, and environment.
//
// Sentinel: Represents edge components that provide monitoring, routing, and
// security functions. Sentinels are deployed alongside applications to provide
// runtime observability and control.
//
// # Usage
//
// Basic krane agent setup:
//
//	cfg := krane.Config{
//		InstanceID:         "krane-node-001",
//		Region:            "us-west-2",
//		Shard:             "production",
//		ControlPlaneURL:   "https://control-plane.example.com",
//		ControlPlaneBearer: "bearer-token",
//		RegistryURL:       "registry.example.com",
//		RegistryUsername:  "krane",
//		RegistryPassword:  "registry-token",
//		PrometheusPort:    9090,
//	}
//	err := krane.Run(context.Background(), cfg)
//
// The agent will:
//  1. Connect to the control plane and establish streaming synchronization
//  2. Create Kubernetes controllers for deployments and sentinels
//  3. Start reconciling desired state with actual cluster state
//  4. Expose metrics on the configured Prometheus port
//  5. Handle graceful shutdown on context cancellation
//
// # Synchronization Flow
//
// 1. Pull: Krane continuously streams desired state updates from the control plane
// 2. Buffer: Events are buffered to handle temporary network interruptions
// 3. Reconcile: Controllers apply changes to Kubernetes resources
// 4. Push: Status updates are sent back to the control plane
// 5. Watch: Kubernetes resource changes are monitored and reported
//
// # Error Handling and Resilience
//
// The system uses circuit breakers to prevent cascade failures when the control
// plane is unavailable. Local buffering ensures that operations can continue during
// brief network interruptions. All controllers implement exponential backoff for
// retry operations and follow Kubernetes best practices for error handling.
//
// # Label Management
//
// All managed resources are labeled with standardized metadata for identification
// and selection. The k8s package provides utilities for label management following
// Kubernetes conventions with the "unkey.com/" prefix for organization-specific
// labels.
//
// # Observability
//
// Krane integrates with OpenTelemetry for distributed tracing and metrics.
// Prometheus metrics are exposed for monitoring the health and performance of
// the synchronization process. Structured logging provides detailed visibility
// into operations and error conditions.
package krane
