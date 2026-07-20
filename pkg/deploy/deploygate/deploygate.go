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

// Input is a deployment's lifecycle-relevant state, typed to the pkg/db enums:
// the API passes its db row fields directly, ctrl converts from its own db
// package at the boundary. Fields unused by a check (the current-pointer fields
// for stop/start) may be left zero.
type Input struct {
	Status               db.DeploymentsStatus
	DesiredState         db.DeploymentsDesiredState
	EnvironmentSlug      string
	HasCurrentDeployment bool
	CurrentDeploymentID  string
	DeploymentID         string
	IsRolledBack         bool
	SpendSuspended       bool
}

// isCurrent reports whether this deployment is the app's current (live) one.
func (in Input) isCurrent() bool {
	return in.HasCurrentDeployment && in.CurrentDeploymentID == in.DeploymentID
}

// PromotionReason is why a deployment fails a promote/rollback precondition.
// PromotionOK means it is eligible to become the app's current deployment.
type PromotionReason int

const (
	PromotionOK PromotionReason = iota
	PromotionNotReady
	PromotionDraining
	PromotionNotProduction
	PromotionNoCurrentDeployment
	PromotionAlreadyCurrent
)

// Message returns a caller-facing explanation. It is deliberately generic across
// promote and rollback so both surface identical wording.
func (r PromotionReason) Message() string {
	switch r {
	case PromotionOK:
		return ""
	case PromotionNotReady:
		return "The deployment is not ready."
	case PromotionDraining:
		return "The deployment is shutting down and cannot serve traffic."
	case PromotionNotProduction:
		return "Only production deployments can be promoted or rolled back."
	case PromotionNoCurrentDeployment:
		return "The app has no current deployment."
	case PromotionAlreadyCurrent:
		return "The deployment is already the current deployment."
	default:
		return ""
	}
}

// promotionCore holds the preconditions common to promote and rollback: the
// deployment must be ready, running (not draining), in production, and its app
// must already have a current deployment. Order matters — it is the order every
// layer reports failures in.
func promotionCore(in Input) PromotionReason {
	switch {
	case in.Status != db.DeploymentsStatusReady:
		return PromotionNotReady
	case in.DesiredState != db.DeploymentsDesiredStateRunning:
		return PromotionDraining
	case in.EnvironmentSlug != envProduction:
		return PromotionNotProduction
	case !in.HasCurrentDeployment:
		return PromotionNoCurrentDeployment
	default:
		return PromotionOK
	}
}

// CheckPromoteTarget validates a deployment may be promoted to current.
// Promoting the deployment that is already current is rejected, unless the app
// is in a rolled-back state (then it is a rollback confirmation, allowed).
func CheckPromoteTarget(in Input) PromotionReason {
	if r := promotionCore(in); r != PromotionOK {
		return r
	}
	if in.isCurrent() && !in.IsRolledBack {
		return PromotionAlreadyCurrent
	}
	return PromotionOK
}

// CheckRollbackTarget validates a deployment may be rolled back to. Rolling back
// to the deployment that is already current is always a no-op, so it is rejected.
func CheckRollbackTarget(in Input) PromotionReason {
	if r := promotionCore(in); r != PromotionOK {
		return r
	}
	if in.isCurrent() {
		return PromotionAlreadyCurrent
	}
	return PromotionOK
}

// StopReason is why a deployment cannot be stopped. StopOK means it can.
type StopReason int

const (
	StopOK StopReason = iota
	StopNotRunning
	StopAlreadyStopping
	StopIsProduction
)

// Message returns a caller-facing explanation.
func (r StopReason) Message() string {
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

// CheckStoppable validates a deployment may be stopped: it must be running
// (ready with a running desired state) and not in production. Order matches the
// order every layer reports failures in.
func CheckStoppable(in Input) StopReason {
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

// StartReason is why a deployment cannot be started (woken). StartOK means it
// can.
type StartReason int

const (
	StartOK StartReason = iota
	StartNotStopped
	StartIsProduction
	StartSpendSuspended
)

// Message returns a caller-facing explanation.
func (r StartReason) Message() string {
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

// CheckStartable validates a deployment may be started (woken). "Stopped" is
// keyed on desired_state, not status: stopping sets desired_state=stopped
// immediately, while status only flips to stopped once krane has drained the
// last instance. Starting flips the intent back to running, so it is valid for
// any deployment whose intent is stopped (including one still draining), which
// is what the ctrl service and worker enforce. Starting resumes compute spend,
// so a workspace suspended by its spend cap is refused last.
func CheckStartable(in Input) StartReason {
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
