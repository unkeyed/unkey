// Package versioning provides per-region version counters for state synchronization.
//
// The VersioningService is a Restate virtual object that generates monotonically
// increasing version numbers. These versions are used to track state changes in
// deployments and sentinels tables, enabling efficient incremental synchronization
// between the control plane and edge agents (krane).
//
// # Usage
//
// Before mutating a deployment or sentinel, pass the region as the virtual object key:
//
//	client := hydrav1.NewVersioningServiceClient(ctx, region)
//	resp, err := client.NextVersion(ctx, &hydrav1.NextVersionRequest{})
//	// Use resp.Version when updating the resource row
//
// Edge agents track their last-seen version and request changes after it:
//
//	SELECT * FROM deployments WHERE region = ? AND version > ? ORDER BY version
//
// # Per-Region Pattern
//
// This service uses the region name as the virtual object key, creating one
// version counter per region. This allows version requests for different regions
// to be processed in parallel while maintaining ordering within each region.
package versioning
