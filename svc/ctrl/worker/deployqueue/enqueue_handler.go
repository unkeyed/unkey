package deployqueue

import (
	"database/sql"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Enqueue adds a deploy request to the queue. Since the queue is keyed by
// app_id:branch, all entries are for the same branch. New entries supersede
// all older queued entries (marked as superseded in DB). If an active deploy
// exists for the same branch, its Restate invocation is cancelled.
func (s *Service) Enqueue(ctx restate.ObjectContext, req *hydrav1.QueueEnqueueRequest) (*hydrav1.QueueEnqueueResponse, error) {
	// Persist workspace ID for later scheduler calls.
	restate.Set(ctx, stateWorkspaceID, req.GetWorkspaceId())

	// --- Supersede queued entries ---
	queue, _ := restate.Get[[]queueEntry](ctx, stateQueue)
	if len(queue) > 0 {
		for _, entry := range queue {
			logger.Info("superseding queued deployment",
				"superseded", entry.DeploymentID,
				"by", req.GetDeploymentId(),
			)
			restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
				return db.Query.UpdateDeploymentStatus(runCtx, s.db.RW(), db.UpdateDeploymentStatusParams{
					ID:        entry.DeploymentID,
					Status:    db.DeploymentsStatusSuperseded,
					UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				})
			}, restate.WithName("mark superseded: "+entry.DeploymentID))
		}
		// Clear the queue — all entries are superseded.
		queue = nil
	}

	// --- Cancel active deploy if one exists ---
	active, _ := restate.Get[*activeDeploy](ctx, stateActive)
	if active != nil {
		logger.Info("cancelling active deployment",
			"cancelled", active.DeploymentID,
			"by", req.GetDeploymentId(),
		)
		// Cancel the Restate invocation. This triggers the deploy workflow's
		// compensation stack which will mark it as superseded and clean up.
		restate.CancelInvocation(ctx, active.InvocationID)

		// Release the build slot immediately. We can't rely on OnDeployComplete
		// because we're about to clear the active state — the callback would
		// see active==nil and skip the release, leaking the slot.
		hydrav1.NewDeploySchedulerServiceClient(ctx, req.GetWorkspaceId()).ReleaseBuildSlot().
			Send(&hydrav1.ReleaseBuildSlotRequest{
				QueueKey: restate.Key(ctx),
			})

		restate.Set[*activeDeploy](ctx, stateActive, nil)
	}

	// --- Add new entry ---
	now, _ := restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
		return time.Now().UnixMilli(), nil
	}, restate.WithName("get timestamp"))

	// Serialize the DeployRequest using protojson because Restate state uses
	// encoding/json which can't round-trip protobuf oneof fields.
	deployReqBytes, marshalErr := marshalDeployRequest(req.GetDeployRequest())
	if marshalErr != nil {
		return nil, restate.TerminalError(marshalErr)
	}

	entry := queueEntry{
		DeploymentID:   req.GetDeploymentId(),
		DeployReqBytes: deployReqBytes,
		IsProduction:   req.GetIsProduction(),
		Branch:         req.GetBranch(),
		EnqueuedAt:     now,
	}

	// Insert in priority order: production entries go before preview entries.
	inserted := false
	for i, existing := range queue {
		if entry.IsProduction && !existing.IsProduction {
			queue = append(queue[:i+1], queue[i:]...)
			queue[i] = entry
			inserted = true
			break
		}
	}
	if !inserted {
		queue = append(queue, entry)
	}
	restate.Set(ctx, stateQueue, queue)

	// Kick off processing if nothing is active. If we just cancelled an
	// active deploy above, active was set to nil so this triggers.
	// If there's still an active deploy (no superseding happened), the new
	// entry just waits in the queue — OnDeployComplete will trigger ProcessNext.
	currentActive, _ := restate.Get[*activeDeploy](ctx, stateActive)
	if currentActive == nil {
		hydrav1.NewDeployQueueServiceClient(ctx, restate.Key(ctx)).ProcessNext().Send(&hydrav1.ProcessNextRequest{})
	}

	return &hydrav1.QueueEnqueueResponse{}, nil
}
