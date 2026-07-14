// Package deploybilling reports month-to-date Deploy usage to the billing
// provider. It has two restate services:
//
//   - CronService.RunDeployBillingPush / RunDeployBillingClose /
//     CloseDeployBillingWorkspace (Handler): the orchestrators. Each computes
//     the month-to-date total for the five Deploy meters (CPU, memory, egress,
//     disk, active keys) from Heimdall checkpoint and key-verification data in
//     ClickHouse, resolves the billable workspaces, and fans out one push per
//     workspace. The invoice.created webhook dispatches CloseDeployBillingWorkspace
//     per renewal invoice; the 00:30 UTC backup cron runs RunDeployBillingClose
//     as a fleet sweep.
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
// Both orchestrators fan out one push per workspace and await the outcomes,
// withholding their heartbeat when any child failed. Retries and failure
// isolation still live in the child invocations (a broken push surfaces as
// its own failed invocation, not a flag buried in a batch); awaiting only
// surfaces success or failure to monitoring. The month-end close depends on
// the await for correctness, because the closing invoice must see the final
// total before we finalize it. A push that fails there leaves the workspace's
// draft open for the backup close rather than finalizing a stale hourly
// value.
//
// The orchestrators are keyed by billing period "YYYY-MM" for fleet sweeps
// and by workspace id for per-invoice close, so concurrent triggers for the
// same month serialize while different months stay independent. They are a
// no-op when ClickHouse is not configured.
package deploybilling
