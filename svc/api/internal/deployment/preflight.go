package deployment

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
)

// FindAuthorized loads a deployment for a lifecycle action. Cross-workspace
// matches AND authorization failures are both masked as not found, so a caller
// can never learn whether a deployment exists without holding the
// environment-scoped action (wildcard or exact) for it.
func FindAuthorized(ctx context.Context, database db.Database, p *principal.Principal, deploymentID string, action rbac.ActionType) (db.Deployment, error) {
	dep, err := db.Query.FindDeploymentById(ctx, database.RO(), deploymentID)
	if err != nil && !db.IsNotFound(err) {
		return db.Deployment{}, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployment."),
		)
	}

	if db.IsNotFound(err) || dep.WorkspaceID != p.WorkspaceID {
		return db.Deployment{}, fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("deployment not found or belongs to another workspace"),
			fault.Public("The requested deployment does not exist."),
		)
	}

	err = p.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       action,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   dep.EnvironmentID,
			Action:       action,
		}),
	))
	if err != nil {
		return db.Deployment{}, fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking deployment existence"),
			fault.Public("The requested deployment does not exist."),
		)
	}

	return dep, nil
}

// RequireProduction guards promote and rollback: both swap
// apps.current_deployment_id, which tracks the production live deployment, so
// running them outside production would corrupt that pointer. The dashboard
// enforces the same rule in its action eligibility.
func RequireProduction(ctx context.Context, database db.Database, environmentID string, publicMsg string) error {
	slug, err := environmentSlug(ctx, database, environmentID)
	if err != nil {
		return err
	}
	if slug != "production" {
		return fault.New(
			"not a production deployment",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("environment slug is not production: "+slug),
			fault.Public(publicMsg),
		)
	}
	return nil
}

// RequireNonProduction guards stop and start: production deployments are never
// stopped, so neither action applies to them. The dashboard enforces the same
// rule in its action eligibility.
func RequireNonProduction(ctx context.Context, database db.Database, environmentID string, publicMsg string) error {
	slug, err := environmentSlug(ctx, database, environmentID)
	if err != nil {
		return err
	}
	if slug == "production" {
		return fault.New(
			"production deployment",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("action is not allowed on production environments"),
			fault.Public(publicMsg),
		)
	}
	return nil
}

func environmentSlug(ctx context.Context, database db.Database, environmentID string) (string, error) {
	environment, err := db.Query.FindEnvironmentById(ctx, database.RO(), environmentID)
	if err != nil {
		return "", fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load environment for eligibility check"),
			fault.Public("Failed to resolve the deployment's environment."),
		)
	}
	return environment.Slug, nil
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
