// Package reflector bridges the control plane and Kubernetes for Sentinel resources.
//
// This package implements the database-to-Kubernetes synchronization mechanism
// for sentinel resources. Unlike traditional Kubernetes controllers that watch
// API events, this reflector watches sentinel state changes from the control plane
// via streaming gRPC and applies them to the Kubernetes cluster.
//
// # Architecture
//
// The reflector follows a push-based architecture where the control plane
// pushes state changes to the krane agent, which then reflects those changes
// in Kubernetes through custom resources. This approach ensures consistency
// even when Kubernetes events are missed or the controller is restarted.
//
// # Key Components
//
//   - [Reflector]: Main struct handling control plane event processing
//   - [applySentinel]: Creates or updates Sentinel custom resources
//   - [deleteSentinel]: Handles sentinel resource deletion
//   - [refreshCurrentSentinels]: Periodic sync for consistency
//   - [ensureNamespaceExists]: Creates target namespaces for sentinel resources
//
// # Event Processing Flow
//
//  1. Control plane streams sentinel state changes via gRPC
//  2. Events are buffered to handle high throughput and backpressure
//  3. Reflector processes events and creates/updates/deletes Sentinel CRDs
//  4. Reconciler handles the actual Kubernetes resource management
//  5. Periodic refresh ensures eventual consistency
//
// # Why This Architecture
//
// The push-based reflector architecture was chosen over traditional Kubernetes
// watch patterns because:
//   - Sentinel state originates from the control plane database
//   - We need bidirectional sync between database and cluster
//   - Control plane events provide authoritative source of truth
//   - Periodic refresh handles missed events and network partitions
//
// # Error Handling
//
// The reflector handles different types of errors appropriately:
//   - Validation errors: Reject invalid requests immediately
//   - Namespace conflicts: Create missing namespaces automatically
//   - Resource conflicts: Update existing resources instead of failing
//   - Network errors: Buffer events and retry automatically
//   - Not found errors: Graceful handling during deletion operations
package reflector
