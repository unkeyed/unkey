// Package deploygate holds the deployment lifecycle preconditions shared by the
// three layers that each enforce them: the public API handlers, the ctrl RPC
// services, and the ctrl Restate workflows. Centralizing the invariant here is
// what stops the three copies from drifting — a promote that is legal at the API
// is legal at the worker, by construction.
//
// The Check* functions return nil when the action is allowed, or a fault
// carrying the precondition code and a caller-facing message. Surface that
// message with fault.UserFacingMessage(err) — never err.Error(), which also
// includes internal detail. The API returns the fault directly (its error
// middleware renders UserFacingMessage); ctrl wraps UserFacingMessage into its
// connect or restate error.
package deploygate

import (
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
)

// envProduction is the environment whose deployment serves production traffic:
// promote/rollback require it, stop/start reject it.
const envProduction = "production"

// Each check takes its own input holding exactly the fields it needs. Status and
// DesiredState are typed to the shared pkg/mysql/types enums that the generated db
// row fields already use, so callers pass their db row fields directly without casting.

// PromoteInput is the state CheckPromoteTarget needs.
type PromoteInput struct {
	Status              dbtype.DeploymentsStatus
	DesiredState        dbtype.DeploymentsDesiredState
	EnvironmentSlug     string
	CurrentDeploymentID string
	DeploymentID        string
	IsRolledBack        bool
}

// RollbackInput is the state CheckRollbackTarget needs.
type RollbackInput struct {
	Status              dbtype.DeploymentsStatus
	DesiredState        dbtype.DeploymentsDesiredState
	EnvironmentSlug     string
	CurrentDeploymentID string
	DeploymentID        string
}

// StopInput is the state CheckStopTarget needs.
type StopInput struct {
	Status          dbtype.DeploymentsStatus
	DesiredState    dbtype.DeploymentsDesiredState
	EnvironmentSlug string
}

// StartInput is the state CheckStartTarget needs.
type StartInput struct {
	DesiredState    dbtype.DeploymentsDesiredState
	EnvironmentSlug string
	SpendSuspended  bool
}

// isCurrent reports whether the target is the app's current (live) deployment.
func isCurrent(currentDeploymentID, deploymentID string) bool {
	return currentDeploymentID != "" && currentDeploymentID == deploymentID
}

// The *FailureReason enums are the internal discriminator each Check* builds its
// fault from; they are also the source of the caller-facing message strings.

// TargetFailureReason is why a deployment cannot become the app's current
// deployment (promote/rollback). TargetOK means it is eligible.
type TargetFailureReason int

const (
	TargetOK TargetFailureReason = iota
	TargetNotReady
	TargetIsDraining
	TargetNotProduction
	TargetNoCurrentDeployment
	TargetIsCurrent
)

// Message returns a caller-facing explanation. It is deliberately generic across
// promote and rollback so both surface identical wording.
func (r TargetFailureReason) Message() string {
	switch r {
	case TargetOK:
		return ""
	case TargetNotReady:
		return "The deployment is not ready."
	case TargetIsDraining:
		return "The deployment is shutting down and cannot serve traffic."
	case TargetNotProduction:
		return "Only production deployments can be promoted or rolled back."
	case TargetNoCurrentDeployment:
		return "The app has no current deployment."
	case TargetIsCurrent:
		return "The deployment is already the current deployment."
	default:
		return ""
	}
}

// targetFault maps a target failure reason onto a fault with its precondition
// code, or nil for TargetOK.
func targetFault(r TargetFailureReason) error {
	var code codes.Code
	switch r {
	case TargetNotReady, TargetIsDraining:
		code = codes.App.Precondition.DeploymentNotReady
	case TargetNotProduction:
		code = codes.App.Precondition.DeploymentNotProduction
	case TargetNoCurrentDeployment:
		code = codes.App.Precondition.DeploymentNoCurrent
	case TargetIsCurrent:
		code = codes.App.Precondition.DeploymentIsCurrent
	case TargetOK:
		return nil
	default:
		code = codes.App.Precondition.PreconditionFailed
	}
	return fault.New(
		"target precondition failed",
		fault.Code(code.URN()),
		fault.Internal("deploygate rejected promote/rollback: "+r.Message()),
		fault.Public(r.Message()),
	)
}

// targetCore holds the preconditions common to promote and rollback: the
// deployment must be ready, running (not draining), in production, and its app
// must already have a current deployment. Order matters — it is the order every
// layer reports failures in.
func targetCore(
	status dbtype.DeploymentsStatus,
	desiredState dbtype.DeploymentsDesiredState,
	environmentSlug, currentDeploymentID string,
) TargetFailureReason {
	switch {
	case status != dbtype.DeploymentsStatusReady:
		return TargetNotReady
	case desiredState != dbtype.DeploymentsDesiredStateRunning:
		return TargetIsDraining
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
func CheckPromoteTarget(in PromoteInput) error {
	if r := targetCore(in.Status, in.DesiredState, in.EnvironmentSlug, in.CurrentDeploymentID); r != TargetOK {
		return targetFault(r)
	}
	if isCurrent(in.CurrentDeploymentID, in.DeploymentID) && !in.IsRolledBack {
		return targetFault(TargetIsCurrent)
	}
	return nil
}

// CheckRollbackTarget validates a deployment may be rolled back to. Rolling back
// to the deployment that is already current is always a no-op, so it is rejected.
func CheckRollbackTarget(in RollbackInput) error {
	if r := targetCore(in.Status, in.DesiredState, in.EnvironmentSlug, in.CurrentDeploymentID); r != TargetOK {
		return targetFault(r)
	}
	if isCurrent(in.CurrentDeploymentID, in.DeploymentID) {
		return targetFault(TargetIsCurrent)
	}
	return nil
}

// StopFailureReason is why a deployment cannot be stopped. StopOK means it can.
type StopFailureReason int

const (
	StopOK StopFailureReason = iota
	StopNotRunning
	StopIsStopping
	StopIsProduction
)

// Message returns a caller-facing explanation.
func (r StopFailureReason) Message() string {
	switch r {
	case StopOK:
		return ""
	case StopNotRunning:
		return "The deployment is not running."
	case StopIsStopping:
		return "The deployment is already stopping."
	case StopIsProduction:
		return "Production deployments cannot be stopped."
	default:
		return ""
	}
}

// stopFault maps a stop failure reason onto a fault with its precondition code,
// or nil for StopOK.
func stopFault(r StopFailureReason) error {
	var code codes.Code
	switch r {
	case StopNotRunning:
		code = codes.App.Precondition.DeploymentNotRunning
	case StopIsStopping:
		code = codes.App.Precondition.DeploymentIsStopping
	case StopIsProduction:
		code = codes.App.Precondition.DeploymentIsProduction
	case StopOK:
		return nil
	default:
		code = codes.App.Precondition.PreconditionFailed
	}
	return fault.New(
		"stop precondition failed",
		fault.Code(code.URN()),
		fault.Internal("deploygate rejected stop: "+r.Message()),
		fault.Public(r.Message()),
	)
}

// CheckStopTarget validates a deployment may be stopped: it must be running
// (ready with a running desired state) and not in production. Order matches the
// order every layer reports failures in.
func CheckStopTarget(in StopInput) error {
	switch {
	case in.Status != dbtype.DeploymentsStatusReady:
		return stopFault(StopNotRunning)
	case in.DesiredState != dbtype.DeploymentsDesiredStateRunning:
		return stopFault(StopIsStopping)
	case in.EnvironmentSlug == envProduction:
		return stopFault(StopIsProduction)
	default:
		return nil
	}
}

// StartFailureReason is why a deployment cannot be started (woken). StartOK means
// it can.
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

// startFault maps a start failure reason onto a fault with its precondition
// code, or nil for StartOK. SpendSuspended is a billing state, not a deployment
// lifecycle one, so it carries the generic precondition code.
func startFault(r StartFailureReason) error {
	var code codes.Code
	switch r {
	case StartNotStopped:
		code = codes.App.Precondition.DeploymentNotStopped
	case StartIsProduction:
		code = codes.App.Precondition.DeploymentIsProduction
	case StartSpendSuspended:
		code = codes.App.Precondition.PreconditionFailed
	case StartOK:
		return nil
	default:
		code = codes.App.Precondition.PreconditionFailed
	}
	return fault.New(
		"start precondition failed",
		fault.Code(code.URN()),
		fault.Internal("deploygate rejected start: "+r.Message()),
		fault.Public(r.Message()),
	)
}

// CheckStartTarget validates a deployment may be started (woken). "Stopped" is
// keyed on desired_state, not status: stopping sets desired_state=stopped
// immediately, while status only flips to stopped once krane has drained the
// last instance. Starting flips the intent back to running, so it is valid for
// any deployment whose intent is stopped (including one still draining), which
// is what the ctrl service and worker enforce. Starting resumes compute spend,
// so a workspace suspended by its spend cap is refused last.
func CheckStartTarget(in StartInput) error {
	switch {
	case in.DesiredState != dbtype.DeploymentsDesiredStateStopped:
		return startFault(StartNotStopped)
	case in.EnvironmentSlug == envProduction:
		return startFault(StartIsProduction)
	case in.SpendSuspended:
		return startFault(StartSpendSuspended)
	default:
		return nil
	}
}
