// Package cluster implements the Connect ClusterService for synchronizing desired state to krane agents.
//
// # Overview
//
// Krane agents (similar to kubelets in Kubernetes) run in each region and act as managers
// for their respective Kubernetes clusters. They connect to the control plane and request
// state synchronization via the Sync RPC. The control plane streams the desired state
// for deployments and sentinels that should run in that region.
//
// # State Synchronization Model
//
// The synchronization uses a version-based approach:
//
//  1. Each resource (deployment_topology, sentinel) has a version column that is updated
//     on every mutation via the Restate VersioningService singleton.
//
//  2. Krane agents track the last version they've seen. On reconnect, they request changes
//     after that version.
//
//  3. If version is 0 (new agent or reset), a full bootstrap is performed: all resources
//     are streamed ordered by version. Stream close signals completion.
//
// # Convergence Guarantees
//
// The system guarantees eventual consistency through:
//   - Idempotent apply/delete operations: applying the same state multiple times is safe
//   - Soft-delete semantics: "deletes" set desired_replicas=0, keeping the row with its version
//   - Bootstrap + GC: after bootstrap, agents delete any k8s resources not in the stream
//   - Reconnection with last-seen version: agents catch up on missed changes
//
// # Key Types
//
// The main service type is [Service], which implements [ctrlv1connect.ClusterServiceHandler].
// The primary RPCs are [Service.WatchDeployments] and [Service.WatchSentinels] for streaming
// state changes, and [Service.ReportDeploymentStatus] and [Service.ReportSentinelStatus] for
// receiving agent status updates.
package cluster
