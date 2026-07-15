package deployment

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
)

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
// fault: precondition failures become a 412 with preconditionMsg, not-found
// stays a 404, and everything else falls through to the generic ctrl mapping.
func MapCtrlError(err error, action string, preconditionMsg string) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		//nolint:exhaustive // all other Connect error codes fall through to the generic mapping
		switch connectErr.Code() {
		case connect.CodeFailedPrecondition:
			return fault.Wrap(
				err,
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("ctrl reported a precondition failure: "+connectErr.Message()),
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
