package db

import mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

// The deployment lifecycle enums live in pkg/mysql/types so pkg/db and
// svc/ctrl/internal/db share one type instead of each generating its own copy
// (which forced casting at every boundary). The sqlc go_type override points the
// generated deployments.status/desired_state columns at that package; these
// aliases and re-exports keep the db.DeploymentsStatus* call sites unchanged. See
// pkg/mysql/types/deployment_status.go for IsTerminal and the status slices.

type (
	DeploymentsStatus       = mysqltype.DeploymentsStatus
	DeploymentsDesiredState = mysqltype.DeploymentsDesiredState
)

const (
	DeploymentsStatusPending          = mysqltype.DeploymentsStatusPending
	DeploymentsStatusStarting         = mysqltype.DeploymentsStatusStarting
	DeploymentsStatusBuilding         = mysqltype.DeploymentsStatusBuilding
	DeploymentsStatusDeploying        = mysqltype.DeploymentsStatusDeploying
	DeploymentsStatusNetwork          = mysqltype.DeploymentsStatusNetwork
	DeploymentsStatusFinalizing       = mysqltype.DeploymentsStatusFinalizing
	DeploymentsStatusReady            = mysqltype.DeploymentsStatusReady
	DeploymentsStatusFailed           = mysqltype.DeploymentsStatusFailed
	DeploymentsStatusSkipped          = mysqltype.DeploymentsStatusSkipped
	DeploymentsStatusAwaitingApproval = mysqltype.DeploymentsStatusAwaitingApproval
	DeploymentsStatusStopped          = mysqltype.DeploymentsStatusStopped
	DeploymentsStatusSuperseded       = mysqltype.DeploymentsStatusSuperseded
	DeploymentsStatusCancelled        = mysqltype.DeploymentsStatusCancelled

	DeploymentsDesiredStateRunning = mysqltype.DeploymentsDesiredStateRunning
	DeploymentsDesiredStateStopped = mysqltype.DeploymentsDesiredStateStopped
)

var (
	TerminalDeploymentStatuses    = mysqltype.TerminalDeploymentStatuses
	ProgressingDeploymentStatuses = mysqltype.ProgressingDeploymentStatuses
	AllDeploymentStatuses         = mysqltype.AllDeploymentStatuses
)
