package dbtype

import (
	"database/sql/driver"
	"fmt"
)

// DeploymentsStatus is the canonical deployment lifecycle status enum, shared by
// every db package instead of each sqlc config regenerating its own copy. The
// generated code in pkg/db and svc/ctrl/internal/db points its deployments.status
// column at this type via a go_type override, and each package re-exports it under
// the DeploymentsStatus name so existing call sites are unaffected.
type DeploymentsStatus string

const (
	DeploymentsStatusPending          DeploymentsStatus = "pending"
	DeploymentsStatusStarting         DeploymentsStatus = "starting"
	DeploymentsStatusBuilding         DeploymentsStatus = "building"
	DeploymentsStatusDeploying        DeploymentsStatus = "deploying"
	DeploymentsStatusNetwork          DeploymentsStatus = "network"
	DeploymentsStatusFinalizing       DeploymentsStatus = "finalizing"
	DeploymentsStatusReady            DeploymentsStatus = "ready"
	DeploymentsStatusFailed           DeploymentsStatus = "failed"
	DeploymentsStatusSkipped          DeploymentsStatus = "skipped"
	DeploymentsStatusAwaitingApproval DeploymentsStatus = "awaiting_approval"
	DeploymentsStatusStopped          DeploymentsStatus = "stopped"
	DeploymentsStatusSuperseded       DeploymentsStatus = "superseded"
	DeploymentsStatusCancelled        DeploymentsStatus = "cancelled"
)

func (e *DeploymentsStatus) Scan(src any) error {
	switch s := src.(type) {
	case []byte:
		*e = DeploymentsStatus(s)
	case string:
		*e = DeploymentsStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for DeploymentsStatus: %T", src)
	}
	return nil
}

type NullDeploymentsStatus struct {
	DeploymentsStatus DeploymentsStatus
	Valid             bool // Valid is true if DeploymentsStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullDeploymentsStatus) Scan(value any) error {
	if value == nil {
		ns.DeploymentsStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.DeploymentsStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullDeploymentsStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.DeploymentsStatus), nil
}

// IsTerminal reports whether a deployment has reached a final lifecycle
// state. Acts as the spec the TerminalDeploymentStatuses and
// ProgressingDeploymentStatuses slices must mirror; the test in
// deployment_status_test.go enforces that mapping. Unknown statuses fall
// through to false; the slices, not this function, drive cancellation
// decisions.
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
// guard transitions against terminal rows (UpdateDeploymentStatusIfActive).
// Must stay in sync with IsTerminal; the test in
// deployment_status_test.go enforces that.
var TerminalDeploymentStatuses = []DeploymentsStatus{
	DeploymentsStatusReady,
	DeploymentsStatusFailed,
	DeploymentsStatusSkipped,
	DeploymentsStatusStopped,
	DeploymentsStatusSuperseded,
	DeploymentsStatusCancelled,
}

// ProgressingDeploymentStatuses enumerates every status that represents
// an in-flight deployment. Single source of truth for SQL queries that
// cancel in-progress work (ListProgressingDeploymentsByEnvironmentId).
// Cancellation is destructive, so this is an explicit allowlist: new
// statuses are not cancelled by default until intentionally added here.
// Must stay in sync with IsTerminal; the test in
// deployment_status_test.go enforces that.
var ProgressingDeploymentStatuses = []DeploymentsStatus{
	DeploymentsStatusPending,
	DeploymentsStatusStarting,
	DeploymentsStatusBuilding,
	DeploymentsStatusDeploying,
	DeploymentsStatusNetwork,
	DeploymentsStatusFinalizing,
	DeploymentsStatusAwaitingApproval,
}

// ActiveComputeDeploymentStatuses enumerates the statuses a running-intent
// deployment holds while it has, or is on its way to having, live compute:
// every in-flight status plus the ready steady state. Teardown selects exactly
// these; the drain-terminal statuses (failed/stopped/cancelled/superseded/
// skipped) have no compute to stop. Ready is terminal for the deploy workflow
// but live here, so this is its own set derived from ProgressingDeploymentStatuses
// rather than a reuse of the terminal slice.
var ActiveComputeDeploymentStatuses = append(
	append([]DeploymentsStatus{}, ProgressingDeploymentStatuses...),
	DeploymentsStatusReady,
)

// AllDeploymentStatuses lists every value of the DeploymentsStatus enum.
// Exists so deployment_status_test.go does not maintain a parallel copy:
// adding a new status here forces classification in IsTerminal and
// membership in exactly one of the Terminal/Progressing slices.
var AllDeploymentStatuses = []DeploymentsStatus{
	DeploymentsStatusPending,
	DeploymentsStatusStarting,
	DeploymentsStatusBuilding,
	DeploymentsStatusDeploying,
	DeploymentsStatusNetwork,
	DeploymentsStatusFinalizing,
	DeploymentsStatusReady,
	DeploymentsStatusFailed,
	DeploymentsStatusSkipped,
	DeploymentsStatusAwaitingApproval,
	DeploymentsStatusStopped,
	DeploymentsStatusSuperseded,
	DeploymentsStatusCancelled,
}
