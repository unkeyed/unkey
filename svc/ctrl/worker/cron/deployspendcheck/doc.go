// Package deployspendcheck implements the Compute spend-cap check: it prices
// every budgeted workspace's month-to-date Deploy usage, compares the gross
// total spend against the configured budget, and emails the workspace's admins
// at 50/75/100%. The cap is on total metered spend, credits included.
//
// It is split like the deploy billing push: a CronService orchestrator
// (RunDeploySpendCheck, see handler.go) lists the opt-in set, prices it from
// one grouped ClickHouse scan scoped to those workspaces, and fans out to
// DeploySpendCheckService (CheckWorkspaceSpend, see check.go) only for
// workspaces whose gross spend has reached the lowest alert threshold, one
// awaited invocation per workspace with the priced gross in the request.
// Per-workspace ClickHouse point queries would scale with the budgeted fleet;
// one scoped scan plus a near-budget fan-out scales with the workspaces that
// can actually act. A customer's checks still
// serialize on its VO and one broken workspace still fails in isolation. The
// alert email lives in alert.go, the threshold math in thresholds.go.
//
// Notify (ENG-2904) is the email path here. Enforcement (ENG-2923) suspends
// compute at 100% via the ENG-2922 teardown primitive and is wired separately.
//
// Local dev runs the check every 3 minutes; production runs every minute
// (infra repo). Full design:
// docs/engineering/architecture/services/control-plane/worker/workflows/deploy-spend-cap.mdx
// (added by the docs PR at the top of this stack).
package deployspendcheck
