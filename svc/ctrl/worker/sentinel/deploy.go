package sentinel

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/observability"
)

const workflowSentinelDeploy = "sentinel_deploy"

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
func (s *Service) Deploy(
	ctx restate.ObjectContext,
	req *hydrav1.SentinelServiceDeployRequest,
) (resp *hydrav1.SentinelServiceDeployResponse, retErr error) {
	defer observability.RunTimer(workflowSentinelDeploy, &retErr)()

	sentinelID := restate.Key(ctx)

	// Read current config to detect no-ops and to merge partial updates.
	current, err := restate.Run(ctx, func(rc restate.RunContext) (db.Sentinel, error) {
		return db.Query.FindSentinelByID(rc, s.db.RO(), sentinelID)
	}, restate.WithName("read current sentinel"))
	if err != nil {
		return nil, fmt.Errorf("read sentinel %s: %w", sentinelID, err)
	}

	// Merge request fields over current config (zero values mean "keep current").
	newImage := current.Image
	if req.GetImage() != "" {
		newImage = req.GetImage()
	}

	newCPU := current.CpuMillicores
	if req.GetCpuMillicores() != 0 {
		newCPU = req.GetCpuMillicores()
	}

	newMem := current.MemoryMib
	if req.GetMemoryMib() != 0 {
		newMem = req.GetMemoryMib()
	}

	newReplicas := current.DesiredReplicas
	if req.GetDesiredReplicas() != 0 {
		newReplicas = req.GetDesiredReplicas()
	}

	noConfigChange := newImage == current.Image && newCPU == current.CpuMillicores &&
		newMem == current.MemoryMib && newReplicas == current.DesiredReplicas

	// No config change, already serving, and desired image is actually running: nothing to do.
	if noConfigChange &&
		current.Health == db.SentinelsHealthHealthy &&
		current.RunningImage == current.Image {
		return &hydrav1.SentinelServiceDeployResponse{
			Status: hydrav1.SentinelDeployStatus_SENTINEL_DEPLOY_STATUS_READY,
		}, nil
	}

	// Apply new config if changed, or just mark progressing if waiting for first startup.
	if !noConfigChange {
		logger.Info("deploying sentinel",
			"sentinel_id", sentinelID,
			"image", newImage,
			"cpu", newCPU,
			"mem", newMem,
			"replicas", newReplicas,
		)

		err := s.applyConfigAndNotify(ctx, sentinelID, current.RegionID, db.UpdateSentinelConfigParams{
			ID:              sentinelID,
			Image:           newImage,
			CpuMillicores:   newCPU,
			MemoryMib:       newMem,
			DesiredReplicas: newReplicas,
			DeployStatus:    db.SentinelsDeployStatusProgressing,
			UpdatedAt:       sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}, "apply config")
		if err != nil {
			return nil, fmt.Errorf("apply config: %w", err)
		}
	} else {
		// No config change but not healthy yet (first startup). Mark progressing.
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

	// The caller relies on the FAILED status field rather than a returned
	// error (so Restate doesn't retry the whole handler). Force the deferred
	// RunTimer to see a categorized error so this terminal-failure path
	// shows up correctly in metrics.
	retErr = fault.Wrap(
		fmt.Errorf("sentinel %s did not become healthy in %v", sentinelID, deployTimeout),
		fault.Code(codes.Workflow.Infra.SentinelDeployTimeout.URN()),
	)
	logger.Error("sentinel deploy timed out, returning FAILED status",
		"sentinel_id", sentinelID, "error", retErr)

	resp = &hydrav1.SentinelServiceDeployResponse{
		Status: hydrav1.SentinelDeployStatus_SENTINEL_DEPLOY_STATUS_FAILED,
	}
	// Convert to a terminal so Restate treats it as final (not retried),
	// matching the original (FAILED, nil) intent while exposing the cause.
	retErr = restate.TerminalError(retErr)
	return resp, retErr
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
