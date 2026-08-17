package deployment

import (
	"context"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

// FindDeployment loads a deployment by ID scoped to the caller's workspace. A
// cross-workspace match is masked as not found so a caller can't probe for
// deployments it can't see. The row carries the joined environment's slug and
// kind so lifecycle handlers can gate without a second query.
//
// It deliberately does no authorization: each handler authorizes inline so the
// exact permission checked stays visible at the call site.
func FindDeployment(ctx context.Context, database db.Database, workspaceID, deploymentID string) (db.FindDeploymentWithEnvironmentRow, error) {
	dep, err := db.Query.FindDeploymentWithEnvironment(ctx, database.RW(), deploymentID)
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
