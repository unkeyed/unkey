// Package sentinelcontroller provides Kubernetes controller management for
// Sentinel custom resources.
//
// This package implements a Kubernetes controller that manages Sentinel resources,
// which are edge components providing monitoring, routing, and security functions
// for Unkey deployments. Sentinels operate alongside application deployments to
// provide runtime observability and control.
//
// # Controller Architecture
//
// The sentinel controller follows the same pattern as the deployment controller:
//  1. Watch: Monitors Sentinel resources for changes
//  2. Reconcile: Processes events and applies desired state
//  3. Buffer: Queues events to handle high throughput and backpressure
//  4. Apply: Creates/updates Kubernetes resources based on specifications
//
// # Sentinel Functionality
//
// Sentinels provide several key functions:
//   - Traffic routing and load balancing
//   - Request monitoring and analytics
//   - Security policy enforcement
//   - API gateway functionality
//   - Health checking and circuit breaking
//
// # Resource Management
//
// The controller manages the complete lifecycle of sentinel resources including:
//   - Creating StatefulSets for sentinel workloads
//   - Managing Services for network exposure
//   - Configuring routing rules and policies
//   - Handling resource cleanup on deletion
//   - Updating status and conditions
//
// # Integration
//
// The controller integrates with the krane sync engine to receive sentinel
// events from the control plane and report status updates back. It uses the
// k8s package for standardized label management and resource operations.
//
// # Usage
//
// The controller is typically created by the main krane agent and runs
// as part of the controller-runtime manager:
//
//	cfg := sentinelcontroller.Config{
//		Logger: logger,
//		Events: eventBuffer,
//	}
//	controller, err := sentinelcontroller.New(cfg)
package sentinelcontroller
