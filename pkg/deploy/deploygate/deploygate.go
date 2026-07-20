// Package deploygate holds the deployment lifecycle preconditions shared by the
// three layers that each enforce them: the public API handlers, the ctrl RPC
// services, and the ctrl Restate workflows. Centralizing the invariant here is
// what stops the three copies from drifting — a promote that is legal at the API
// is legal at the worker, by construction.
package deploygate

import "github.com/unkeyed/unkey/pkg/db"

// envProduction is the environment whose deployment serves production traffic:
// promote/rollback require it, stop/start reject it.
const envProduction = "production"

// Each check takes its own input holding exactly the fields it needs. Status and
// DesiredState are typed to the pkg/db enums: the API passes its db row fields
// directly, ctrl converts from its own db package at the boundary.

// PromoteInput is the state CheckPromoteTarget needs.
type PromoteInput struct {
	Status              db.DeploymentsStatus
	DesiredState        db.DeploymentsDesiredState
	EnvironmentSlug     string
	CurrentDeploymentID string
	DeploymentID        string
	IsRolledBack        bool
}

// RollbackInput is the state CheckRollbackTarget needs.
type RollbackInput struct {
	Status              db.DeploymentsStatus
	DesiredState        db.DeploymentsDesiredState
	EnvironmentSlug     string
	CurrentDeploymentID string
	DeploymentID        string
}

// StopInput is the state CheckStopTarget needs.
type StopInput struct {
	Status          db.DeploymentsStatus
	DesiredState    db.DeploymentsDesiredState
	EnvironmentSlug string
}

// StartInput is the state CheckStartTarget needs.
type StartInput struct {
	DesiredState    db.DeploymentsDesiredState
	EnvironmentSlug string
	SpendSuspended  bool
}

// isCurrent reports whether the target is the app's current (live) deployment.
func isCurrent(currentDeploymentID, deploymentID string) bool {
	return currentDeploymentID != "" && currentDeploymentID == deploymentID
}

// TargetFailureReason is why a deployment fails a promote/rollback precondition.
// TargetOK means it is eligible to become the app's current deployment.
type TargetFailureReason int

const (
	TargetOK TargetFailureReason = iota
	TargetNotReady
	TargetDraining
	TargetNotProduction
	TargetNoCurrentDeployment
	TargetAlreadyCurrent
)

// Message returns a caller-facing explanation. It is deliberately generic across
// promote and rollback so both surface identical wording.
func (r TargetFailureReason) Message() string {
	switch r {
	case TargetOK:
		return ""
	case TargetNotReady:
		return "The deployment is not ready."
	case TargetDraining:
		return "The deployment is shutting down and cannot serve traffic."
	case TargetNotProduction:
		return "Only production deployments can be promoted or rolled back."
	case TargetNoCurrentDeployment:
		return "The app has no current deployment."
	case TargetAlreadyCurrent:
		return "The deployment is already the current deployment."
	default:
		return ""
	}
}

// targetCore holds the preconditions common to promote and rollback: the
// deployment must be ready, running (not draining), in production, and its app
// must already have a current deployment. Order matters — it is the order every
// layer reports failures in.
func targetCore(
	status db.DeploymentsStatus,
	desiredState db.DeploymentsDesiredState,
	environmentSlug, currentDeploymentID string,
) TargetFailureReason {
	switch {
	case status != db.DeploymentsStatusReady:
		return TargetNotReady
	case desiredState != db.DeploymentsDesiredStateRunning:
		return TargetDraining
	case environmentSlug != envProduction:
		return TargetNotProduction
	case currentDeploymentID == "":
		return TargetNoCurrentDeployment
	default:
		return TargetOK
	}
}

// CheckPromoteTarget validates a deployment may be promoted to current.
// Promoting the deployment that is already current is rejected, unless the app
// is in a rolled-back state (then it is a rollback confirmation, allowed).
func CheckPromoteTarget(in PromoteInput) TargetFailureReason {
	if r := targetCore(
		in.Status,
		in.DesiredState,
		in.EnvironmentSlug,
		in.CurrentDeploymentID,
	); r != TargetOK {
		return r
	}
	if isCurrent(in.CurrentDeploymentID, in.DeploymentID) && !in.IsRolledBack {
		return TargetAlreadyCurrent
	}
	return TargetOK
}

// CheckRollbackTarget validates a deployment may be rolled back to. Rolling back
// to the deployment that is already current is always a no-op, so it is rejected.
func CheckRollbackTarget(in RollbackInput) TargetFailureReason {
	if r := targetCore(
		in.Status,
		in.DesiredState,
		in.EnvironmentSlug,
		in.CurrentDeploymentID,
	); r != TargetOK {
		return r
	}
	if isCurrent(in.CurrentDeploymentID, in.DeploymentID) {
		return TargetAlreadyCurrent
	}
	return TargetOK
}

// StopFailureReason is why a deployment cannot be stopped. StopOK means it can.
type StopFailureReason int

const (
	StopOK StopFailureReason = iota
	StopNotRunning
	StopAlreadyStopping
	StopIsProduction
)

// Message returns a caller-facing explanation.
func (r StopFailureReason) Message() string {
	switch r {
	case StopOK:
		return ""
	case StopNotRunning:
		return "The deployment is not running."
	case StopAlreadyStopping:
		return "The deployment is already stopping."
	case StopIsProduction:
		return "Production deployments cannot be stopped."
	default:
		return ""
	}
}

// CheckStopTarget validates a deployment may be stopped: it must be running
// (ready with a running desired state) and not in production. Order matches the
// order every layer reports failures in.
func CheckStopTarget(in StopInput) StopFailureReason {
	switch {
	case in.Status != db.DeploymentsStatusReady:
		return StopNotRunning
	case in.DesiredState != db.DeploymentsDesiredStateRunning:
		return StopAlreadyStopping
	case in.EnvironmentSlug == envProduction:
		return StopIsProduction
	default:
		return StopOK
	}
}

// StartFailureReason is why a deployment cannot be started (woken). StartOK means it
// can.
type StartFailureReason int

const (
	StartOK StartFailureReason = iota
	StartNotStopped
	StartIsProduction
	StartSpendSuspended
)

// Message returns a caller-facing explanation.
func (r StartFailureReason) Message() string {
	switch r {
	case StartOK:
		return ""
	case StartNotStopped:
		return "The deployment is not stopped."
	case StartIsProduction:
		return "Production deployments cannot be started."
	case StartSpendSuspended:
		return "The workspace is suspended by its Compute spend cap. Raise the spend limit to resume."
	default:
		return ""
	}
}

// CheckStartTarget validates a deployment may be started (woken). "Stopped" is
// keyed on desired_state, not status: stopping sets desired_state=stopped
// immediately, while status only flips to stopped once krane has drained the
// last instance. Starting flips the intent back to running, so it is valid for
// any deployment whose intent is stopped (including one still draining), which
// is what the ctrl service and worker enforce. Starting resumes compute spend,
// so a workspace suspended by its spend cap is refused last.
func CheckStartTarget(in StartInput) StartFailureReason {
	switch {
	case in.DesiredState != db.DeploymentsDesiredStateStopped:
		return StartNotStopped
	case in.EnvironmentSlug == envProduction:
		return StartIsProduction
	case in.SpendSuspended:
		return StartSpendSuspended
	default:
		return StartOK
	}
}
