package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// CancelDeploy cancels a workspace's Unkey Deploy plan. It stops the Stripe
// renewal (cancel_at_period_end for a Deploy-only subscription, or removes the
// plan-fee item from a mixed subscription, never refunding), clears the local
// deploy_plan entitlement, and tears down the workspace's running compute via
// DeployTeardownService.Teardown(ARCHIVE). Clearing deploy_plan makes both the
// invoice.created webhook and the month-end close skip the workspace, so the
// final invoice auto-finalizes via Stripe's normal path. Idempotent: succeeds
// if the workspace already has no Deploy plan.
//
// Trust boundary: authentication is the shared ctrl bearer only, and the
// caller-supplied workspace id is taken at face value. That is safe only
// because ctrl-api is internal (the dashboard authorizes the requesting user
// against the workspace before calling here, and the service is not exposed
// publicly). Do not route this from anything that has not already done that
// authorization.
func (s *Service) CancelDeploy(ctx context.Context, req *connect.Request[ctrlv1.CancelDeployRequest]) (*connect.Response[ctrlv1.CancelDeployResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	workspaceID := req.Msg.GetWorkspaceId()
	if workspaceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("workspace_id is required"))
	}

	ws, err := db.Query.FindWorkspaceByID(ctx, s.db.RO(), workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("workspace not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load workspace: %w", err))
	}

	// Idempotency: no Deploy plan means there is nothing to cancel. Return
	// success so retries and double-clicks are safe.
	if !ws.DeployPlan.Valid || ws.DeployPlan.String == "" {
		logger.Info("deploy already cancelled", "workspace_id", workspaceID)
		return connect.NewResponse(&ctrlv1.CancelDeployResponse{}), nil
	}

	if s.deploysub == nil || !s.deploysub.Configured() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deploy billing not configured"))
	}

	// Stop the Stripe renewal. A workspace with a Deploy plan but no
	// subscription id (e.g. a comped override) skips Stripe and proceeds to
	// clear the entitlement and tear down compute.
	if ws.StripeSubscriptionID.Valid && ws.StripeSubscriptionID.String != "" {
		topology, cancelErr := s.deploysub.Cancel(ctx, ws.StripeSubscriptionID.String)
		if cancelErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to cancel stripe subscription: %w", cancelErr))
		}
		logger.Info("cancelled deploy stripe subscription",
			"workspace_id", workspaceID,
			"topology", topology.String(),
		)
	}

	// Drain the workspace's running compute before clearing the entitlement.
	// Order matters for crash-safety: the idempotency short-circuit above keys
	// on deploy_plan, so the teardown must be dispatched while deploy_plan is
	// still set. If we cleared first and crashed before dispatching, a retry
	// would short-circuit and never tear down. Send is fire-and-forget and
	// idempotent (keyed), so a retry that re-dispatches is harmless.
	_, err = hydrav1.NewDeployTeardownServiceIngressClient(s.restate, workspaceID).
		Teardown().
		Send(ctx, &hydrav1.TeardownRequest{
			Mode: hydrav1.TeardownMode_TEARDOWN_MODE_ARCHIVE,
		}, restate.WithIdempotencyKey("deploy-teardown-archive-"+workspaceID))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to dispatch teardown: %w", err))
	}

	// Clear the local entitlement so the dashboard gate blocks new deploys and
	// both the invoice.created webhook and month-end close skip the workspace.
	// No audit log here: the dashboard records the user actor, mirroring the
	// deployment cancel path.
	if err := db.Query.ClearWorkspaceDeployPlan(ctx, s.db.RW(), db.ClearWorkspaceDeployPlanParams{
		ID:        workspaceID,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to clear deploy plan: %w", err))
	}

	logger.Info("deploy cancelled", "workspace_id", workspaceID)
	return connect.NewResponse(&ctrlv1.CancelDeployResponse{}), nil
}
