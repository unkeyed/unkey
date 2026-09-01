package deployment

import (
	"context"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
)

// EnsureWorkspaceCanDeploy refuses an action that creates or activates compute
// for a workspace with no Compute plan or one stopped by its spend cap. The
// faults it returns already carry precondition codes, so a handler returns them
// as they are.
//
// Callers whose own state gate has to be reported first, such as the start
// route, use [LoadBilling] and order the checks themselves.
func EnsureWorkspaceCanDeploy(ctx context.Context, database db.Database, workspaceID string) error {
	billing, err := LoadBilling(ctx, database, workspaceID)
	if err != nil {
		return err
	}

	if err := deploygate.CheckWorkspacePlan(billing.Plan, billing.PlanOverride); err != nil {
		return err
	}
	return deploygate.CheckWorkspaceSpend(billing.SpendSuspended)
}

// LoadBilling reads a workspace's Compute billing state from the PRIMARY: a
// workspace suspended moments ago must not slip an action through on a stale
// replica. A workspace with no billing row reads as the zero value, which is
// the normal state before anyone subscribes and means "no plan, not
// suspended".
func LoadBilling(ctx context.Context, database db.Database, workspaceID string) (db.FindWorkspaceBillingByWorkspaceIDRow, error) {
	billing, err := db.Query.FindWorkspaceBillingByWorkspaceID(ctx, database.RW(), workspaceID)
	if err != nil && !db.IsNotFound(err) {
		return db.FindWorkspaceBillingByWorkspaceIDRow{}, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error loading workspace billing"),
			fault.Public("Failed to retrieve workspace billing state."),
		)
	}
	return billing, nil
}

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
