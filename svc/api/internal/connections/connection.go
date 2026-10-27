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

func LockConnection(ctx context.Context, tx db.DBTX, principal *principal.Principal, id string) (db.LockAppConnectionRow, error) {
	connection, err := db.Query.LockAppConnection(ctx, tx, db.LockAppConnectionParams{
		ID:          id,
		WorkspaceID: principal.AuthorizedWorkspaceID,
	})
	if db.IsNotFound(err) {
		return db.LockAppConnectionRow{}, fault.New("connection not found", fault.Code(codes.Data.Connection.NotFound.URN()), fault.Public("The requested connection does not exist."))
	}
	if err != nil {
		return db.LockAppConnectionRow{}, fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}

	if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(connection.ProjectID).App(connection.AppID).Environment(connection.EnvironmentID), permissions.Read)); err != nil {
		return db.LockAppConnectionRow{}, apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Connection.NotFound.URN(), "The requested connection does not exist.")
	}

	return connection, nil
}
