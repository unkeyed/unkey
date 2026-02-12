// Package deployment provides a Restate virtual object that serialises all
// mutations targeting a single deployment.
//
// Multiple actors may need to mutate a deployment concurrently — the deploy
// workflow scheduling a standby transition, a cron job scaling down idle
// previews, an operator intervening manually, or future operations yet to be
// added. Without coordination these requests would race, potentially
// overwriting each other or applying changes in the wrong order.
//
// # Serialisation via Restate Virtual Objects
//
// The package solves this with a Restate virtual object keyed by deployment ID.
// Restate guarantees that all calls to the same virtual object key are
// processed sequentially: if two requests arrive for the same deployment at
// the same time, Restate queues one until the other completes. This eliminates
// the need for external locks or optimistic-concurrency checks in the
// database and makes it safe to add new operations to this object without
// worrying about cross-operation races.
//
// # Last-Writer-Wins for Scheduled State Changes
//
// Sequential execution alone is not enough for delayed operations. Consider a
// deployment that is scheduled for standby in 30 minutes, but five minutes
// later someone requests it to stay running. The delayed call is already
// enqueued in Restate and will fire regardless. Cancelling Restate timers is
// not possible, so the package uses a nonce-based last-writer-wins mechanism
// instead.
//
// When [VirtualObject.ScheduleDesiredStateChange] is called, it generates a
// unique nonce, writes a transition record (nonce + target state) into Restate
// state, and sends a delayed [VirtualObject.ChangeDesiredState] call carrying
// that nonce. If ScheduleDesiredStateChange is called again before the delay
// elapses, it overwrites the stored transition with a new nonce. When the
// first delayed ChangeDesiredState finally executes, it compares its nonce to
// the stored one, sees a mismatch, and returns without making any changes.
// Only the most recently scheduled transition's nonce will match, so it is
// the only one that takes effect.
//
// [VirtualObject.ClearScheduledStateChanges] removes the stored transition
// entirely, which causes any in-flight delayed call to no-op because there is
// no transition record left to match against.
package deployment
