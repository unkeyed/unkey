package usagelimiter

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/buffer"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
)

func TestReplayWithRetryRecoversTransientFailure(t *testing.T) {
	var calls atomic.Int64
	svc := &counterService{
		decrementKeyCredits: func(context.Context, string, int64) error {
			if calls.Add(1) < 3 {
				return errors.New("transient connection error")
			}
			return nil
		},
		dbCircuitBreaker: circuitbreaker.New[any]("test_replay_retry"),
	}

	err := svc.replayWithRetry(context.Background(), CreditChange{KeyID: "key_test", Cost: 1})
	require.NoError(t, err)
	require.EqualValues(t, 3, calls.Load())
}

func TestReplayWithRetryGivesUpAfterBoundedAttempts(t *testing.T) {
	var calls atomic.Int64
	svc := &counterService{
		decrementKeyCredits: func(context.Context, string, int64) error {
			calls.Add(1)
			return errors.New("database unavailable")
		},
		dbCircuitBreaker: circuitbreaker.New[any]("test_replay_drop"),
	}

	err := svc.replayWithRetry(context.Background(), CreditChange{KeyID: "key_test", Cost: 1})
	require.Error(t, err)
	require.EqualValues(t, replayMaxAttempts, calls.Load())
}

func TestReplayRequestsRetriesBeforeDiscarding(t *testing.T) {
	var calls atomic.Int64
	svc := &counterService{
		decrementKeyCredits: func(context.Context, string, int64) error {
			if calls.Add(1) < 2 {
				return errors.New("transient connection error")
			}
			return nil
		},
		replayBuffer:     buffer.New[CreditChange](buffer.Config{Name: "test_replay_requests", Capacity: 1, Drop: false}),
		dbCircuitBreaker: circuitbreaker.New[any]("test_replay_requests"),
	}

	svc.replayBuffer.Buffer(CreditChange{KeyID: "key_test", Cost: 1})
	svc.replayBuffer.Close()
	svc.replayRequests()

	require.EqualValues(t, 2, calls.Load())
}
