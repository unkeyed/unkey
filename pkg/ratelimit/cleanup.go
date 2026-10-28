package ratelimit

import (
	"context"
	"errors"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
)

// errCleanupBacklog means that a cleanup pass stopped at its bound with
// faded rows left. The next pass then starts after `cleanupBacklogInterval`
// instead of `cleanupInterval`.
var errCleanupBacklog = errors.New("the cleanup pass reached its bound with faded rows left")

// cleanUpUntilStopped runs a cleanup pass about every `cleanupInterval`, or
// about every `cleanupBacklogInterval` while passes end at their bound,
// until `ctx` is canceled. Only New starts it, in one goroutine.
func (l *limiter) cleanUpUntilStopped(ctx context.Context) {
	interval := cleanupInterval
	for {
		jitter := interval / 5
		timer := l.clock.NewTimer(interval - jitter + time.Duration(l.randomInt64N(2*jitter.Nanoseconds())))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C():
		}
		passCtx, cancel := context.WithTimeout(ctx, cleanupTimeout)
		err := l.cleanUp(passCtx)
		cancel()
		interval = cleanupInterval
		if errors.Is(err, errCleanupBacklog) {
			interval = cleanupBacklogInterval
		}
		if err != nil && !errors.Is(err, errCleanupBacklog) && ctx.Err() == nil {
			logger.Warn("unable to clean up rate limit rows", "error", err.Error())
		}
	}
}

// cleanUp runs one cleanup pass and counts it in `cleanupPassesTotal`. See
// `deleteFadedRows`. It is safe to call concurrently. The caller must not
// hold a shard lock, because it waits on the store.
func (l *limiter) cleanUp(ctx context.Context) error {
	err := l.deleteFadedRows(ctx)
	switch {
	case err == nil:
		cleanupPassesTotal.WithLabelValues("complete").Inc()
	case errors.Is(err, errCleanupBacklog):
		cleanupPassesTotal.WithLabelValues("bound").Inc()
	default:
		cleanupPassesTotal.WithLabelValues("error").Inc()
	}
	return err
}

// deleteFadedRows deletes rows whose `DeleteAfterNs` has passed by the store
// clock, in batches of `rowsPerDelete`, until a batch is not full or the pass
// has run `maxDeletesPerCleanupPass` batches. Then it returns
// `errCleanupBacklog`.
//
// Pods do not coordinate. Deletes of the same rows by two pods are
// harmless, and so is a write that races a delete: either the write raised
// `DeleteAfterNs` first and the row stays, or it lands after and inserts the
// pod's whole sum again. The row then holds what it would have held without
// the delete.
//
// It is safe to call concurrently: it uses only the store and `randomMu`.
// The caller must not hold a shard lock, because it waits on the store.
func (l *limiter) deleteFadedRows(ctx context.Context) error {
	nowNs, err := l.store.NowNs(ctx)
	if err != nil {
		return err
	}
	if time.Duration(nowNs-l.clock.Now().UnixNano()).Abs() >= maxClockSkew {
		clockSkewExceededTotal.Inc()
		logger.Warn("the pod clock differs from the MySQL clock by more than the rate limiter allows", "max_clock_skew", maxClockSkew.String())
	}
	for range maxDeletesPerCleanupPass {
		deleted, err := l.store.DeleteFadedRows(ctx, nowNs, rowsPerDelete)
		if err != nil {
			return err
		}
		rowsDeletedTotal.Add(float64(deleted))
		if deleted < rowsPerDelete {
			return nil
		}
	}
	return errCleanupBacklog
}
