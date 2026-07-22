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
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// DeprovisionCompute stops a workspace's Compute (via DeployTeardownService)
// and clears its deploy_plan, which makes the invoice.created webhook and
// month-end close skip it so Stripe finalizes the last invoice normally.
// Idempotent. Stopping the Stripe renewal is the dashboard's job and runs
// before this; ctrl owns only the teardown and the local writes.
//
// Auth is the shared ctrl bearer and the workspace id is trusted at face value,
// safe only because ctrl-api is internal and the dashboard authorizes the user
// against the workspace first.
func (s *Service) DeprovisionCompute(ctx context.Context, req *connect.Request[ctrlv1.DeprovisionComputeRequest]) (*connect.Response[ctrlv1.DeprovisionComputeResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	workspaceID := req.Msg.GetWorkspaceId()
	if workspaceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("workspace_id is required"))
	}

	if _, err := s.db.FindWorkspaceByID(ctx, workspaceID); err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("workspace not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load workspace: %w", err))
	}

	// Nothing to cancel without a billing row or plan; return success so retries
	// are safe.
	billing, err := s.db.FindWorkspaceBillingByWorkspaceID(ctx, workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			logger.Info("compute already deprovisioned", "workspace_id", workspaceID)
			return connect.NewResponse(&ctrlv1.DeprovisionComputeResponse{}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load workspace billing: %w", err))
	}
	if !billing.Plan.Valid || billing.Plan.String == "" {
		logger.Info("compute already deprovisioned", "workspace_id", workspaceID)
		return connect.NewResponse(&ctrlv1.DeprovisionComputeResponse{}), nil
	}

	// Tear down before clearing deploy_plan: the idempotency guard above keys on
	// deploy_plan, so clearing first then crashing would skip teardown on retry.
	// The Send is keyed and idempotent, so a re-dispatch is harmless.
	//
	// The key carries the subscription id and updated_at, not just the workspace
	// id: a constant key let a cancel/resubscribe/deploy/cancel cycle dedupe the
	// second teardown against the first inside Restate's retention window,
	// leaving resubscribed compute running unbilled. A retry of the same call
	// (row unchanged) still dedupes.
	idempotencyKey := fmt.Sprintf("deploy-teardown-archive-%s-%s-%d",
		workspaceID, billing.StripeSubscriptionID.String, billing.UpdatedAtM.Int64)
	_, err = hydrav1.NewDeployTeardownServiceIngressClient(s.restate, workspaceID).
		Teardown().
		Send(ctx, &hydrav1.TeardownRequest{
			Mode: hydrav1.TeardownMode_TEARDOWN_MODE_ARCHIVE,
		}, restate.WithIdempotencyKey(idempotencyKey))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to dispatch teardown: %w", err))
	}

	// End enforcement: a plan-less workspace must not stay spend-suspended, or a
	// later resubscribe starts blocked. Runs before the deploy_plan clear because
	// the idempotency guard keys on deploy_plan; a crash after the clear would
	// strand the suspension with no retry path.
	//
	// Accepted race: a spend-check child dispatched from a pre-cancel snapshot
	// can re-suspend after this clear. A complete fix needs deploy_plan plumbed
	// into CheckWorkspaceSpendRequest (proto change, deferred); the window is a
	// single tick and support can clear the flag.
	if err := s.db.SetWorkspaceDeploySpendSuspended(ctx, db.SetWorkspaceDeploySpendSuspendedParams{
		Suspended: false,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:        workspaceID,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to clear spend-suspended: %w", err))
	}

	// Clear the entitlement last (it is the idempotency key, so it flips only
	// after every other step succeeded): blocks new deploys, drops billing. No
	// audit log; the dashboard records the user actor.
	if err := s.db.ClearWorkspaceDeployPlan(ctx, db.ClearWorkspaceDeployPlanParams{
		ID:        workspaceID,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to clear deploy plan: %w", err))
	}

	logger.Info("compute deprovisioned", "workspace_id", workspaceID)
	return connect.NewResponse(&ctrlv1.DeprovisionComputeResponse{}), nil
}
