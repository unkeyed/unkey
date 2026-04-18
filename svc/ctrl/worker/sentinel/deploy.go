package sentinel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/compensation"
)

const (
	// deployTimeout is how long Deploy waits for krane to report the sentinel
	// as healthy before marking the deploy as failed.
	deployTimeout = 10 * time.Minute

	// notifyReadyAwakeableKey is the Restate virtual object state key used to store
	// the awakeable ID so the NotifyReady shared handler can resolve it.
	notifyReadyAwakeableKey = "notify_ready_awakeable"
)

// Deploy updates a sentinel's configuration and waits for krane to report
// it as healthy via a Restate awakeable. If the sentinel does not become
// healthy within the timeout, the deploy is marked as failed.
//
// Compensation: any abnormal exit (non-nil retErr) flips deploy_status back
// to `failed` via MarkSentinelFailedIfProgressing. Conditional so a concurrent
// ReportSentinelStatus that already observed convergence and flipped to
// `ready` isn't overwritten.
func (s *Service) Deploy(
	ctx restate.ObjectContext,
	req *hydrav1.SentinelServiceDeployRequest,
) (_ *hydrav1.SentinelServiceDeployResponse, retErr error) {
	sentinelID := restate.Key(ctx)

	comp := compensation.New()
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, comp.Execute(ctx))
		}
	}()

	// Read current config to detect no-ops and to merge partial updates.
	joined, err := restate.Run(ctx, func(rc restate.RunContext) (db.FindSentinelByIDRow, error) {
		return db.Query.FindSentinelByID(rc, s.db.RO(), sentinelID)
	}, restate.WithName("read current sentinel"))
	if err != nil {
		return nil, fmt.Errorf("read sentinel %s: %w", sentinelID, err)
	}
	sentinel := joined.Sentinel

	// Register the progressing→failed reset as soon as we know we might
	// flip status to progressing. Runs in reverse order if we exit with
	// an error after this point. Uses the conditional update so a
	// successful "mark ready" immediately before the error is not
	// overwritten.
	comp.Add("reset stuck sentinel to failed", func(rc restate.RunContext) error {
		return db.Query.MarkSentinelFailedIfProgressing(rc, s.db.RW(), db.MarkSentinelFailedIfProgressingParams{
			ID:        sentinelID,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	})

	// Merge request fields over current config (zero values mean "keep current").
	// Replica changes are NOT driven through Deploy — ChangeReplicas writes
	// them synchronously before enqueuing this workflow. Deploy handles only
	// image rollouts here; any DesiredReplicas on the request is ignored.
	newImage := sentinel.Image
	if req.GetImage() != "" {
		newImage = req.GetImage()
	}

	noConfigChange := newImage == sentinel.Image

	// Steady state: nothing changing here AND no rollout is already in flight
	// (e.g. one kicked off by ChangeTier, which marks progressing in its own
	// tx without touching image or replicas). deploy_status is the authoritative
	// rollout signal; Health + RunningImage are observational and can lie — on
	// a tier swap the image is unchanged so RunningImage trivially matches
	// until krane picks up the outbox.
	if noConfigChange && sentinel.DeployStatus == db.SentinelsDeployStatusReady {
		return &hydrav1.SentinelServiceDeployResponse{
			Status: hydrav1.SentinelDeployStatus_SENTINEL_DEPLOY_STATUS_READY,
		}, nil
	}

	// Apply new config if this invocation carries one. If the caller already
	// wrote config + outbox + progressing elsewhere (ChangeTier, or a prior
	// Deploy that resumed) we fall through straight to awaiting NotifyReady.
	if !noConfigChange {
		logger.Info("deploying sentinel",
			"sentinel_id", sentinelID,
			"image", newImage,
		)

		err := s.applyConfigAndNotify(ctx, sentinelID, sentinel.RegionID, db.UpdateSentinelConfigParams{
			ID:              sentinelID,
			Image:           newImage,
			DesiredReplicas: sentinel.DesiredReplicas,
			DeployStatus:    db.SentinelsDeployStatusProgressing,
			UpdatedAt:       sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}, "apply config")
		if err != nil {
			return nil, fmt.Errorf("apply config: %w", err)
		}
	} else if sentinel.DeployStatus != db.SentinelsDeployStatusProgressing {
		// No config change and no rollout marker yet (rare path: first Deploy
		// after a sentinel insert that didn't flip status for us). Mark it.
		_, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateSentinelDeployStatus(rc, s.db.RW(), db.UpdateSentinelDeployStatusParams{
				ID:           sentinelID,
				DeployStatus: db.SentinelsDeployStatusProgressing,
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName("mark progressing"))
		if err != nil {
			return nil, fmt.Errorf("mark progressing: %w", err)
		}
	}

	// Create an awakeable that NotifyReady will resolve when krane reports healthy.
	awk := restate.Awakeable[restate.Void](ctx)
	restate.Set(ctx, notifyReadyAwakeableKey, awk.Id())

	// Race the awakeable against a timeout.
	timeout := restate.After(ctx, deployTimeout)
	winner, err := restate.WaitFirst(ctx, awk, timeout)
	if err != nil {
		return nil, fmt.Errorf("wait for ready or timeout: %w", err)
	}

	// Clear the awakeable from state regardless of outcome.
	restate.Clear(ctx, notifyReadyAwakeableKey)

	if winner == awk {
		// Drain the awakeable result to check for errors.
		if _, err := awk.Result(); err != nil {
			return nil, fmt.Errorf("awakeable result: %w", err)
		}

		_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateSentinelDeployStatus(rc, s.db.RW(), db.UpdateSentinelDeployStatusParams{
				ID:           sentinelID,
				DeployStatus: db.SentinelsDeployStatusReady,
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName("mark ready"))
		if err != nil {
			return nil, fmt.Errorf("mark ready: %w", err)
		}

		logger.Info("sentinel deploy ready", "sentinel_id", sentinelID)
		return &hydrav1.SentinelServiceDeployResponse{
			Status: hydrav1.SentinelDeployStatus_SENTINEL_DEPLOY_STATUS_READY,
		}, nil
	}

	// Timeout fired: mark as failed.
	if err := timeout.Done(); err != nil {
		return nil, fmt.Errorf("timeout: %w", err)
	}

	logger.Warn("sentinel deploy timed out", "sentinel_id", sentinelID)
	_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateSentinelDeployStatus(rc, s.db.RW(), db.UpdateSentinelDeployStatusParams{
			ID:           sentinelID,
			DeployStatus: db.SentinelsDeployStatusFailed,
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("mark failed"))
	if err != nil {
		return nil, fmt.Errorf("mark failed: %w", err)
	}

	return &hydrav1.SentinelServiceDeployResponse{
		Status: hydrav1.SentinelDeployStatus_SENTINEL_DEPLOY_STATUS_FAILED,
	}, nil
}

// NotifyReady resolves the awakeable created by Deploy, unblocking it.
// This is a shared handler so it can run concurrently with a suspended Deploy.
func (s *Service) NotifyReady(
	ctx restate.ObjectSharedContext,
	_ *hydrav1.SentinelServiceNotifyReadyRequest,
) (*hydrav1.SentinelServiceNotifyReadyResponse, error) {
	id, err := restate.Get[string](ctx, notifyReadyAwakeableKey)
	if err != nil {
		return nil, fmt.Errorf("get awakeable id: %w", err)
	}
	if id == "" {
		// No pending deploy — nothing to resolve.
		return &hydrav1.SentinelServiceNotifyReadyResponse{}, nil
	}

	restate.ResolveAwakeable[restate.Void](ctx, id, restate.Void{})
	return &hydrav1.SentinelServiceNotifyReadyResponse{}, nil
}

// applyConfigAndNotify updates sentinel config and inserts a deployment_changes
// outbox entry in a single transaction. Krane picks up the outbox entry and
// applies the new config to Kubernetes.
//
// Billing / subscription rotation lives at the intent layer (ChangeTier,
// ChangeReplicas) — Deploy is purely the convergence mechanism.
func (s *Service) applyConfigAndNotify(
	ctx restate.ObjectContext,
	sentinelID string,
	regionID string,
	params db.UpdateSentinelConfigParams,
	stepName string,
) error {
	_, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Tx(rc, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			if err := db.Query.UpdateSentinelConfig(txCtx, tx, params); err != nil {
				return fmt.Errorf("update sentinel config: %w", err)
			}
			return db.Query.InsertDeploymentChange(txCtx, tx, db.InsertDeploymentChangeParams{
				ResourceType: db.DeploymentChangesResourceTypeSentinel,
				ResourceID:   sentinelID,
				RegionID:     regionID,
				CreatedAt:    time.Now().UnixMilli(),
			})
		})
	}, restate.WithName(stepName))
	return err
}
