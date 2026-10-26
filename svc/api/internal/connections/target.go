package connections

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
)

// LockTarget resolves the target selected by mode within app and returns the
// environment and deployment IDs to store for it. A pinned deployment must be
// readable, ready, and running, and stays share-locked until tx ends.
func LockTarget(ctx context.Context, tx db.DBTX, principal *principal.Principal, app db.App, mode db.ConnectionAppTargetsSelectionMode, environmentID, deploymentID *string) (sql.NullString, sql.NullString, error) {
	switch mode {
	case db.ConnectionAppTargetsSelectionModeAutomatic:
		return sql.NullString{}, sql.NullString{}, nil
	case db.ConnectionAppTargetsSelectionModeEnvironment:
		if environmentID == nil {
			return sql.NullString{}, sql.NullString{}, fault.New("target environment required", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Select a target environment."))
		}

		environment, err := db.Query.FindEnvironmentById(ctx, tx, *environmentID)
		if db.IsNotFound(err) || (err == nil && (environment.WorkspaceID != principal.AuthorizedWorkspaceID || environment.ProjectID != app.ProjectID || environment.AppID != app.ID)) {
			return sql.NullString{}, sql.NullString{}, fault.New("target environment not found", fault.Code(codes.Data.Environment.NotFound.URN()), fault.Public("The requested environment does not exist."))
		}
		if err != nil {
			return sql.NullString{}, sql.NullString{}, fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
		}

		query := rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(app.ProjectID).App(app.ID).Environment(environment.ID), permissions.Read)
		if err := principal.Authorize(query); err != nil {
			return sql.NullString{}, sql.NullString{}, apierrors.MaskInsufficientPermissionsAsNotFound(err, principal.Permissions, query, codes.Data.Environment.NotFound.URN(), "The requested environment does not exist.")
		}

		return sql.NullString{String: environment.ID, Valid: true}, sql.NullString{}, nil
	case db.ConnectionAppTargetsSelectionModeDeployment:
		if deploymentID == nil {
			return sql.NullString{}, sql.NullString{}, fault.New("target deployment required", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Select a target deployment."))
		}

		deployment, err := db.Query.LockConnectionTargetDeployment(ctx, tx, db.LockConnectionTargetDeploymentParams{
			ID:          *deploymentID,
			WorkspaceID: principal.AuthorizedWorkspaceID,
			ProjectID:   app.ProjectID,
			AppID:       app.ID,
		})
		if db.IsNotFound(err) {
			return sql.NullString{}, sql.NullString{}, fault.New("target deployment not found", fault.Code(codes.Data.Deployment.NotFound.URN()), fault.Public("The requested deployment does not exist."))
		}
		if err != nil {
			return sql.NullString{}, sql.NullString{}, fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
		}

		query := rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(app.ProjectID).App(app.ID).Environment(deployment.EnvironmentID).Deployment(deployment.ID), permissions.Read)
		if err := principal.Authorize(query); err != nil {
			return sql.NullString{}, sql.NullString{}, apierrors.MaskInsufficientPermissionsAsNotFound(err, principal.Permissions, query, codes.Data.Deployment.NotFound.URN(), "The requested deployment does not exist.")
		}

		if !deployment.FirstReadyAt.Valid || deployment.Status != mysqltype.DeploymentsStatusReady || deployment.DesiredState != mysqltype.DeploymentsDesiredStateRunning {
			return sql.NullString{}, sql.NullString{}, fault.New("target deployment not running", fault.Code(codes.App.Precondition.DeploymentNotRunning.URN()), fault.Public("The target deployment is not running. Select a ready deployment."))
		}

		return sql.NullString{}, sql.NullString{String: deployment.ID, Valid: true}, nil
	default:
		return sql.NullString{}, sql.NullString{}, fault.New("invalid target type", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Select automatic, environment, or deployment targeting."))
	}
}

// RequireTargetApp returns the app with appID in projectID. A missing app and
// one principal cannot read both return the same app-not-found error.
func RequireTargetApp(ctx context.Context, tx db.DBTX, principal *principal.Principal, projectID, appID string) (db.App, error) {
	app, err := db.Query.FindAppById(ctx, tx, appID)
	if db.IsNotFound(err) || (err == nil && (app.WorkspaceID != principal.AuthorizedWorkspaceID || app.ProjectID != projectID)) {
		return db.App{}, fault.New("target app not found", fault.Code(codes.Data.App.NotFound.URN()), fault.Public("The requested app does not exist."))
	}
	if err != nil {
		return db.App{}, fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}

	query := rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(projectID).App(app.ID), permissions.Read)
	if err := principal.Authorize(query); err != nil {
		return db.App{}, apierrors.MaskInsufficientPermissionsAsNotFound(err, principal.Permissions, query, codes.Data.App.NotFound.URN(), "The requested app does not exist.")
	}

	return app, nil
}
