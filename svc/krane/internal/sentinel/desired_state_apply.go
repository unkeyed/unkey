package sentinel

import (
	"context"
	"math/rand/v2"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

// runDesiredStateApplyLoop connects to the control plane's SyncSentinels stream
// and applies desired state updates to the Kubernetes cluster.
//
// The loop automatically reconnects with jittered backoff on stream errors.
// Each received state is processed via applyDesiredState, and the version cursor
// is advanced on successful processing.
func (c *Controller) runDesiredStateApplyLoop(ctx context.Context) {
	intervalMin := time.Second
	intervalMax := 5 * time.Second

	for {
		interval := intervalMin + time.Millisecond*time.Duration(rand.Float64()*float64(intervalMax.Milliseconds()-intervalMin.Milliseconds()))
		time.Sleep(interval)

		err := c.streamDesiredStateOnce(ctx)
		if err != nil {
			c.logger.Error("error streaming desired state from control plane", "error", err)
		}
	}
}

// streamDesiredStateOnce opens a single connection to the control plane's
// WatchSentinels stream, processes all received states until the stream
// closes or errors, then returns. The caller handles reconnection.
func (c *Controller) streamDesiredStateOnce(ctx context.Context) error {
	c.logger.Info("connecting to control plane for desired state")

	stream, err := c.cluster.WatchSentinels(ctx, connect.NewRequest(&ctrlv1.WatchSentinelsRequest{
		Region:          c.region,
		VersionLastSeen: c.versionLastSeen,
	}))
	if err != nil {
		return err
	}

	for stream.Receive() {
		c.logger.Info("received desired state from control plane")
		state := stream.Msg()

		switch op := state.GetState().(type) {
		case *ctrlv1.SentinelState_Apply:
			if err := c.ApplySentinel(ctx, op.Apply); err != nil {
				return err
			}
		case *ctrlv1.SentinelState_Delete:
			if err := c.DeleteSentinel(ctx, op.Delete); err != nil {
				return err
			}
		}
		if state.GetVersion() > c.versionLastSeen {
			c.versionLastSeen = state.GetVersion()
		}
	}

	if err := stream.Close(); err != nil {
		c.logger.Error("unable to close control plane stream", "error", err)
		return err
	}

	return nil
}
