package sync

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

func (s *SyncEngine) push() {

	s.logger.Info("starting push to control plane")

	r := retry.New(
		retry.Attempts(5),
		retry.Backoff(func(n int) time.Duration { return time.Second * time.Duration(n*n) }),
	)
	cb := circuitbreaker.New[any]("krane_push")

	for {
		select {
		case <-s.close:
			return
		case d := <-s.instanceUpdates.Consume():
			err := r.Do(func() error {
				_, err := cb.Do(context.Background(), func(ctx context.Context) (any, error) {
					_, err := s.ctrl.UpdateInstance(ctx, connect.NewRequest(d))
					return nil, err
				})
				return err
			})
			if err != nil {
				s.logger.Error("failed to push deployment update", "error", err.Error())
			}

		case g := <-s.sentinelUpdates.Consume():
			err := r.Do(func() error {
				_, err := cb.Do(context.Background(), func(ctx context.Context) (any, error) {
					_, err := s.ctrl.UpdateSentinel(ctx, connect.NewRequest(g))
					return nil, err
				})
				return err
			})
			if err != nil {
				s.logger.Error("failed to push sentinel update", "error", err.Error())
			}

		}
	}

}
