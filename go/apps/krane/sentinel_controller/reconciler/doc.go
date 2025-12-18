// Package reconciler implements the Kubernetes reconciliation logic for Sentinel resources.
//
// This package contains the core reconciliation logic that ensures the desired state
// specified in Sentinel custom resources is reflected in the actual Kubernetes
// resources (Deployments and Services). The reconciler follows controller-runtime
// patterns and integrates with the broader krane ecosystem.
//
// # Reconciliation Flow
//
// The reconciliation process follows a predictable pattern:
//  1. Fetch the current Sentinel resource from the Kubernetes API
//  2. Ensure required Kubernetes resources exist (Deployment, Service)
//  3. Reconcile individual aspects (image, replicas, resources)
//  4. Update status conditions based on reconciliation results
//  5. Return appropriate requeue timing for next reconciliation
//
// # Resource Management Strategy
//
// The reconciler uses a declarative approach where each reconciliation cycle
// computes the desired state and applies it, rather than tracking specific
// changes. This approach is more robust against missed events and ensures
// eventual consistency even when the controller is restarted.
//
// # Key Components
//
//   - [Reconciler]: Main reconciliation struct implementing controller-runtime.Reconciler
//   - [ensureDeploymentExists]: Creates and updates sentinel Deployments
//   - [ensureServiceExists]: Creates Services for sentinel network exposure
//   - [reconcileImage]: Updates container images when they change
//   - [reconcileReplicas]: Adjusts pod replica counts
//
// # Error Handling and Recovery
//
// The reconciler distinguishes between different types of errors:
//   - Not found errors are handled gracefully (resource deletion)
//   - API errors trigger immediate requeue for retry
//   - Successful operations may trigger periodic requeue for consistency
//
// # Integration Points
//
// The reconciler integrates with:
//   - Kubernetes API server for resource CRUD operations
//   - controller-runtime manager for event handling and lifecycle
//   - krane k8s package for standardized label management
//   - OpenTelemetry for structured logging and observability
package reconciler
