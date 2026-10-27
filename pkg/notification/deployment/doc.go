// Package deployment maps deployment transitions to notification events and reports.
//
// [Change] distinguishes occurrences that can share an event type. [Event]
// validates a version-one [Payload], and [Definitions] supplies its renderers.
//
// # Usage
//
// In a Restate handler, outside restate.Run, enqueue a report without waiting
// for downstream delivery:
//
//	deployment.Report(ctx, deploymentID, deployment.Live)
//
// Use [Event] when constructing an event from already resolved deployment metadata.
package deployment
