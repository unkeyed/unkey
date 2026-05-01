package ratelimit

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/testutil/containers"

	rldb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
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

// integrationTestEnv bundles a per-test MySQL container plus both a
// pkg/mysql database (for handing to [ratelimit.New] under the [DB]
// interface) and a wrapped ratelimit DB (for direct query assertions).
// Each test gets independent service instances against the same data
// plane; that's the multi-region scenario the integration tests assert.
//
// Uses dockertest.MySQL rather than containers.MySQL so each test gets
// its own isolated table state. The integration tests assert on row
// counts and table contents, which would race under a shared database.
type integrationTestEnv struct {
	t    *testing.T
	db   DB
	rldb *rldb.Database
}

func newIntegrationTestEnv(t *testing.T) *integrationTestEnv {
	t.Helper()

	cfg := dockertest.MySQL(t)
	database, err := mysql.New(mysql.Config{
		PrimaryDSN:  cfg.DSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	return &integrationTestEnv{
		t:    t,
		db:   database,
		rldb: rldb.New(database.RW(), database.RO()),
	}
}

// newRegion builds a ratelimit service tagged with the default test
// region. Single-region scenarios use this.
func (e *integrationTestEnv) newRegion(clk clock.Clock) *service {
	return e.newRegionAs(clk, "test-region")
}

// newRegionAs builds a ratelimit service tagged with regionTag. Multi-
// region tests use distinct tags so each region's window-counts rows live
// in their own (workspace, ..., region) bucket and the sync loop's
// region != self predicate makes each region see the others' contributions.
func (e *integrationTestEnv) newRegionAs(clk clock.Clock, regionTag string) *service {
	e.t.Helper()
	svc, err := New(Config{
		Clock:   clk,
		Counter: counter.NewMemory(),
		DB:      e.db,
		Region:  regionTag,
	})
	require.NoError(e.t, err)
	e.t.Cleanup(func() { _ = svc.Close() })
	return svc
}
