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

func RequireEnvironment(ctx context.Context, tx db.DBTX, principal *principal.Principal, project, app, environment string) (db.Environment, error) {
	row, err := db.Query.FindEnvironmentByIdentifiers(ctx, tx, db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		Project:     project,
		App:         app,
		Environment: environment,
	})
	if db.IsNotFound(err) {
		return db.Environment{}, fault.New("environment not found", fault.Code(codes.Data.Environment.NotFound.URN()), fault.Public("The requested environment does not exist."))
	}
	if err != nil {
		return db.Environment{}, fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}

	if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(row.ProjectID).App(row.AppID).Environment(row.ID), permissions.Read)); err != nil {
		return db.Environment{}, apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Environment.NotFound.URN(), "The requested environment does not exist.")
	}

	return row, nil
}
