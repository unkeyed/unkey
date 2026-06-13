// Package deploybilling reports month-to-date Deploy usage to the billing
// provider. It has two restate services:
//
//   - CronService.RunDeployBillingPush / RunDeployBillingClose (Handler): the
//     orchestrators. Each computes the month-to-date total for the five Deploy
//     meters (CPU, memory, egress, disk, active keys) from Heimdall checkpoint
//     and key-verification data in ClickHouse, resolves the billable
//     workspaces, and fans out one push per workspace.
//   - DeployBillingPushService.PushWorkspaceUsage (PushHandler): the per-
//     workspace push. Each runs as its own invocation, keyed by workspace id,
//     so a customer's pushes serialize and a broken workspace (deleted
//     customer, frozen test clock) retries and fails in isolation without
//     blocking the others or the orchestrating tick.
//
// The pusher sets (not increments) the period quantity, so sending the
// absolute month-to-date value is idempotent: a retry, an overlapping tick, or
// a replay all converge on the same number. There are no per-event deltas to
// dedupe and no end-of-month timing window; the last value the provider
// received before invoice finalize is the one it bills.
//
// The hourly orchestrator fans out fire-and-forget: it hands each push off and
// completes once dispatched, leaving retries and failure to the child
// invocations (a stuck push surfaces as its own failed invocation, not a flag
// buried in a batch). The month-end close awaits each push instead, because
// the closing invoice must see the final total before it is finalized; a push
// that fails there is tolerated (the invoice keeps its last hourly value
// rather than being stranded in draft) and the close heartbeat is withheld so
// monitoring fires.
//
// The orchestrators are keyed by billing period "YYYY-MM" so concurrent
// triggers for the same month serialize while different months stay
// independent. They are a no-op when ClickHouse is not configured.
package deploybilling
