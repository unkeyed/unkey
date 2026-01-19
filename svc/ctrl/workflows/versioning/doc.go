// Package versioning provides a global version counter for state synchronization.
//
// The VersioningService is a Restate virtual object that generates monotonically
// increasing version numbers. These versions are used to track state changes in
// deployments and sentinels tables, enabling efficient incremental synchronization
// between the control plane and edge agents (krane).
//
// # Usage
//
// Before mutating a deployment or sentinel:
//
//	client := hydrav1.NewVersioningServiceClient(ctx, "")
//	resp, err := client.NextVersion(ctx, &hydrav1.NextVersionRequest{})
//	// Use resp.Version when updating the resource row
//
// Edge agents track their last-seen version and request changes after it:
//
//	SELECT * FROM deployments WHERE region = ? AND version > ? ORDER BY version
//
// # Singleton Pattern
//
// This service uses an empty string as the virtual object key, making it a
// singleton. All version requests are serialized through a single instance,
// guaranteeing global ordering.
package versioning
