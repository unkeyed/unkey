package connections

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
)

func RequireDeployment(ctx context.Context, database db.DBTX, principal *principal.Principal, id string) (db.FindDeploymentConnectionScopeRow, error) {
	row, err := db.Query.FindDeploymentConnectionScope(ctx, database, db.FindDeploymentConnectionScopeParams{
		ID:          id,
		WorkspaceID: principal.AuthorizedWorkspaceID,
	})
	if db.IsNotFound(err) {
		return db.FindDeploymentConnectionScopeRow{}, fault.New("deployment not found", fault.Code(codes.Data.Deployment.NotFound.URN()), fault.Public("The requested deployment does not exist."))
	}
	if err != nil {
		return db.FindDeploymentConnectionScopeRow{}, fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}

	if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(row.ProjectID).App(row.AppID).Environment(row.EnvironmentID).Deployment(row.ID), permissions.Read)); err != nil {
		return db.FindDeploymentConnectionScopeRow{}, apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Deployment.NotFound.URN(), "The requested deployment does not exist.")
	}

	return row, nil
}
