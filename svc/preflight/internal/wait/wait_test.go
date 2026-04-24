package wait

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoll_SuccessOnFirstAttempt(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := Poll(ctx, 10*time.Millisecond, func(context.Context) (int, bool, error) {
		return 42, true, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestPoll_SuccessAfterRetries(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var attempts atomic.Int32
	got, err := Poll(ctx, 5*time.Millisecond, func(context.Context) (string, bool, error) {
		n := attempts.Add(1)
		if n < 3 {
			return "", false, nil
		}
		return "ok", true, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ok" {
		t.Fatalf("expected ok, got %q", got)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestPoll_ReturnsDeadlineOnTimeout(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err := Poll(ctx, 5*time.Millisecond, func(context.Context) (int, bool, error) {
		return 0, false, nil
	})
	if !errors.Is(err, ErrDeadline) {
		t.Fatalf("expected ErrDeadline, got %v", err)
	}
}

func TestPoll_PropagatesConditionError(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := Poll(ctx, 10*time.Millisecond, func(context.Context) (int, bool, error) {
		return 0, false, boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
}

func TestPoll_AlreadyCancelledContextReturnsDeadline(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var called atomic.Bool
	_, err := Poll(ctx, 10*time.Millisecond, func(context.Context) (int, bool, error) {
		called.Store(true)
		return 0, true, nil
	})
	if !errors.Is(err, ErrDeadline) {
		t.Fatalf("expected ErrDeadline, got %v", err)
	}
	if called.Load() {
		t.Fatalf("Poll must not invoke fn when ctx is already cancelled")
	}
}
