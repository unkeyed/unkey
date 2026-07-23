// Package billingreconcile reconciles one workspace's finalized Deploy Stripe
// invoice for one calendar-month billing period against live ClickHouse usage.
//
// There is no persisted snapshot to compare against. The finalized invoice is
// the durable record of what was billed (the hourly/close push sets its line
// quantities directly from ClickHouse, so "intended" and "billed" are the same
// object), and a live ClickHouse re-derivation of the period's usage is the
// recorded side, valid because reconcile runs well inside the checkpoint
// retention window. ReconcileWorkspace compares the two and returns a [Verdict]
// plus flat per-check [Finding]s.
//
// The engine only classifies. It reads Stripe and ClickHouse through the seams
// in types.go, takes a plain context.Context (nothing to journal), and does not
// touch any database, schedule itself, page anyone, or emit metrics. Wiring
// this into a cron, fanning it out over every billable workspace, and reacting
// to its verdicts (Slack summary, logs, the response playbook) all belong to a
// caller built on top of this package.
package billingreconcile
