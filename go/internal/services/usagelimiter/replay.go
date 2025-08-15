package usagelimiter

import (
	"context"
	"database/sql"
	"time"

	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

// replayRequests processes buffered credit changes and writes them to the database
// This follows the same pattern as ratelimit service
func (s *redisService) replayRequests() {
	for change := range s.replayBuffer.Consume() {
		err := s.syncWithDB(context.Background(), change)
		if err != nil {
			s.logger.Error("failed to replay credit change", "error", err.Error())
		}
	}
}

func (s *redisService) syncWithDB(ctx context.Context, change CreditChange) error {
	defer func(start time.Time) {
		metrics.UsagelimiterReplayLatency.Observe(time.Since(start).Seconds())
	}(time.Now())

	ctx, span := tracing.Start(ctx, "usagelimiter.syncWithDB")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.dbCircuitBreaker.Do(ctx, func(innerCtx context.Context) (any, error) {
		innerCtx, cancel := context.WithTimeout(innerCtx, 2*time.Second)
		defer cancel()

		err := db.Query.UpdateKeyCredits(innerCtx, s.db.RW(), db.UpdateKeyCreditsParams{
			ID:        change.KeyID,
			Operation: "decrement",
			Credits:   sql.NullInt32{Int32: change.Amount, Valid: true},
		})

		return nil, err
	})

	if err != nil {
		tracing.RecordError(span, err)
		metrics.UsagelimiterReplayOperations.WithLabelValues("error").Inc()
		return err
	}

	metrics.UsagelimiterReplayOperations.WithLabelValues("success").Inc()
	return nil
}
