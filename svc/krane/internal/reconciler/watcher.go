package reconciler

import (
	"context"
	"math/rand/v2"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

func (r *Reconciler) Watch(ctx context.Context) {

	intervalMin := time.Second
	intervalMax := 5 * time.Second

	for {

		interval := intervalMin + time.Millisecond*time.Duration(rand.Float64()*float64(intervalMax.Milliseconds()-intervalMin.Milliseconds()))
		time.Sleep(interval)

		err := r.watch(ctx)
		if err != nil {
			r.logger.Error("error while watching for state changes", "error", err)
		}

	}

}

func (r *Reconciler) watch(ctx context.Context) error {

	r.logger.Info("starting watch")

	stream, err := r.cluster.Sync(ctx, connect.NewRequest(&ctrlv1.SyncRequest{
		Region:           r.region,
		SequenceLastSeen: r.sequenceLastSeen,
	}))
	if err != nil {
		return err
	}

	for stream.Receive() {
		r.logger.Info("received message")
		if err := r.HandleState(ctx, stream.Msg()); err != nil {
			r.logger.Error("error handling state", "error", err)
		}
	}
	err = stream.Close()
	if err != nil {
		r.logger.Error("unable to close stream", "error", err)
	}
	return nil

}
