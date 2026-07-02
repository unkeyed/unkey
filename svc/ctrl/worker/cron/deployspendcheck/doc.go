// Package deployspendcheck implements the Compute spend-cap check: it prices
// each budgeted workspace's month-to-date Deploy usage from ClickHouse, computes
// the net-of-credit overage against the configured budget, and emails the
// workspace's admins at 50/75/100%.
//
// It is split like the deploy billing push: a CronService orchestrator
// (RunDeploySpendCheck, see handler.go) lists the opt-in set and fans out to
// DeploySpendCheckService (CheckWorkspaceSpend, see check.go), one invocation
// per workspace, so a customer's checks serialize and one broken workspace fails
// in isolation. The alert email lives in alert.go, the threshold math in
// thresholds.go.
//
// Notify (ENG-2904) is the email path here. Enforcement (ENG-2923) suspends
// compute at 100% via the ENG-2922 teardown primitive and is wired separately.
package deployspendcheck
