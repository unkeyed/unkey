package sentinel

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/sentinelpolicy"
)

// ChangeReplicas updates the desired replica count of a running sentinel.
// Mirrors the transactional shape of ChangeTier: all subscription and
// state writes happen here in one tx, and SentinelService.Deploy is used
// purely as the "wait for krane convergence" primitive via an empty
// request.
//
// Inside the tx:
//  1. Close the currently open sentinel_subscriptions row.
//  2. Insert a new subscription row carrying the same tier forward with
//     the new replica count.
//  3. Repoint sentinels.subscription_id at the new row.
//  4. Update sentinels.desired_replicas + deploy_status=progressing so
//     krane sees the new intent.
//  5. Drop a deployment_changes outbox row so krane picks up the spec
//     change on its incremental watch.
//
// After the tx commits, Deploy is enqueued with an empty request. It
// reads the current state (already updated here), sees no image
// change, and awaits NotifyReady until krane reports the scaled
// deployment as healthy.
//
// No-op (no tx, no Deploy) if the requested count equals the current
// desired_replicas.
func (s *Service) ChangeReplicas(
	ctx context.Context,
	req *connect.Request[ctrlv1.ChangeReplicasRequest],
) (*connect.Response[ctrlv1.ChangeReplicasResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	if err := assert.All(
		assert.NotEmpty(req.Msg.GetSentinelId(), "sentinel_id is required"),
		assert.Greater(req.Msg.GetDesiredReplicas(), int32(0), "desired_replicas must be > 0"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	joined, err := db.Query.FindSentinelByID(ctx, s.db.RO(), req.Msg.GetSentinelId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	sentinel, currentSub := joined.Sentinel, joined.SentinelSubscription
	newReplicas := req.Msg.GetDesiredReplicas()

	// No-op: already at this replica count.
	if sentinel.DesiredReplicas == newReplicas {
		return connect.NewResponse(&ctrlv1.ChangeReplicasResponse{}), nil
	}

	env, err := db.Query.FindEnvironmentById(ctx, s.db.RO(), sentinel.EnvironmentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("find environment: %w", err))
	}

	minReplicas := sentinelpolicy.MinReplicasForEnv(env.Slug)
	if newReplicas < minReplicas {
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf("%s sentinels require at least %d replicas", env.Slug, minReplicas),
		)
	}

	newSubscriptionID := uid.New(uid.SentinelSubscriptionPrefix)
	now := time.Now().UnixMilli()

	err = db.Tx(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if err := db.Query.TerminateOpenSentinelSubscription(txCtx, tx, db.TerminateOpenSentinelSubscriptionParams{
			SentinelID:   sentinel.ID,
			TerminatedAt: sql.NullInt64{Valid: true, Int64: now},
		}); err != nil {
			return fmt.Errorf("terminate open sentinel subscription: %w", err)
		}

		if err := db.Query.InsertSentinelSubscription(txCtx, tx, db.InsertSentinelSubscriptionParams{
			ID:             newSubscriptionID,
			SentinelID:     sentinel.ID,
			WorkspaceID:    sentinel.WorkspaceID,
			RegionID:       sentinel.RegionID,
			TierID:         currentSub.TierID,
			TierVersion:    currentSub.TierVersion,
			CpuMillicores:  currentSub.CpuMillicores,
			MemoryMib:      currentSub.MemoryMib,
			Replicas:       newReplicas,
			PricePerSecond: currentSub.PricePerSecond,
			CreatedAt:      now,
		}); err != nil {
			return fmt.Errorf("insert sentinel subscription: %w", err)
		}

		if err := db.Query.UpdateSentinelSubscription(txCtx, tx, db.UpdateSentinelSubscriptionParams{
			ID:             sentinel.ID,
			SubscriptionID: newSubscriptionID,
			UpdatedAt:      sql.NullInt64{Valid: true, Int64: now},
		}); err != nil {
			return fmt.Errorf("repoint sentinel subscription: %w", err)
		}

		if err := db.Query.UpdateSentinelConfig(txCtx, tx, db.UpdateSentinelConfigParams{
			ID:              sentinel.ID,
			Image:           sentinel.Image,
			DesiredReplicas: newReplicas,
			DeployStatus:    db.SentinelsDeployStatusProgressing,
			UpdatedAt:       sql.NullInt64{Valid: true, Int64: now},
		}); err != nil {
			return fmt.Errorf("update sentinel config: %w", err)
		}

		return db.Query.InsertDeploymentChange(txCtx, tx, db.InsertDeploymentChangeParams{
			ResourceType: db.DeploymentChangesResourceTypeSentinel,
			ResourceID:   sentinel.ID,
			RegionID:     sentinel.RegionID,
			CreatedAt:    now,
		})
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Deploy is now just the "wait for krane convergence" primitive —
	// all state was written above, so pass an empty request.
	if err := s.enqueueDeploy(ctx, sentinel.ID, &hydrav1.SentinelServiceDeployRequest{}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.ChangeReplicasResponse{}), nil
}
