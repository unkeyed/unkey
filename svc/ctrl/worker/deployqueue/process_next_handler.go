package deployqueue

import (
	"database/sql"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// ProcessNext pops the highest-priority item from the queue, requests a build
// slot from the workspace scheduler, and dispatches the deploy if granted. If
// no slot is available, the entry is pushed back and the scheduler will call
// ProcessNext later when a slot opens.
func (s *Service) ProcessNext(ctx restate.ObjectContext, _ *hydrav1.ProcessNextRequest) (*hydrav1.ProcessNextResponse, error) {
	queue, _ := restate.Get[[]queueEntry](ctx, stateQueue)
	if len(queue) == 0 {
		return &hydrav1.ProcessNextResponse{}, nil
	}

	// Check if something is already active — shouldn't happen, but be safe.
	active, _ := restate.Get[*activeDeploy](ctx, stateActive)
	if active != nil {
		logger.Info("ProcessNext called but deploy already active",
			"active_deployment", active.DeploymentID,
		)
		return &hydrav1.ProcessNextResponse{}, nil
	}

	// Pop the first entry (highest priority).
	entry := queue[0]
	queue = queue[1:]
	restate.Set(ctx, stateQueue, queue)

	// Request a build slot from the workspace scheduler.
	workspaceID, _ := restate.Get[string](ctx, stateWorkspaceID)
	resp, err := hydrav1.NewDeploySchedulerServiceClient(ctx, workspaceID).RequestBuildSlot().
		Request(&hydrav1.RequestBuildSlotRequest{
			QueueKey:     restate.Key(ctx),
			IsProduction: entry.IsProduction,
		})
	if err != nil {
		// On error, push the entry back and retry later.
		queue = append([]queueEntry{entry}, queue...)
		restate.Set(ctx, stateQueue, queue)
		return nil, err
	}

	if !resp.GetGranted() {
		// No slot available. Push entry back to the front of the queue.
		// The scheduler will call ProcessNext when a slot becomes available.
		queue = append([]queueEntry{entry}, queue...)
		restate.Set(ctx, stateQueue, queue)
		logger.Info("build slot not available, waiting",
			"deployment_id", entry.DeploymentID,
			"queue_key", restate.Key(ctx),
		)
		return &hydrav1.ProcessNextResponse{}, nil
	}

	// Deserialize the deploy request from protojson bytes.
	deployReq, unmarshalErr := unmarshalDeployRequest(entry.DeployReqBytes)
	if unmarshalErr != nil {
		return nil, restate.TerminalError(unmarshalErr)
	}

	// Slot granted — dispatch the deploy workflow.
	queueKey := restate.Key(ctx)
	invocation := hydrav1.NewDeployServiceClient(ctx, queueKey).Deploy().
		Send(deployReq)

	invocationID := invocation.GetInvocationId()

	// Store the invocation ID in the DB for UI cancellation.
	restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.UpdateDeploymentRestateInvocationID(runCtx, s.db.RW(), db.UpdateDeploymentRestateInvocationIDParams{
			ID:                  entry.DeploymentID,
			RestateInvocationID: sql.NullString{String: invocationID, Valid: true},
			UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("store invocation id"))

	// Mark as active.
	restate.Set(ctx, stateActive, &activeDeploy{
		DeploymentID: entry.DeploymentID,
		InvocationID: invocationID,
		IsProduction: entry.IsProduction,
	})

	logger.Info("dispatched deploy",
		"deployment_id", entry.DeploymentID,
		"invocation_id", invocationID,
		"queue_key", queueKey,
	)

	return &hydrav1.ProcessNextResponse{}, nil
}
