// Package deployspendcheck implements the Compute spend-cap check: it prices
// every budgeted workspace's month-to-date Deploy usage, computes the
// net-of-credit overage against the configured budget, and emails the
// workspace's admins at 50/75/100%.
//
// It is split like the deploy billing push: a CronService orchestrator
// (RunDeploySpendCheck, see handler.go) lists the opt-in set, prices everyone
// from one grouped ClickHouse scan, and fans out to DeploySpendCheckService
// (CheckWorkspaceSpend, see check.go) only for workspaces whose overage has
// reached the lowest alert threshold, one invocation per workspace with the
// priced gross in the request. Per-workspace ClickHouse point queries would
// scale with the budgeted fleet; one scan plus a near-budget fan-out scales
// with the workspaces that can actually act. A customer's checks still
// serialize on its VO and one broken workspace still fails in isolation. The
// alert email lives in alert.go, the threshold math in thresholds.go.
//
// Notify (ENG-2904) is the email path here. Enforcement (ENG-2923) suspends
// compute at 100% via the ENG-2922 teardown primitive and is wired separately.
package deployspendcheck
