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

type Target struct {
	Type          db.ConnectionAppTargetsSelectionMode
	EnvironmentID *string
	DeploymentID  *string
}

// ValidateTarget must run in the transaction that writes the connection, so a
// pinned deployment cannot stop between validation and commit.
func ValidateTarget(ctx context.Context, tx db.DBTX, principal *principal.Principal, projectID, appID string, target Target) (db.InsertConnectionAppTargetParams, error) {
	selection := db.InsertConnectionAppTargetParams{
		ConnectionID:        "",
		SelectionMode:       target.Type,
		TargetEnvironmentID: sql.NullString{},
		TargetDeploymentID:  sql.NullString{},
	}
	switch target.Type {
	case db.ConnectionAppTargetsSelectionModeAutomatic:
		return selection, nil
	case db.ConnectionAppTargetsSelectionModeEnvironment:
		if target.EnvironmentID == nil || *target.EnvironmentID == "" {
			return selection, fault.New("target environment required", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Select a target environment."))
		}

		environment, err := db.Query.FindEnvironmentById(ctx, tx, *target.EnvironmentID)
		if db.IsNotFound(err) || (err == nil && (environment.WorkspaceID != principal.AuthorizedWorkspaceID || environment.ProjectID != projectID || environment.AppID != appID)) {
			return selection, fault.New("target environment not found", fault.Code(codes.Data.Environment.NotFound.URN()), fault.Public("The requested environment does not exist."))
		}
		if err != nil {
			return selection, fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
		}

		if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(projectID).App(appID).Environment(environment.ID), permissions.Read)); err != nil {
			return selection, apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Environment.NotFound.URN(), "The requested environment does not exist.")
		}

		selection.TargetEnvironmentID = sql.NullString{String: environment.ID, Valid: true}
	case db.ConnectionAppTargetsSelectionModeDeployment:
		if target.DeploymentID == nil || *target.DeploymentID == "" {
			return selection, fault.New("target deployment required", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Select a target deployment."))
		}

		deployment, err := db.Query.LockConnectionTargetDeployment(ctx, tx, db.LockConnectionTargetDeploymentParams{
			ID:          *target.DeploymentID,
			WorkspaceID: principal.AuthorizedWorkspaceID,
			ProjectID:   projectID,
			AppID:       appID,
		})
		if db.IsNotFound(err) {
			return selection, fault.New("target deployment not found", fault.Code(codes.Data.Deployment.NotFound.URN()), fault.Public("The requested deployment does not exist."))
		}
		if err != nil {
			return selection, fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
		}

		if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(projectID).App(appID).Environment(deployment.EnvironmentID).Deployment(deployment.ID), permissions.Read)); err != nil {
			return selection, apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Deployment.NotFound.URN(), "The requested deployment does not exist.")
		}

		if !deployment.FirstReadyAt.Valid || deployment.Status != mysqltype.DeploymentsStatusReady || deployment.DesiredState != mysqltype.DeploymentsDesiredStateRunning {
			return selection, fault.New("target deployment is not running", fault.Code(codes.Data.Connection.Conflict.URN()), fault.Public("The target deployment is no longer running. Select a ready deployment."))
		}

		selection.TargetDeploymentID = sql.NullString{String: deployment.ID, Valid: true}
	default:
		return selection, fault.New("invalid target type", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Select automatic, environment, or deployment targeting."))
	}

	return selection, nil
}

func RequireTargetApp(ctx context.Context, tx db.DBTX, principal *principal.Principal, projectID, appID string) (db.App, error) {
	app, err := db.Query.FindAppById(ctx, tx, appID)
	if db.IsNotFound(err) || (err == nil && (app.WorkspaceID != principal.AuthorizedWorkspaceID || app.ProjectID != projectID)) {
		return db.App{}, fault.New("target app not found", fault.Code(codes.Data.App.NotFound.URN()), fault.Public("The requested app does not exist."))
	}
	if err != nil {
		return db.App{}, fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}

	if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(projectID).App(app.ID), permissions.Read)); err != nil {
		return db.App{}, apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.App.NotFound.URN(), "The requested app does not exist.")
	}

	return app, nil
}
