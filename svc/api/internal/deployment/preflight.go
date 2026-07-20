package deployment

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
)

// TargetFault maps a deploygate promote/rollback rejection onto the API
// fault with the matching precondition code, so both handlers surface the same
// code and message for the same reason.
func TargetFault(r deploygate.TargetFailureReason) error {
	var code codes.Code
	switch r {
	case deploygate.TargetNotReady, deploygate.TargetDraining:
		code = codes.App.Precondition.DeploymentNotReady
	case deploygate.TargetNotProduction:
		code = codes.App.Precondition.DeploymentNotProduction
	case deploygate.TargetNoCurrentDeployment:
		code = codes.App.Precondition.DeploymentNoCurrent
	case deploygate.TargetAlreadyCurrent:
		code = codes.App.Precondition.DeploymentIsCurrent
	case deploygate.TargetOK:
		return nil
	default:
		code = codes.App.Precondition.PreconditionFailed
	}
	return fault.New(
		"deployment lifecycle precondition failed",
		fault.Code(code.URN()),
		fault.Internal("deploygate rejected promote/rollback: "+r.Message()),
		fault.Public(r.Message()),
	)
}

// StopFault maps a deploygate stop rejection onto the API fault with the
// matching precondition code.
func StopFault(r deploygate.StopFailureReason) error {
	var code codes.Code
	switch r {
	case deploygate.StopNotRunning:
		code = codes.App.Precondition.DeploymentNotRunning
	case deploygate.StopAlreadyStopping:
		code = codes.App.Precondition.DeploymentIsStopping
	case deploygate.StopIsProduction:
		code = codes.App.Precondition.DeploymentIsProduction
	case deploygate.StopOK:
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

// StartFault maps a deploygate start rejection onto the API fault with the
// matching precondition code.
func StartFault(r deploygate.StartFailureReason) error {
	var code codes.Code
	switch r {
	case deploygate.StartNotStopped:
		code = codes.App.Precondition.DeploymentNotStopped
	case deploygate.StartIsProduction:
		code = codes.App.Precondition.DeploymentIsProduction
	case deploygate.StartSpendSuspended:
		// Billing state, not deployment lifecycle; no dedicated code exists, so
		// the generic precondition code carries the specific message.
		code = codes.App.Precondition.PreconditionFailed
	case deploygate.StartOK:
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

// FindDeployment loads a deployment by ID scoped to the caller's workspace. A
// cross-workspace match is masked as not found so a caller can't probe for
// deployments it can't see. The row carries EnvironmentSlug (joined) so
// lifecycle handlers can gate on the environment without a second query.
//
// It deliberately does no authorization: each handler authorizes inline so the
// exact permission checked stays visible at the call site.
func FindDeployment(ctx context.Context, database db.Database, workspaceID, deploymentID string) (db.FindDeploymentWithEnvironmentRow, error) {
	dep, err := db.Query.FindDeploymentWithEnvironment(ctx, database.RO(), deploymentID)
	if err != nil && !db.IsNotFound(err) {
		return db.FindDeploymentWithEnvironmentRow{}, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployment."),
		)
	}

	if db.IsNotFound(err) || dep.WorkspaceID != workspaceID {
		return db.FindDeploymentWithEnvironmentRow{}, fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("deployment not found or belongs to another workspace"),
			fault.Public("The requested deployment does not exist."),
		)
	}

	return dep, nil
}

// MapCtrlError converts a ctrl connect error from a lifecycle RPC into an API
// fault. Ctrl re-runs the same deploygate checks the handler already passed, so
// a precondition message matching a known gate rejection (a race: the state
// changed between the two checks) is mapped back to its reason and surfaces the
// identical code and message the local gate would have produced. Unrecognized
// precondition failures (e.g. a spend-cap suspension) become a generic 412 with
// preconditionMsg, not-found stays a 404, and everything else falls through to
// the generic ctrl mapping.
func MapCtrlError(err error, action string, preconditionMsg string) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		//nolint:exhaustive // all other Connect error codes fall through to the generic mapping
		switch connectErr.Code() {
		case connect.CodeFailedPrecondition:
			msg := connectErr.Message()
			for _, r := range []deploygate.TargetFailureReason{
				deploygate.TargetNotReady,
				deploygate.TargetDraining,
				deploygate.TargetNotProduction,
				deploygate.TargetNoCurrentDeployment,
				deploygate.TargetAlreadyCurrent,
			} {
				if msg == r.Message() {
					return TargetFault(r)
				}
			}
			for _, r := range []deploygate.StopFailureReason{
				deploygate.StopNotRunning,
				deploygate.StopAlreadyStopping,
				deploygate.StopIsProduction,
			} {
				if msg == r.Message() {
					return StopFault(r)
				}
			}
			for _, r := range []deploygate.StartFailureReason{
				deploygate.StartNotStopped,
				deploygate.StartIsProduction,
				deploygate.StartSpendSuspended,
			} {
				if msg == r.Message() {
					return StartFault(r)
				}
			}
			return fault.Wrap(
				err,
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("ctrl reported a precondition failure: "+msg),
				fault.Public(preconditionMsg),
			)
		case connect.CodeNotFound:
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Deployment.NotFound.URN()),
				fault.Internal("ctrl reported not found: "+connectErr.Message()),
				fault.Public("The requested deployment does not exist."),
			)
		default:
		}
	}
	return ctrlclient.HandleError(err, action)
}
