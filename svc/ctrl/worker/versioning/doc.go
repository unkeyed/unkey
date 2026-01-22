// Package versioning provides per-region version counters for state synchronization.
//
// The [Service] is a Restate virtual object that generates monotonically increasing
// version numbers. These versions are used to track state changes in deployments and
// sentinels tables, enabling efficient incremental synchronization between the
// control plane and edge agents (krane).
//
// # Why Per-Region Versioning
//
// This service uses the region name as the virtual object key, creating one version
// counter per region. This design allows version requests for different regions to
// be processed in parallel while maintaining strict ordering within each region.
// A global counter would serialize all writes across all regions, creating a
// bottleneck. The per-region approach matches the data partitioning in the
// deployments and sentinels tables.
//
// # Usage
//
// Before mutating a deployment or sentinel, pass the region as the virtual object key:
//
//	client := hydrav1.NewVersioningServiceClient(ctx, region)
//	resp, err := client.NextVersion().Request(&hydrav1.NextVersionRequest{})
//	if err != nil {
//	    // Restate errors indicate infrastructure problems; fail the operation
//	    return err
//	}
//	// Use resp.Version when inserting/updating the resource row
//
// Edge agents track their last-seen version and request changes since then:
//
//	SELECT * FROM deployments WHERE region = ? AND version > ? ORDER BY version
//
// # Stale Cursor Detection
//
// If a client's cursor version is older than the minimum retained version in the
// database (due to compaction or cleanup), it must perform a full bootstrap. Use
// [Service.GetVersion] to check the current version without incrementing.
package versioning
