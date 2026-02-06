---
title: versioning
description: "provides per-region version counters for state synchronization"
---

Package versioning provides per-region version counters for state synchronization.

The \[Service] is a Restate virtual object that generates monotonically increasing version numbers. These versions are used to track state changes in deployments and sentinels tables, enabling efficient incremental synchronization between the control plane and edge agents (krane).

### Why Per-Region Versioning

This service uses the region name as the virtual object key, creating one version counter per region. This design allows version requests for different regions to be processed in parallel while maintaining strict ordering within each region. A global counter would serialize all writes across all regions, creating a bottleneck. The per-region approach matches the data partitioning in the deployments and sentinels tables.

### Usage

Before mutating a deployment or sentinel, pass the region as the virtual object key:

	client := hydrav1.NewVersioningServiceClient(ctx, region)
	resp, err := client.NextVersion().Request(&hydrav1.NextVersionRequest{})
	if err != nil {
	    // Restate errors indicate infrastructure problems; fail the operation
	    return err
	}
	// Use resp.Version when inserting/updating the resource row

Edge agents track their last-seen version and request changes since then:

	SELECT * FROM deployments WHERE region = ? AND version > ? ORDER BY version

### Stale Cursor Detection

If a client's cursor version is older than the minimum retained version in the database (due to compaction or cleanup), it must perform a full bootstrap. Use \[Service.GetVersion] to check the current version without incrementing.

## Constants

versionStateKey is the Restate state key used to store the version counter. Each virtual object instance (keyed by region) has its own isolated state, so this key is scoped to the region.
```go
const versionStateKey = "version"
```


## Variables

```go
var _ hydrav1.VersioningServiceServer = (*Service)(nil)
```


## Types

### type Service

```go
type Service struct {
	hydrav1.UnimplementedVersioningServiceServer
}
```

Service provides per-region, monotonically increasing versions for state sync.

This is a Restate virtual object that maintains a durable counter per region. Each call to \[Service.NextVersion] atomically increments and returns the next version number for that region, with exactly-once semantics guaranteed by Restate. The service is stateless; all state is stored in Restate's virtual object storage.

#### func New

```go
func New() *Service
```

New creates a new versioning service instance. The returned service should be registered with a Restate router using \[hydrav1.NewVersioningServiceServer].

#### func (Service) GetVersion

```go
func (s *Service) GetVersion(ctx restate.ObjectContext, _ *hydrav1.GetVersionRequest) (*hydrav1.GetVersionResponse, error)
```

GetVersion returns the current version without incrementing.

This is useful for stale cursor detection: if a client's cursor version is older than the minimum retained version in the database (due to row deletion or compaction), the client must perform a full bootstrap instead of incremental sync.

Returns 0 if no versions have been generated yet for this region.

#### func (Service) NextVersion

```go
func (s *Service) NextVersion(ctx restate.ObjectContext, _ *hydrav1.NextVersionRequest) (*hydrav1.NextVersionResponse, error)
```

NextVersion atomically increments and returns the next version number.

The version is durably stored in Restate's virtual object state, guaranteeing monotonically increasing values within each region. Version numbers start at 1 (the first call to a new region returns 1, not 0). Restate guarantees exactly-once execution, so retried invocations return the same version that was originally assigned.

Returns an error only if Restate state operations fail, which indicates an infrastructure problem. On success, the returned version is always positive.

