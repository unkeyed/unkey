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

	deploymentChanges := s.deploymentcontroller.Changes()
	sentinelChanges := s.sentinelcontroller.Changes()

	for {
		select {
		case <-s.close:
			return
		case c := <-deploymentChanges:
			err := r.Do(func() error {
				_, err := cb.Do(context.Background(), func(ctx context.Context) (any, error) {
					_, err := s.ctrl.UpdateInstance(ctx, connect.NewRequest(c))
					return nil, err
				})
				return err
			})
			if err != nil {
				s.logger.Error("failed to push deployment update", "error", err.Error())
			}

		case c := <-sentinelChanges:
			err := r.Do(func() error {
				_, err := cb.Do(context.Background(), func(ctx context.Context) (any, error) {
					_, err := s.ctrl.UpdateSentinel(ctx, connect.NewRequest(c))
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
