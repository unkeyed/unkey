// Package cluster implements the Connect ClusterService for synchronizing desired state
// between the control plane and krane agents running in regional Kubernetes clusters.
//
// # Overview
//
// Krane agents run in each region and manage their local Kubernetes clusters. They maintain
// long-lived streaming connections to the control plane, receiving desired state for
// deployments and sentinels via [Service.WatchDeployments] and [Service.WatchSentinels].
// Agents report observed state back through [Service.ReportDeploymentStatus] and
// [Service.ReportSentinelStatus], enabling drift detection and health tracking.
//
// # State Synchronization Model
//
// Synchronization uses version-based cursors for resumable streaming. Each resource
// (deployment_topology, sentinel) has a version column updated on every mutation. Agents
// track the maximum version seen and reconnect with that version to resume without
// replaying history. When version is 0 (new agent or reset), all resources are streamed
// in version order for a full bootstrap.
//
// The streaming RPCs poll the database every second when no new versions are available.
// Each poll fetches up to 100 resources with versions greater than the cursor.
//
// # Convergence Guarantees
//
// The system achieves eventual consistency through idempotent operations: agents can
// safely apply the same state multiple times. Deletes use soft-delete semantics by
// setting desired state to archived or standby, preserving the version for streaming.
// After bootstrap, agents garbage-collect any Kubernetes resources not present in the
// stream, ensuring convergence even if messages were missed.
//
// # Authentication
//
// All RPCs require bearer token authentication via the Authorization header. Agents must
// also provide their region in the X-Krane-Region header for region-scoped operations.
//
// # Key Types
//
// [Service] implements [ctrlv1connect.ClusterServiceHandler] with six RPCs: two streaming
// watchers ([Service.WatchDeployments], [Service.WatchSentinels]), two point queries
// ([Service.GetDesiredDeploymentState], [Service.GetDesiredSentinelState]), and two
// status reporters ([Service.ReportDeploymentStatus], [Service.ReportSentinelStatus]).
// Configuration is provided through [Config].
package cluster
