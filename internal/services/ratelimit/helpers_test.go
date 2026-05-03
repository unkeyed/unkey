package ratelimit

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

// newTestDB returns a [mysql.MySQL] handle for the shared docker-compose
// MySQL (managed externally via `make up`). Connection is closed on
// t.Cleanup. Used by unit tests that need to satisfy [Config.DB] but don't
// exercise the blocklist itself.
func newTestDB(t testing.TB) DB {
	t.Helper()

	cfg := containers.MySQL(t)
	database, err := mysql.New(mysql.Config{
		PrimaryDSN:  cfg.FormatDSN(),
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	return database
}

// trackingCounter wraps an underlying counter.Counter and records how many
// times Get and Increment are invoked. All other operations pass through
// unchanged. Safe for concurrent use.
type trackingCounter struct {
	counter.Counter
	getCalls  atomic.Int64
	incrCalls atomic.Int64
}

func newTrackingCounter() *trackingCounter {
	return &trackingCounter{Counter: counter.NewMemory()}
}

func (c *trackingCounter) Get(ctx context.Context, key string) (int64, error) {
	c.getCalls.Add(1)
	return c.Counter.Get(ctx, key)
}

func (c *trackingCounter) Increment(ctx context.Context, key string, value int64, ttl ...time.Duration) (int64, error) {
	c.incrCalls.Add(1)
	return c.Counter.Increment(ctx, key, value, ttl...)
}

// failingCounter wraps an underlying counter.Counter but returns a fixed error
// from Get and Increment, while counting how many times each was invoked.
// Other operations fall through to the embedded counter. Safe for concurrent use.
type failingCounter struct {
	counter.Counter
	err       error
	getCalls  atomic.Int64
	incrCalls atomic.Int64
}

func newFailingCounter(err error) *failingCounter {
	return &failingCounter{Counter: counter.NewMemory(), err: err}
}

func (c *failingCounter) Get(_ context.Context, _ string) (int64, error) {
	c.getCalls.Add(1)
	return 0, c.err
}

func (c *failingCounter) Increment(_ context.Context, _ string, _ int64, _ ...time.Duration) (int64, error) {
	c.incrCalls.Add(1)
	return 0, c.err
}
