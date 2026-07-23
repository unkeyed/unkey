// Package billingreconcile is the Restate cron layer that runs the Compute
// billing reconcile engine (svc/ctrl/internal/billingreconcile) fleet-wide for
// a closed billing period.
//
// It is one handler on the unified hydra.v1.CronService virtual object,
// RunDeployBillingReconcile, keyed by the CLOSED period behind a task-slug
// prefix ("deploy-billing-reconcile-YYYY-MM") so it does not share a
// serialization queue with the other period-keyed tasks (billing push, spend
// check, quota check). It fires monthly at ~T+72h, a few days after month end,
// past Stripe's 48h invoice auto-finalize backstop so every invoice is final.
//
// The pass enumerates billable workspaces from OUR DB once
// (ListDeployBillableWorkspaces, the same query the close finalizes against)
// and fans out one journaled reconcile per workspace. The engine is read-only
// and idempotent, so the whole pass writes nothing and is safe to re-run: a
// replay reuses each workspace's journaled Result instead of re-hitting Stripe.
//
// Scope for now is the DB->Stripe direction only: it finds workspaces whose
// invoice is missing, mispriced, or drifted from live usage. The reverse
// direction (orphan Stripe subscriptions with no matching DB workspace) is
// deferred; see the TODO in buildRefs.
//
// Output: per-workspace verdict to a structured log line (workspace id,
// invoice id, verdict, per-finding
// meter/class/drift); a single monthly summary line
// with counts by verdict; and a Slack page for every structural finding (a code
// or catalog bug, not a money decision), falling back to an error-level log when
// no Slack webhook is configured.
//
// The heartbeat asserts completion only. It pings when the pass finishes
// regardless of what drift was found: drift is a finding, never a failed run.
// It is withheld only when a reconcile actually failed to run -- an engine read
// error (Stripe outage, ClickHouse timeout) surfaces as a returned error, the
// invocation retries, and a persistent failure trips the dead-man switch.
package billingreconcile
