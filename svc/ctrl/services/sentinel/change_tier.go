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
)

// ChangeTier swaps a running sentinel to a new (tier_id, version). In one
// transaction it inserts a fresh sentinel_subscriptions row denormalizing
// the tier's price + envelope, repoints sentinels.subscription_id, marks
// deploy_status=progressing, and writes a deployment_changes outbox row so
// krane rolls the pod to the new resource envelope.
//
// After the tx commits, it fires an empty SentinelService.Deploy through
// Restate ingress. Deploy's noConfigChange path creates an awakeable and
// awaits NotifyReady, which ReportSentinelStatus fires when krane's
// convergedImage check passes — i.e. k8s has fully rolled out the new spec.
// deploy_status flips to ready (or failed on timeout) inside Deploy.
func (s *Service) ChangeTier(
	ctx context.Context,
	req *connect.Request[ctrlv1.ChangeTierRequest],
) (*connect.Response[ctrlv1.ChangeTierResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	if err := assert.All(
		assert.NotEmpty(req.Msg.GetSentinelId(), "sentinel_id is required"),
		assert.NotEmpty(req.Msg.GetTierId(), "tier_id is required"),
		assert.NotEmpty(req.Msg.GetTierVersion(), "tier_version is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	sentinelID := req.Msg.GetSentinelId()
	joined, err := db.Query.FindSentinelByID(ctx, s.db.RO(), sentinelID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	sentinel, currentSub := joined.Sentinel, joined.SentinelSubscription

	// No-op: already on this tier + version.
	if currentSub.TierID == req.Msg.GetTierId() && currentSub.TierVersion == req.Msg.GetTierVersion() {
		return connect.NewResponse(&ctrlv1.ChangeTierResponse{}), nil
	}

	tier, err := db.Query.FindSentinelTier(ctx, s.db.RO(), db.FindSentinelTierParams{
		TierID:  req.Msg.GetTierId(),
		Version: req.Msg.GetTierVersion(),
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("tier not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if tier.EffectiveUntil.Valid {
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf("tier is retired and cannot be subscribed to"),
		)
	}

	newSubscriptionID := uid.New(uid.SentinelSubscriptionPrefix)
	now := time.Now().UnixMilli()

	err = db.Tx(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		// Close the currently open subscription before opening the new one.
		// The `one_open_subscription_per_sentinel` unique index would
		// otherwise reject the insert.
		if err := db.Query.TerminateOpenSentinelSubscription(txCtx, tx, db.TerminateOpenSentinelSubscriptionParams{
			SentinelID:   sentinel.ID,
			TerminatedAt: sql.NullInt64{Valid: true, Int64: now},
		}); err != nil {
			return fmt.Errorf("terminate open sentinel subscription: %w", err)
		}

		if err := db.Query.InsertSentinelSubscription(txCtx, tx, db.InsertSentinelSubscriptionParams{
			ID:            newSubscriptionID,
			SentinelID:    sentinel.ID,
			WorkspaceID:   sentinel.WorkspaceID,
			RegionID:      sentinel.RegionID,
			TierID:        tier.TierID,
			TierVersion:   tier.Version,
			CpuMillicores: tier.CpuMillicores,
			MemoryMib:     tier.MemoryMib,
			// Carry the replica count forward from the subscription being
			// terminated — tier change doesn't touch the pod count, just
			// the resource envelope and per-second price.
			Replicas:       currentSub.Replicas,
			PricePerSecond: tier.PricePerSecond,
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

		if err := db.Query.UpdateSentinelDeployStatus(txCtx, tx, db.UpdateSentinelDeployStatusParams{
			ID:           sentinel.ID,
			DeployStatus: db.SentinelsDeployStatusProgressing,
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: now},
		}); err != nil {
			return fmt.Errorf("mark progressing: %w", err)
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

	if err := s.enqueueDeploy(ctx, sentinel.ID, &hydrav1.SentinelServiceDeployRequest{}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.ChangeTierResponse{}), nil
}
