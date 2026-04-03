package deployment

import (
	"context"
	"math/rand/v2"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// runDesiredStateApplyLoop connects to the control plane's WatchDeployments
// stream and applies desired state updates to the Kubernetes cluster.
//
// The loop automatically reconnects with jittered backoff (1-5 seconds) on stream
// errors or disconnections. Each received state is processed via [Controller.ApplyDeployment]
// or [Controller.DeleteDeployment], and the version cursor is advanced on successful
// processing to enable resumable streaming.
//
// Every fullSyncInterval the version cursor is reset to 0 so the next stream
// replays all topology rows. This acts as a consistency safety net: if a DB
// update didn't bump the version (e.g. manual fix), or if a message was
// processed but the apply failed silently, the periodic full sync will
// reconcile the drift.
//
// The loop runs indefinitely until the context is cancelled. It does not use the
// done channel since the jittered sleep handles graceful reconnection.
func (c *Controller) runDesiredStateApplyLoop(ctx context.Context) {
	const fullSyncInterval = 5 * time.Minute

	intervalMin := time.Second
	intervalMax := 5 * time.Second
	lastFullSync := time.Now()

	for {
		interval := intervalMin + time.Millisecond*time.Duration(rand.Float64()*float64(intervalMax.Milliseconds()-intervalMin.Milliseconds()))
		time.Sleep(interval)

		if time.Since(lastFullSync) >= fullSyncInterval {
			c.versionLastSeen = 0
			lastFullSync = time.Now()
			logger.Info("resetting deployment watch cursor for full sync")
		}

		err := c.streamDesiredStateOnce(ctx)
		if err != nil {
			logger.Error("error streaming desired state from control plane", "error", err)
		}
	}
}

// streamDesiredStateOnce opens a single connection to the control plane's
// WatchDeployments stream, processes all received states until the stream
// closes or errors, then returns.
//
// The method sends the current versionLastSeen to resume from where it left off,
// avoiding reprocessing of already-applied states. On each received message, it
// dispatches to [Controller.ApplyDeployment] or [Controller.DeleteDeployment] based
// on the state type. Individual apply/delete failures are logged and skipped so
// that one broken deployment does not block the entire stream. The version cursor
// advances past failed states; the periodic full sync will retry them.
func (c *Controller) streamDesiredStateOnce(ctx context.Context) error {
	logger.Info("connecting to control plane for desired state")

	stream, err := c.cluster.WatchDeployments(ctx, &ctrlv1.WatchDeploymentsRequest{
		Region:          c.region,
		VersionLastSeen: c.versionLastSeen,
	})
	if err != nil {
		return err
	}

	for stream.Receive() {
		state := stream.Msg()

		switch op := state.GetState().(type) {
		case *ctrlv1.DeploymentState_Apply:
			logger.Info("applying deployment from stream", "deployment_id", op.Apply.GetDeploymentId(), "version", state.GetVersion())
			if err := c.ApplyDeployment(ctx, op.Apply); err != nil {
				logger.Error("unable to apply deployment", "error", err, "deployment_id", op.Apply.GetDeploymentId())
			}
		case *ctrlv1.DeploymentState_Delete:
			logger.Info("deleting deployment from stream", "namespace", op.Delete.GetK8SNamespace(), "name", op.Delete.GetK8SName(), "version", state.GetVersion())
			if err := c.DeleteDeployment(ctx, op.Delete); err != nil {
				logger.Error("unable to delete deployment", "error", err, "namespace", op.Delete.GetK8SNamespace(), "name", op.Delete.GetK8SName())
			}
		}

		if state.GetVersion() > c.versionLastSeen {
			c.versionLastSeen = state.GetVersion()
		}
	}

	if err := stream.Close(); err != nil {
		logger.Error("unable to close control plane stream", "error", err)
		return err
	}

	return nil
}
