package db

// IsTerminal reports whether a deployment has reached a final lifecycle
// state. New statuses default to non-terminal: callers that cancel
// in-flight work (project delete, compensation rollback) should treat
// unknown statuses as still active rather than silently skip them.
func (s DeploymentsStatus) IsTerminal() bool {
	switch s {
	case DeploymentsStatusReady,
		DeploymentsStatusFailed,
		DeploymentsStatusSkipped,
		DeploymentsStatusStopped,
		DeploymentsStatusSuperseded,
		DeploymentsStatusCancelled:
		return true
	case DeploymentsStatusPending,
		DeploymentsStatusStarting,
		DeploymentsStatusBuilding,
		DeploymentsStatusDeploying,
		DeploymentsStatusNetwork,
		DeploymentsStatusFinalizing,
		DeploymentsStatusAwaitingApproval:
		return false
	}
	return false
}

// TerminalDeploymentStatuses enumerates every status that ends the
// deployment lifecycle. Single source of truth for SQL queries that
// filter against the terminal set (ListActiveDeploymentsByProjectId,
// UpdateDeploymentStatusIfActive). Must stay in sync with IsTerminal;
// the test in deployment_status_test.go enforces that.
var TerminalDeploymentStatuses = []DeploymentsStatus{
	DeploymentsStatusReady,
	DeploymentsStatusFailed,
	DeploymentsStatusSkipped,
	DeploymentsStatusStopped,
	DeploymentsStatusSuperseded,
	DeploymentsStatusCancelled,
}
