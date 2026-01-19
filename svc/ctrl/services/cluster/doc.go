// Package cluster implements the gRPC ClusterService for synchronizing desired state to edge nodes.
//
// # Overview
//
// Edge nodes (running in different regions) connect to the control plane and request
// state synchronization via the Sync RPC. The control plane streams the desired state
// for deployments and sentinels that should run in that region.
//
// # State Synchronization Model
//
// The synchronization uses a sequence-based approach:
//
//  1. Each state change (create, update, delete) is recorded in the state_changes table
//     with an auto-incrementing sequence number per region.
//
//  2. Edge nodes track the last sequence they've seen. On reconnect, they request changes
//     after that sequence.
//
//  3. If sequence is 0 (new node or reset), a full bootstrap is performed: all running
//     deployments and sentinels are streamed. Stream close signals completion.
//
// # Convergence Guarantees
//
// The system guarantees eventual consistency through:
//   - Idempotent apply/delete operations: applying the same state multiple times is safe
//   - Delete-if-uncertain semantics: if we cannot prove a resource should run in a region,
//     we instruct deletion to prevent stale resources
//   - Reconnection with last-seen sequence: clients catch up on missed changes
//
// # Key Types
//
// The main service type is [Service], which implements [ctrlv1connect.ClusterServiceHandler].
// The primary RPC is [Service.Sync] which handles state synchronization.
package cluster
