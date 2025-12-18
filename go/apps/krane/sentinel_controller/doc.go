// Package sentinelcontroller implements a Kubernetes controller for managing Sentinel
// custom resources as part of the Unkey krane system.
//
// This package provides the infrastructure to deploy and manage Sentinel components,
// which are edge services that provide traffic routing, monitoring, and security
// capabilities for Unkey deployments. Sentinels operate alongside application
// workloads to provide runtime observability and control.
//
// # Architecture Overview
//
// The controller follows Kubernetes controller patterns with a custom event source:
//   - Reflector: Watches sentinel events from the control plane via streaming gRPC
//   - Reconciler: Implements controller-runtime reconcile loop for resource management
//   - Event Buffer: Handles high-throughput events with backpressure management
//   - Resource Management: Creates/updates Deployments and Services based on Sentinel specs
//
// This hybrid architecture was chosen because sentinel state originates from the
// control plane rather than Kubernetes API events, requiring bidirectional sync
// between the database and Kubernetes cluster.
//
// # Sentinel Resources
//
// The controller manages the complete lifecycle of Sentinel resources:
//   - Deployments: Manages sentinel pod replicas and rolling updates
//   - Services: Exposes sentinel endpoints for traffic routing
//   - Status Conditions: Tracks availability, progress, and degradation states
//   - Resource Cleanup: Handles deletion and garbage collection
//
// # Key Components
//
//   - [SentinelController]: Main controller orchestrating all sentinel management
//   - [Reconciler]: Handles Kubernetes-style reconciliation of sentinel resources
//   - [Reflector]: Bridges control plane events to Kubernetes resources
//   - [Sentinel]: Custom resource definition for sentinel specifications
//
// # Integration Points
//
// The controller integrates with several external systems:
//   - Control Plane: Receives sentinel state changes via gRPC streaming
//   - Kubernetes API: Manages Deployments, Services, and CRDs
//   - krane k8s Package: Uses standardized label and annotation management
//   - OpenTelemetry: Provides structured logging and metrics
//
// # Usage
//
// The controller is instantiated by the main krane agent and integrated into the
// controller-runtime manager:
//
//	cfg := sentinelcontroller.Config{
//		Logger:     logger,
//		Scheme:     scheme,
//		Client:     k8sClient,
//		Manager:    manager,
//		Cluster:    clusterClient,
//		InstanceID: "instance-123",
//		Region:     "us-west-2",
//		Shard:      "shard-a",
//	}
//	controller, err := sentinelcontroller.New(cfg)
//
// # Error Handling
//
// The controller distinguishes between expected operational states (resource not found,
// reconciliation conflicts) and system errors (API failures, network issues). Reconciliation
// failures trigger immediate requeue with exponential backoff, while successful operations
// are periodically rechecked to maintain eventual consistency.
//
// # Concurrency and Performance
//
// The controller is designed for high-concurrency environments:
//   - Event buffering prevents system overload during control plane bursts
//   - Parallel reconciliation allows multiple sentinel updates simultaneously
//   - Periodic full sync ensures consistency despite missed events
//   - Resource version tracking prevents unnecessary updates
package sentinelcontroller
