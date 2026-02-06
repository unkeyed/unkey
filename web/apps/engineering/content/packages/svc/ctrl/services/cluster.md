---
title: cluster
description: "implements the Connect ClusterService for synchronizing desired state"
---

Package cluster implements the Connect ClusterService for synchronizing desired state between the control plane and krane agents running in regional Kubernetes clusters.

### Overview

Krane agents run in each region and manage their local Kubernetes clusters. They maintain long-lived streaming connections to the control plane, receiving desired state for deployments and sentinels via \[Service.WatchDeployments] and \[Service.WatchSentinels]. Agents report observed state back through \[Service.ReportDeploymentStatus] and \[Service.ReportSentinelStatus], enabling drift detection and health tracking.

### State Synchronization Model

Synchronization uses version-based cursors for resumable streaming. Each resource (deployment\_topology, sentinel) has a version column updated on every mutation. Agents track the maximum version seen and reconnect with that version to resume without replaying history. When version is 0 (new agent or reset), all resources are streamed in version order for a full bootstrap.

The streaming RPCs poll the database every second when no new versions are available. Each poll fetches up to 100 resources with versions greater than the cursor.

### Convergence Guarantees

The system achieves eventual consistency through idempotent operations: agents can safely apply the same state multiple times. Deletes use soft-delete semantics by setting desired state to archived or standby, preserving the version for streaming. After bootstrap, agents garbage-collect any Kubernetes resources not present in the stream, ensuring convergence even if messages were missed.

### Authentication

All RPCs require bearer token authentication via the Authorization header. Agents must also provide their region in the X-Krane-Region header for region-scoped operations.

### Key Types

\[Service] implements \[ctrlv1connect.ClusterServiceHandler] with six RPCs: two streaming watchers (\[Service.WatchDeployments], \[Service.WatchSentinels]), two point queries (\[Service.GetDesiredDeploymentState], \[Service.GetDesiredSentinelState]), and two status reporters (\[Service.ReportDeploymentStatus], \[Service.ReportSentinelStatus]). Configuration is provided through \[Config].

## Variables

```go
var _ ctrlv1connect.ClusterServiceHandler = (*Service)(nil)
```


## Functions


## Types

### type Config

```go
type Config struct {
	// Database provides read and write access for querying and updating resource state.
	Database db.Database

	// Bearer is the authentication token that agents must provide in the Authorization header.
	Bearer string
}
```

Config holds the configuration for creating a new cluster \[Service].

### type Service

```go
type Service struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	db     db.Database
	bearer string
}
```

Service implements \[ctrlv1connect.ClusterServiceHandler] to synchronize desired state between the control plane and krane agents. It provides streaming RPCs for watching deployment and sentinel changes, point queries for fetching individual resource states, and status reporting endpoints for agents to report observed state back to the control plane.

#### func New

```go
func New(cfg Config) *Service
```

New creates a new cluster \[Service] with the given configuration. The returned service is ready to be registered with a Connect server.

#### func (Service) GetDesiredCiliumNetworkPolicyState

```go
func (s *Service) GetDesiredCiliumNetworkPolicyState(
	ctx context.Context,
	req *connect.Request[ctrlv1.GetDesiredCiliumNetworkPolicyStateRequest],
) (*connect.Response[ctrlv1.CiliumNetworkPolicyState], error)
```

GetDesiredCiliumNetworkPolicyState returns the target state for a single Cilium network policy in the caller's region. This is a point query alternative to \[Service.WatchCiliumNetworkPolicies] for cases where an agent needs to fetch state for a specific policy rather than streaming all changes.

Returns CodeUnauthenticated if bearer token is invalid, CodeInvalidArgument if the X-Krane-Region header is missing, CodeNotFound if no policy exists with the given ID in the specified region, or CodeInternal for database errors.

#### func (Service) GetDesiredDeploymentState

```go
func (s *Service) GetDesiredDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error)
```

GetDesiredDeploymentState returns the target state for a single deployment in the caller's region. This is a point query alternative to \[Service.WatchDeployments] for cases where an agent needs to fetch state for a specific deployment rather than streaming all changes.

The response contains either an ApplyDeployment (for running state) or DeleteDeployment (for archived or standby states) based on the deployment\_topology's desired\_state in the database. Unhandled desired states result in CodeInternal.

Returns CodeUnauthenticated if bearer token is invalid, CodeInvalidArgument if the X-Krane-Region header is missing, CodeNotFound if no deployment exists with the given ID in the specified region, or CodeInternal for database errors or unhandled states.

#### func (Service) GetDesiredSentinelState

```go
func (s *Service) GetDesiredSentinelState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error)
```

GetDesiredSentinelState returns the target state for a single sentinel resource. This is a point query alternative to \[Service.WatchSentinels] for cases where an agent needs to fetch state for a specific sentinel rather than streaming all changes.

The response contains either an ApplySentinel (for running state) or DeleteSentinel (for archived or standby states) based on the sentinel's desired\_state in the database. Unhandled desired states result in CodeInternal.

Returns CodeUnauthenticated if bearer token is invalid, CodeInvalidArgument if the X-Krane-Region header is missing, CodeNotFound if no sentinel exists with the given ID, or CodeInternal for database errors or unhandled states.

#### func (Service) ReportDeploymentStatus

```go
func (s *Service) ReportDeploymentStatus(ctx context.Context, req *connect.Request[ctrlv1.ReportDeploymentStatusRequest]) (*connect.Response[ctrlv1.ReportDeploymentStatusResponse], error)
```

ReportDeploymentStatus reconciles the observed deployment state reported by a krane agent. This is the feedback loop for convergence: agents report what's actually running so the control plane can track instance health and detect drift from desired state.

The request contains either an Update or Delete change. For Update, the method upserts all reported instances and garbage-collects any instances in the database that were not included in the report. For Delete, all instances for the deployment in that region are removed. Both operations run within a retryable transaction to handle transient database errors using \[db.TxRetry].

Instance status is mapped from proto values to database enums via \[ctrlDeploymentStatusToDbStatus]. Unspecified or unknown statuses default to inactive.

Returns CodeUnauthenticated if bearer token is invalid. Database errors during the transaction are returned as-is (not wrapped in Connect error codes).

#### func (Service) ReportSentinelStatus

```go
func (s *Service) ReportSentinelStatus(ctx context.Context, req *connect.Request[ctrlv1.ReportSentinelStatusRequest]) (*connect.Response[ctrlv1.ReportSentinelStatusResponse], error)
```

ReportSentinelStatus records the observed replica count and health for a sentinel as reported by a krane agent. This updates the available\_replicas, health, and updated\_at fields in the database, enabling the control plane to track which sentinels are actually running and their current health state.

The health proto value is mapped to database enums: HEALTH\_HEALTHY to SentinelsHealthHealthy, HEALTH\_UNHEALTHY to SentinelsHealthUnhealthy, HEALTH\_PAUSED to SentinelsHealthPaused, and HEALTH\_UNSPECIFIED to SentinelsHealthUnknown.

Returns CodeUnauthenticated if bearer token is invalid, or CodeInternal if the database update fails.

#### func (Service) WatchCiliumNetworkPolicies

```go
func (s *Service) WatchCiliumNetworkPolicies(
	ctx context.Context,
	req *connect.Request[ctrlv1.WatchCiliumNetworkPoliciesRequest],
	stream *connect.ServerStream[ctrlv1.CiliumNetworkPolicyState],
) error
```

WatchCiliumNetworkPolicies streams Cilium network policy state changes from the control plane to agents. This is the primary mechanism for agents to receive desired state updates for their region. Agents apply received state to Kubernetes to converge actual state toward desired state.

The stream uses version-based cursors for resumability. The client provides version\_last\_seen in the request, and the server streams all policies with versions greater than that cursor. Clients should track the maximum version received and use it to reconnect without replaying history. When no new versions are available, the server polls the database every second.

Each poll fetches up to 100 policy rows ordered by version. Every row is sent as an apply since the control plane only stores active policies today. Rows with empty policy payloads are logged and skipped.

Returns when the context is cancelled, or on database or stream errors.

#### func (Service) WatchDeployments

```go
func (s *Service) WatchDeployments(
	ctx context.Context,
	req *connect.Request[ctrlv1.WatchDeploymentsRequest],
	stream *connect.ServerStream[ctrlv1.DeploymentState],
) error
```

WatchDeployments streams deployment state changes from the control plane to agents. This is the primary mechanism for agents to receive desired state updates for their region. Agents apply received state to Kubernetes to converge actual state toward desired state.

The stream uses version-based cursors for resumability. The client provides version\_last\_seen in the request, and the server streams all deployments with versions greater than that cursor. Clients should track the maximum version received and use it to reconnect without replaying history. When no new versions are available, the server polls the database every second.

Each poll fetches up to 100 deployment topology rows ordered by version. The desired\_status field determines whether to send an ApplyDeployment (for started/starting states) or DeleteDeployment (for stopped/stopping states). Rows with unhandled statuses are logged and skipped.

Returns when the context is cancelled, or on database or stream errors.

#### func (Service) WatchSentinels

```go
func (s *Service) WatchSentinels(
	ctx context.Context,
	req *connect.Request[ctrlv1.WatchSentinelsRequest],
	stream *connect.ServerStream[ctrlv1.SentinelState],
) error
```

WatchSentinels streams sentinel state changes from the control plane to agents. This is the primary mechanism for agents to receive desired state updates for their region. Agents apply received state to Kubernetes to converge actual state toward desired state.

The stream uses version-based cursors for resumability. The client provides version\_last\_seen in the request, and the server streams all sentinels with versions greater than that cursor. Clients should track the maximum version received and use it to reconnect without replaying history. When no new versions are available, the server polls the database every second.

Each poll fetches up to 100 sentinel rows ordered by version. The desired\_state field determines whether to send an ApplySentinel (for running state) or DeleteSentinel (for archived or standby states). Rows with unhandled states are logged and skipped.

Returns when the context is cancelled, or on database or stream errors.

