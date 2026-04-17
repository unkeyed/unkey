// Package sentinel provides Restate services for deploying and rolling out
// sentinel configuration changes.
//
// Two virtual objects live in this package:
//
// [Service] (keyed by sentinel ID) handles individual sentinel deploys.
// It updates the sentinel's config, writes an outbox entry to trigger Krane,
// and suspends on a Restate awakeable until Krane reports the sentinel as
// healthy (via [Service.NotifyReady]). If the sentinel does not become healthy
// within 10 minutes, the deploy is marked as failed.
//
// [RolloutService] (keyed by "singleton") orchestrates fleet-wide image
// rollouts. It splits sentinels into percentage-based waves and fans out
// [Service.Deploy] calls within each wave. If any sentinel in a wave fails,
// the rollout pauses for operator intervention. The operator can resume,
// cancel, or rollback all successfully updated sentinels.
//
// # Usage
//
// Deploy a single sentinel (blocking):
//
//	resp, err := hydrav1.NewSentinelServiceClient(ctx, sentinelID).
//	    Deploy().
//	    Request(&hydrav1.SentinelServiceDeployRequest{Image: "ghcr.io/unkeyed/sentinel:v1.2.3"})
//
// Start a fleet rollout:
//
//	hydrav1.NewSentinelRolloutServiceClient(ctx, "singleton").
//	    Rollout().
//	    Send(&hydrav1.SentinelRolloutServiceRolloutRequest{Image: "ghcr.io/unkeyed/sentinel:v1.2.3"})
package sentinel
