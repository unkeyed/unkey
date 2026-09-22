// Package cluster implements the Connect ClusterService for synchronizing desired state
// between the control plane and krane agents running in regional Kubernetes clusters.
//
// # Overview
//
// Krane agents run in each region and manage their local Kubernetes clusters. They maintain
// long-lived streaming connections to the control plane, receiving desired state for
// deployments and Cilium network policies. Agents report observed state back through
// [Service.ReportDeploymentStatus], enabling drift detection and health tracking.
//
// # State Synchronization Model
//
// Streaming uses Vitess VStream resume tokens. Deployment topology payloads also
// carry a revision that advances with each desired-state change. Krane uses that
// revision to prevent delayed full-sync payloads from replacing newer streamed state.
//
// # Convergence Guarantees
//
// The system achieves eventual consistency through idempotent operations: agents can
// safely apply the same state multiple times. Periodic point reads repair missed
// updates and remove Kubernetes resources whose topology rows were hard-deleted.
//
// # Authentication
//
// All RPCs require bearer token authentication via the Authorization header. Cluster-scoped
// RPCs carry a [ctrlv1.ClusterKey] (cell ID, platform, and region) on the request message.
//
// # Key Types
//
// [Service] implements [ctrlv1connect.ClusterServiceHandler]. Configuration is provided
// through [Config].
package cluster
