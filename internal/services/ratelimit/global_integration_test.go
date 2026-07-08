package ratelimit

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	rldb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	unkeymysql "github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/uid"
)

// listAllRows returns every row in ratelimit_global_counters
// regardless of region or expiry. Tests bypass the production query
// (which aggregates and filters out the caller's own region) so they can
// assert which region wrote what.
func (e *integrationTestEnv) listAllRows() []rldb.GlobalCountersListAllRow {
	e.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rows, err := e.rldb.RO().GlobalCountersListAll(ctx)
	require.NoError(e.t, err)
	return rows
}

// findRow returns the row for (key, region) if present, ok=false
// otherwise. Used by both positive and negative assertions.
func (e *integrationTestEnv) findRow(workspaceID, namespace, identifier, region string, durationMs int64) (rldb.GlobalCountersListAllRow, bool) {
	e.t.Helper()
	for _, r := range e.listAllRows() {
		if r.WorkspaceID == workspaceID && r.Namespace == namespace &&
			r.Identifier == identifier && int64(r.DurationMs) == durationMs &&
			r.Region == region {
			return r, true
		}
	}
	return rldb.GlobalCountersListAllRow{}, false
}

// waitForRow blocks until a row from `region` for the matching
// window cell is visible in MySQL. Manual runGlobalPushOnce calls
// in tests are synchronous, so the row is normally visible immediately;
// the polling loop covers the case where the periodic push goroutine
// (running on its own jittered cadence) is the writer. Polling, never
// goroutine-ticker timing.
func (e *integrationTestEnv) waitForRow(workspaceID, namespace, identifier, region string, durationMs int64) rldb.GlobalCountersListAllRow {
	e.t.Helper()
	var found rldb.GlobalCountersListAllRow
	require.Eventually(e.t, func() bool {
		row, ok := e.findRow(workspaceID, namespace, identifier, region, durationMs)
		if !ok {
			return false
		}
		found = row
		return true
	}, 3*time.Second, 50*time.Millisecond, "expected global-counters row from region %q", region)
	return found
}

// hasRow reports whether any row exists for the (key, region)
// tuple. Used by negative tests that assert a flush did NOT happen.
func (e *integrationTestEnv) hasRow(workspaceID, namespace, identifier, region string, durationMs int64) bool {
	e.t.Helper()
	_, ok := e.findRow(workspaceID, namespace, identifier, region, durationMs)
	return ok
}

// TestGlobal_PropagatesCountAcrossRegions is the headline scenario:
// region A consumes part of the limit and flushes its count; region B
// reads the row and folds it into its sliding-window math. A request to
// B that would have passed against B's local-only count is now denied
// because the cross-region picture says the user is already over. A
// never denies in this scenario; the value of sharing is that B sees A's
// count even though no individual region has crossed the limit.
func TestGlobal_PropagatesCountAcrossRegions(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionA := env.newRegionAs(clk, "region-a")
	regionB := env.newRegionAs(clk, "region-b")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-shared"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		}
	}

	// Region A consumes 6 of 10 — past the 0.2 utilization floor so flush
	// is eligible. None of these denies; A is well within its limit on its
	// own, but its consumption is now relevant to other regions.
	for range 6 {
		resp, err := regionA.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	// Trigger flush deterministically rather than waiting for the 5s
	// jittered ticker. The bulk upsert inside runGlobalPushOnce is
	// synchronous, so by the time it returns the row is in MySQL.
	regionA.runGlobalPushOnce()
	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(6), row.Count, "region A's row should reflect its 6 consumed tokens")

	// Region B has seen no traffic. Without count sharing, B would happily
	// allow up to its own limit. Pull from MySQL: B.globalCount should now
	// reflect A's contribution.
	regionB.runGlobalPullOnce()

	// One more cost-1 request to B: effective = 0 (B's val) + 6 (B's globalCount
	// from A) + 0 (prev) + 1 (cost) = 7, still under 10, so it passes.
	resp, err := regionB.Ratelimit(ctx, makeReq())
	require.NoError(t, err)
	require.True(t, resp.Success, "B should allow first request when combined effective is 7/10")

	// After that one passing request, B's val=1 and globalCount=6. A request
	// of cost 4 would push effective to 1+4+6=11 > 10: deny. This is the
	// behavior the cross-region sharing buys us.
	denyReq := makeReq()
	denyReq.Cost = 4
	resp, err = regionB.Ratelimit(ctx, denyReq)
	require.NoError(t, err)
	require.False(t, resp.Success, "B must deny when combined effective exceeds limit; without sharing this would have passed")
}

// TestGlobalPush_EmitsRowsInUniqueKeyOrder guarantees that a flush batch
// follows the ratelimit_global_counters unique key order before reaching MySQL.
// Concurrent upserts that visit the same rows in different orders can deadlock.
func TestGlobalPush_EmitsRowsInUniqueKeyOrder(t *testing.T) {
	t.Parallel()

	keys := []counterKey{
		{workspaceID: "ws-c", namespace: "ns-b", identifier: "id-b", durationMs: 60_000, sequence: 3},
		{workspaceID: "ws-a", namespace: "ns-c", identifier: "id-a", durationMs: 60_000, sequence: 2},
		{workspaceID: "ws-b", namespace: "ns-a", identifier: "id-d", durationMs: 30_000, sequence: 5},
		{workspaceID: "ws-a", namespace: "ns-a", identifier: "id-c", durationMs: 120_000, sequence: 1},
		{workspaceID: "ws-d", namespace: "ns-b", identifier: "id-a", durationMs: 30_000, sequence: 4},
		{workspaceID: "ws-b", namespace: "ns-c", identifier: "id-c", durationMs: 60_000, sequence: 2},
		{workspaceID: "ws-c", namespace: "ns-a", identifier: "id-a", durationMs: 120_000, sequence: 3},
		{workspaceID: "ws-a", namespace: "ns-b", identifier: "id-d", durationMs: 30_000, sequence: 5},
	}

	for attempt := range 20 {
		recorder := &recordingGlobalCounterDB{}
		svc := &service{ //nolint:exhaustruct // test only needs global push dependencies
			clock:                clock.NewTestClock(),
			region:               "region-a",
			db:                   rldb.New(recorder, recorder),
			globalCircuitBreaker: circuitbreaker.New[any]("ratelimit_global_push_order_test"),
		}

		for _, key := range keys {
			entry := &counterEntry{} //nolint:exhaustruct // only push eligibility fields matter here
			entry.val.Store(10)
			entry.globalPushThreshold.Store(1)
			svc.counters.Store(key, entry)
		}

		svc.runGlobalPushOnce()

		require.Len(t, recorder.rows, len(keys))
		require.True(t,
			globalCounterRowsInUniqueKeyOrder(recorder.rows),
			"attempt %d emitted global counter rows out of unique-key order: %v",
			attempt,
			globalCounterRowKeys(recorder.rows),
		)
	}
}

// TestGlobalPush_ConcurrentFlushesDoNotDeadlock reproduces the production lock
// graph with real MySQL transactions. The DB wrapper splits each collected bulk
// batch into per-row upserts inside one transaction, which turns inconsistent
// row order into the same 1213 deadlock MySQL can produce under overlapping
// ON DUPLICATE KEY UPDATE flushes.
func TestGlobalPush_ConcurrentFlushesDoNotDeadlock(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)

	for attempt := range 30 {
		workspaceID := uid.New(uid.WorkspacePrefix)
		keys := []counterKey{
			{workspaceID: workspaceID, namespace: "ns-a", identifier: "id-a", durationMs: 60_000, sequence: 1},
			{workspaceID: workspaceID, namespace: "ns-a", identifier: "id-b", durationMs: 60_000, sequence: 1},
			{workspaceID: workspaceID, namespace: "ns-b", identifier: "id-a", durationMs: 30_000, sequence: 2},
			{workspaceID: workspaceID, namespace: "ns-b", identifier: "id-b", durationMs: 30_000, sequence: 2},
		}
		seedRows := make([]rldb.UpsertRatelimitGlobalCountersParams, 0, len(keys))
		for _, key := range keys {
			seedRows = append(seedRows, globalCounterRowForKey(key, "region-a", 1))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		require.NoError(t, env.rldb.BulkUpsertGlobalCounters(ctx, seedRows))

		db := &transactionalSplitBulkDB{
			rw:      env.db.RW(),
			start:   make(chan struct{}),
			delay:   50 * time.Millisecond,
			timeout: 2 * time.Second,
		}

		svcA := newGlobalPushOnlyService(db, "region-a")
		for _, key := range keys {
			storePushableCounter(svcA, key)
		}

		svcB := newGlobalPushOnlyService(db, "region-a")
		for i := len(keys) - 1; i >= 0; i-- {
			storePushableCounter(svcB, keys[i])
		}

		var wg sync.WaitGroup
		wg.Go(svcA.runGlobalPushOnce)
		wg.Go(svcB.runGlobalPushOnce)
		wg.Wait()
		cancel()

		require.Empty(t, db.deadlockErrors(),
			"attempt %d reproduced a MySQL deadlock during concurrent global counter flushes; row orders: %v",
			attempt,
			db.rowOrders(),
		)
	}
}

func newGlobalPushOnlyService(db rldb.DBTX, region string) *service {
	return &service{ //nolint:exhaustruct // test only needs global push dependencies
		clock:                clock.NewTestClock(),
		region:               region,
		db:                   rldb.New(db, db),
		globalCircuitBreaker: circuitbreaker.New[any]("ratelimit_global_push_deadlock_test"),
	}
}

func storePushableCounter(svc *service, key counterKey) {
	entry := &counterEntry{} //nolint:exhaustruct // only push eligibility fields matter here
	entry.val.Store(10)
	entry.globalPushThreshold.Store(1)
	svc.counters.Store(key, entry)
}

func globalCounterRowForKey(key counterKey, region string, count uint64) rldb.UpsertRatelimitGlobalCountersParams {
	return rldb.UpsertRatelimitGlobalCountersParams{
		WorkspaceID: key.workspaceID,
		Namespace:   key.namespace,
		Identifier:  key.identifier,
		DurationMs:  uint64(key.durationMs),
		Sequence:    key.sequence,
		Region:      region,
		Count:       count,
		ExpiresAt:   uint64((key.sequence + 2) * key.durationMs),
		UpdatedAt:   uint64(time.Now().UnixMilli()),
	}
}

type transactionalSplitBulkDB struct {
	rw      *unkeymysql.Replica
	start   chan struct{}
	delay   time.Duration
	timeout time.Duration

	mu      sync.Mutex
	started int
	errors  []error
	orders  [][]string
}

func (db *transactionalSplitBulkDB) ExecContext(ctx context.Context, _ string, args ...interface{}) (sql.Result, error) {
	rows, err := globalCounterRowsFromArgs(args)
	if err != nil {
		return nil, err
	}
	db.recordOrder(rows)
	if err := db.waitForPeer(ctx); err != nil {
		return nil, err
	}

	txCtx, cancel := context.WithTimeout(ctx, db.timeout)
	defer cancel()

	tx, err := db.rw.Begin(txCtx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	for i, row := range rows {
		err = (&rldb.BulkQueries{}).UpsertRatelimitGlobalCounters(txCtx, tx, []rldb.UpsertRatelimitGlobalCountersParams{row})
		if err != nil {
			db.recordError(err)
			return nil, err
		}
		if i < len(rows)-1 {
			// Hold each row lock briefly so the peer transaction has time to
			// reach its next upsert and expose inverted lock ordering.
			time.Sleep(db.delay)
		}
	}

	if err = tx.Commit(); err != nil {
		db.recordError(err)
		return nil, err
	}
	return nil, nil
}

func (db *transactionalSplitBulkDB) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, fmt.Errorf("unexpected PrepareContext")
}

func (db *transactionalSplitBulkDB) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected QueryContext")
}

func (db *transactionalSplitBulkDB) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return &sql.Row{}
}

func (db *transactionalSplitBulkDB) waitForPeer(ctx context.Context) error {
	db.mu.Lock()
	db.started++
	if db.started == 2 {
		close(db.start)
	}
	db.mu.Unlock()

	select {
	case <-db.start:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (db *transactionalSplitBulkDB) recordError(err error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.errors = append(db.errors, err)
}

func (db *transactionalSplitBulkDB) recordOrder(rows []rldb.UpsertRatelimitGlobalCountersParams) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.orders = append(db.orders, globalCounterRowKeys(rows))
}

func (db *transactionalSplitBulkDB) rowOrders() [][]string {
	db.mu.Lock()
	defer db.mu.Unlock()
	orders := make([][]string, len(db.orders))
	copy(orders, db.orders)
	return orders
}

func (db *transactionalSplitBulkDB) deadlockErrors() []error {
	db.mu.Lock()
	defer db.mu.Unlock()

	deadlocks := make([]error, 0, len(db.errors))
	for _, err := range db.errors {
		if unkeymysql.IsDeadlockError(err) {
			deadlocks = append(deadlocks, err)
		}
	}
	return deadlocks
}

type recordingGlobalCounterDB struct {
	rows []rldb.UpsertRatelimitGlobalCountersParams
}

func (db *recordingGlobalCounterDB) ExecContext(_ context.Context, _ string, args ...interface{}) (sql.Result, error) {
	const paramsPerRow = 9
	if len(args)%paramsPerRow != 0 {
		return nil, fmt.Errorf("expected args to be divisible by %d, got %d", paramsPerRow, len(args))
	}

	for i := 0; i < len(args); i += paramsPerRow {
		row, err := globalCounterRowFromArgs(args[i : i+paramsPerRow])
		if err != nil {
			return nil, err
		}
		db.rows = append(db.rows, row)
	}

	return nil, nil
}

func (db *recordingGlobalCounterDB) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, fmt.Errorf("unexpected PrepareContext")
}

func (db *recordingGlobalCounterDB) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected QueryContext")
}

func (db *recordingGlobalCounterDB) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return &sql.Row{}
}

func globalCounterRowsFromArgs(args []interface{}) ([]rldb.UpsertRatelimitGlobalCountersParams, error) {
	const paramsPerRow = 9
	if len(args)%paramsPerRow != 0 {
		return nil, fmt.Errorf("expected args to be divisible by %d, got %d", paramsPerRow, len(args))
	}

	rows := make([]rldb.UpsertRatelimitGlobalCountersParams, 0, len(args)/paramsPerRow)
	for i := 0; i < len(args); i += paramsPerRow {
		row, err := globalCounterRowFromArgs(args[i : i+paramsPerRow])
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func globalCounterRowFromArgs(args []interface{}) (rldb.UpsertRatelimitGlobalCountersParams, error) {
	row := rldb.UpsertRatelimitGlobalCountersParams{}
	var ok bool

	if row.WorkspaceID, ok = args[0].(string); !ok {
		return row, fmt.Errorf("workspace_id arg has type %T", args[0])
	}
	if row.Namespace, ok = args[1].(string); !ok {
		return row, fmt.Errorf("namespace arg has type %T", args[1])
	}
	if row.Identifier, ok = args[2].(string); !ok {
		return row, fmt.Errorf("identifier arg has type %T", args[2])
	}
	if row.DurationMs, ok = args[3].(uint64); !ok {
		return row, fmt.Errorf("duration_ms arg has type %T", args[3])
	}
	if row.Sequence, ok = args[4].(int64); !ok {
		return row, fmt.Errorf("sequence arg has type %T", args[4])
	}
	if row.Region, ok = args[5].(string); !ok {
		return row, fmt.Errorf("region arg has type %T", args[5])
	}
	if row.Count, ok = args[6].(uint64); !ok {
		return row, fmt.Errorf("count arg has type %T", args[6])
	}
	if row.ExpiresAt, ok = args[7].(uint64); !ok {
		return row, fmt.Errorf("expires_at arg has type %T", args[7])
	}
	if row.UpdatedAt, ok = args[8].(uint64); !ok {
		return row, fmt.Errorf("updated_at arg has type %T", args[8])
	}

	return row, nil
}

func globalCounterRowsInUniqueKeyOrder(rows []rldb.UpsertRatelimitGlobalCountersParams) bool {
	return slices.IsSortedFunc(rows, compareGlobalPushRows) || len(rows) < 2
}

func globalCounterRowKeys(rows []rldb.UpsertRatelimitGlobalCountersParams) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, fmt.Sprintf(
			"%s/%s/%s/%d/%d/%s",
			row.WorkspaceID,
			row.Namespace,
			row.Identifier,
			row.DurationMs,
			row.Sequence,
			row.Region,
		))
	}
	return keys
}

func TestGlobal_RealWorldTwoRegionsWithTwoNodesEach(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionAOrigin := counter.NewMemory()
	regionBOrigin := counter.NewMemory()

	regionA1 := env.newRegionWithCounter(clk, "region-a", regionAOrigin)
	regionA2 := env.newRegionWithCounter(clk, "region-a", regionAOrigin)
	regionB1 := env.newRegionWithCounter(clk, "region-b", regionBOrigin)
	regionB2 := env.newRegionWithCounter(clk, "region-b", regionBOrigin)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-real-world"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID,
			Namespace:   namespace,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    duration,
			Cost:        1,
			Time:        clk.Now(),
		}
	}
	curKey := counterKey{
		workspaceID: workspaceID,
		namespace:   namespace,
		identifier:  identifier,
		durationMs:  duration.Milliseconds(),
		sequence:    calculateSequence(clk.Now(), duration),
	}

	// Region A receives traffic on two nodes backed by the same regional origin
	// counter. This models two API instances sharing one regional Redis while
	// other regions have independent Redis state.
	for range 3 {
		resp, err := regionA1.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	for range 3 {
		resp, err := regionA2.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	waitForLocalCount := func(regions []*service, expected int64) *service {
		t.Helper()
		var selected *service
		require.Eventually(t, func() bool {
			for _, region := range regions {
				entry, ok := region.counters.Load(curKey)
				if ok && entry.(*counterEntry).val.Load() >= expected {
					selected = region
					return true
				}
			}
			return false
		}, 3*time.Second, 50*time.Millisecond, "expected regional origin replay to converge local count to %d", expected)
		if selected == nil {
			t.Fatalf("expected a selected region after convergence")
		}
		return selected
	}

	// Regional replay should converge at least one A node to the region's total
	// count, and the global counter row should publish that converged value.
	waitForLocalCount([]*service{regionA1, regionA2}, 6)
	regionA1.runGlobalPushOnce()
	regionA2.runGlobalPushOnce()
	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(6), row.Count)

	// Both B nodes import region A's contribution. They still use their own
	// regional origin for B-local traffic; MySQL only carries cross-region state.
	regionB1.runGlobalPullOnce()
	regionB2.runGlobalPullOnce()
	for _, region := range []*service{regionB1, regionB2} {
		entryValue, ok := region.counters.Load(curKey)
		require.True(t, ok)
		require.Equal(t, int64(6), entryValue.(*counterEntry).globalCount.Load())
	}

	for range 2 {
		resp, err := regionB1.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	for range 2 {
		resp, err := regionB2.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	// Once B's regional origin converges its two nodes' local traffic to 4, any
	// further cost-1 request in B must deny: local 4 + imported 6 + cost 1 = 11.
	denyRegion := waitForLocalCount([]*service{regionB1, regionB2}, 4)
	resp, err := denyRegion.Ratelimit(ctx, makeReq())
	require.NoError(t, err)
	require.False(t, resp.Success,
		"region B must deny after combining its own regional count with region A's imported count")
}

// TestGlobal_BelowUtilizationFloorDoesNotPush asserts the write-side
// utilization filter that gates emission. An entry whose val/limit is below
// the floor cannot meaningfully push another region over its limit, so
// flushing it would be wasted MySQL load. The cursed scenario it prevents
// is "cost > limit on a fresh window" — the user has consumed nothing in
// this window but would otherwise be propagated to all other regions.
func TestGlobal_BelowUtilizationFloorDoesNotPush(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-low-util"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	// Consume 1 of 10 — strictly below the 0.2 floor (threshold=2).
	resp, err := region.Ratelimit(ctx, RatelimitRequest{
		WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
		Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
	})
	require.NoError(t, err)
	require.True(t, resp.Success)

	region.runGlobalPushOnce()

	// The flush filter must have skipped this entry. require.Never polls
	// to guard against a periodic flush from the background goroutine
	// firing concurrently.
	require.Never(t, func() bool {
		return env.hasRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	}, 2*time.Second, 100*time.Millisecond,
		"sub-floor utilization must not write a global-counters row")
}

// TestGlobal_PushUsesConvergedLocalCount asserts that push eligibility is
// based on the live local counter, not a request-path latch. Two instances in
// the same region can each stay below the utilization floor locally while
// Redis convergence raises one instance's val above the floor; that converged
// value must still flush cross-region.
func TestGlobal_PushUsesConvergedLocalCount(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	sharedCounter := counter.NewMemory()
	regionA1, err := New(Config{
		Clock:   clk,
		Counter: sharedCounter,
		DB:      env.db,
		Region:  "region-a",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = regionA1.Close() })
	regionA2, err := New(Config{
		Clock:   clk,
		Counter: sharedCounter,
		DB:      env.db,
		Region:  "region-a",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = regionA2.Close() })

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-converged"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID,
			Namespace:   namespace,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    duration,
			Cost:        1,
			Time:        clk.Now(),
		}
	}

	// Each instance accepts 1/10 locally, below the 0.2 utilization floor
	// (threshold=2). Their shared Redis origin converges the region total to 2/10,
	// which crosses the floor and triggers the push.
	resp, reqErr := regionA1.Ratelimit(ctx, makeReq())
	require.NoError(t, reqErr)
	require.True(t, resp.Success)

	resp, reqErr = regionA2.Ratelimit(ctx, makeReq())
	require.NoError(t, reqErr)
	require.True(t, resp.Success)

	curKey := counterKey{
		workspaceID: workspaceID,
		namespace:   namespace,
		identifier:  identifier,
		durationMs:  duration.Milliseconds(),
		sequence:    calculateSequence(clk.Now(), duration),
	}
	require.Eventually(t, func() bool {
		for _, region := range []*service{regionA1, regionA2} {
			entry, ok := region.counters.Load(curKey)
			if ok && entry.(*counterEntry).val.Load() >= 2 {
				return true
			}
		}
		return false
	}, 3*time.Second, 50*time.Millisecond, "expected Redis replay to converge local count above the push floor")

	regionA1.runGlobalPushOnce()
	regionA2.runGlobalPushOnce()

	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(2), row.Count, "push must use the converged local count")
}

// TestGlobal_PushIgnoresSpeculativeBatchIncrements covers the small window in
// [Service.RatelimitMany] where a batch has added its optimistic increments to
// the local counters but has not yet decided whether the full batch commits.
// Cross-region push must subtract those speculative increments so a batch that
// later rolls back cannot publish temporary state to MySQL.
//
// The test drives the speculative field directly rather than racing a live
// RatelimitMany call against the background pusher. The invariant under test is
// the push-side read of val minus speculative, not scheduler timing.
func TestGlobal_PushIgnoresSpeculativeBatchIncrements(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-speculative"
		limit       = int64(10)
	)
	duration := time.Minute

	resp, err := region.Ratelimit(context.Background(), RatelimitRequest{
		WorkspaceID: workspaceID,
		Namespace:   namespace,
		Identifier:  identifier,
		Limit:       limit,
		Duration:    duration,
		Cost:        6,
		Time:        clk.Now(),
	})
	require.NoError(t, err)
	require.True(t, resp.Success)

	curKey := counterKey{
		workspaceID: workspaceID,
		namespace:   namespace,
		identifier:  identifier,
		durationMs:  duration.Milliseconds(),
		sequence:    calculateSequence(clk.Now(), duration),
	}
	entryValue, ok := region.counters.Load(curKey)
	require.True(t, ok)
	entry := entryValue.(*counterEntry)

	entry.speculative.Add(6)
	region.runGlobalPushOnce()
	require.False(t, env.hasRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds()),
		"in-flight RatelimitMany increments must not be pushed")

	entry.speculative.Add(-6)
	region.runGlobalPushOnce()
	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(6), row.Count)
}

// TestGlobal_AtFloorPushes is the inverse of the previous test: at
// or above the floor, the entry must flush. Tests the boundary specifically
// to guard against an off-by-one in the comparison.
func TestGlobal_AtFloorPushes(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-at-floor"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	// Consume exactly 2 of 10 — at the 0.2 floor (threshold=2).
	for range 2 {
		resp, err := region.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	region.runGlobalPushOnce()

	row := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(2), row.Count, "row count must reflect the at-floor utilization")
}

// TestGlobal_RemoteEmitForcesLocalSubFloorPush covers ENG-2903: skewed multi-region
// traffic: once one region has published a global counter for a cell, another
// region must publish any non-zero local contribution for the same cell instead
// of waiting for its own utilization floor.
func TestGlobal_RemoteEmitForcesLocalSubFloorPush(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionA := env.newRegionAs(clk, "region-a")
	regionB := env.newRegionAs(clk, "region-b")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-skewed"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID,
			Namespace:   namespace,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    duration,
			Cost:        1,
			Time:        clk.Now(),
		}
	}

	// A crosses the normal floor and emits. B imports that row before it sees
	// enough local traffic to cross its own floor.
	for range 6 {
		resp, err := regionA.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	regionA.runGlobalPushOnce()
	env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())

	regionB.runGlobalPullOnce()
	regionB.runGlobalPushOnce()
	require.False(t, env.hasRow(workspaceID, namespace, identifier, "region-b", duration.Milliseconds()),
		"import alone must not emit a zero-local row")

	resp, err := regionB.Ratelimit(ctx, makeReq())
	require.NoError(t, err)
	require.True(t, resp.Success)

	regionB.runGlobalPushOnce()
	row := env.waitForRow(workspaceID, namespace, identifier, "region-b", duration.Milliseconds())
	require.Equal(t, uint64(1), row.Count,
		"remote emission must force B to publish below its normal utilization floor")
}

// TestGlobal_SyncKeepsOwnRegionOutOfGlobalCount asserts that a region's own
// MySQL row merges into regional val, not globalCount. Folding own traffic into
// globalCount would double-count it because sliding-window math already adds
// val and globalCount together.
func TestGlobal_SyncKeepsOwnRegionOutOfGlobalCount(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-self-only"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	// Push past floor so a row gets written.
	for range 6 {
		resp, err := region.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	region.runGlobalPushOnce()
	env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())

	// Sync from this same region. Own-region count may raise val as a safety net,
	// but globalCount must stay foreign-only.
	region.runGlobalPullOnce()

	curKey := counterKey{
		workspaceID: workspaceID, namespace: namespace, identifier: identifier,
		durationMs: duration.Milliseconds(),
		sequence:   calculateSequence(clk.Now(), duration),
	}
	entry, ok := region.counters.Load(curKey)
	require.True(t, ok)
	counter := entry.(*counterEntry)
	require.Equal(t, int64(6), counter.val.Load(),
		"sync should merge own-region contribution into regional val")
	require.Equal(t, int64(0), counter.globalCount.Load(),
		"sync must not import the region's own contribution into globalCount")
}

// TestGlobal_OwnRegionImportIsRegionalSafetyNet models two instances tagged as
// the same region that have not converged through Redis. If one has pushed a
// higher regional observation to MySQL, the other may use that row as a
// monotonic lower bound for its own regional val. This is only a safety net:
// it must not mark the entry fresh against the regional origin.
func TestGlobal_OwnRegionImportIsRegionalSafetyNet(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionA1 := env.newRegionAs(clk, "region-a")
	regionA2 := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-own-region-import"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID,
			Namespace:   namespace,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    duration,
			Cost:        1,
			Time:        clk.Now(),
		}
	}

	for range 6 {
		resp, err := regionA1.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	regionA1.runGlobalPushOnce()
	env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())

	regionA2.runGlobalPullOnce()

	curKey := counterKey{
		workspaceID: workspaceID,
		namespace:   namespace,
		identifier:  identifier,
		durationMs:  duration.Milliseconds(),
		sequence:    calculateSequence(clk.Now(), duration),
	}
	entry, ok := regionA2.counters.Load(curKey)
	require.True(t, ok, "own-region import should create the local entry")
	counter := entry.(*counterEntry)
	require.Equal(t, int64(6), counter.val.Load())
	require.Equal(t, int64(0), counter.globalCount.Load())
	require.Equal(t, int64(0), counter.originFreshUntilMs.Load(),
		"MySQL safety-net import must not mark regional origin freshness")

	denyReq := makeReq()
	denyReq.Cost = 5
	resp, err := regionA2.Ratelimit(ctx, denyReq)
	require.NoError(t, err)
	require.False(t, resp.Success,
		"own-region import should participate in local sliding-window decisions")
}

// TestGlobal_SumsAcrossMultipleRegions asserts that a third region
// folding in contributions from two others sums them rather than taking
// max or replacing. This is the load-bearing G-Counter property: each
// region's row holds its own contribution; aggregation across rows is sum.
func TestGlobal_SumsAcrossMultipleRegions(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionA := env.newRegionAs(clk, "region-a")
	regionB := env.newRegionAs(clk, "region-b")
	regionC := env.newRegionAs(clk, "region-c")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-multi"
		limit       = int64(20)
	)
	duration := time.Minute
	ctx := context.Background()

	makeReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		}
	}

	// A consumes 12 (above floor=4), B consumes 11 (above floor=4).
	// Total true global at this point is 23.
	for range 12 {
		_, err := regionA.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
	}
	for range 11 {
		_, err := regionB.Ratelimit(ctx, makeReq())
		require.NoError(t, err)
	}

	regionA.runGlobalPushOnce()
	regionB.runGlobalPushOnce()
	env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	env.waitForRow(workspaceID, namespace, identifier, "region-b", duration.Milliseconds())

	// Region C has seen nothing locally. After sync, its globalCount must
	// equal the sum of A's 12 and B's 11. Max would give 12 (under-count),
	// replace would give whichever arrived last.
	regionC.runGlobalPullOnce()

	curKey := counterKey{
		workspaceID: workspaceID, namespace: namespace, identifier: identifier,
		durationMs: duration.Milliseconds(),
		sequence:   calculateSequence(clk.Now(), duration),
	}
	entry, ok := regionC.counters.Load(curKey)
	require.True(t, ok, "sync must create the entry on C even with no local traffic")
	require.Equal(t, int64(23), entry.(*counterEntry).globalCount.Load(),
		"region C must see SUM(12+11)=23, not max or last-write-wins")

	// Concretely: any cost-1 request on C must now deny because effective
	// is already 23 > 20.
	resp, err := regionC.Ratelimit(ctx, makeReq())
	require.NoError(t, err)
	require.False(t, resp.Success,
		"C must deny when globalCount sum already exceeds the limit")
}

// TestGlobal_ChangeFilterAvoidsRedundantWrites asserts that a flush
// run twice with no intervening traffic does not produce a second write.
// The change filter (skip if val <= lastPushed) is the dominant cost
// reduction in steady state where most active windows are quiet between
// flushes.
func TestGlobal_ChangeFilterAvoidsRedundantWrites(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-quiet"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	// Push to 6/10 so the first flush emits.
	for range 6 {
		resp, err := region.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	region.runGlobalPushOnce()
	first := env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.Equal(t, uint64(6), first.Count)

	// Advance the test clock so a second flush, if it emits, writes a
	// distinguishably newer updated_at. Without this advance, the second
	// upsert (if it happens) would carry the same updated_at as the
	// first, and the assertion below would pass trivially.
	clk.Tick(time.Second)

	// Trigger another flush with no intervening traffic. The filter must
	// recognize val unchanged and skip the upsert. The bulk upsert is
	// synchronous, so by the time runGlobalPushOnce returns the
	// row is either updated or it isn't — no waiting needed.
	region.runGlobalPushOnce()

	second, ok := env.findRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	require.True(t, ok)
	require.Equal(t, first.UpdatedAt, second.UpdatedAt,
		"change filter must skip the second flush; a re-emitted upsert would advance updated_at")
}

// TestGlobal_DoesNotPropagateColdOversizedRequest asserts that a single
// oversized request denied locally does not propagate to other regions.
// A user has consumed zero tokens this window; the deny path doesn't
// bump val, so val stays 0 and the utilization filter excludes the entry
// from the flush. Other regions see no row and do not over-block this
// user on their next sync.
func TestGlobal_DoesNotPropagateColdOversizedRequest(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	region := env.newRegionAs(clk, "region-a")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-oversized"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	resp, err := region.Ratelimit(ctx, RatelimitRequest{
		WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
		Limit: limit, Duration: duration, Cost: limit + 1, Time: clk.Now(),
	})
	require.NoError(t, err)
	require.False(t, resp.Success, "cost > limit must deny locally")

	region.runGlobalPushOnce()

	// The cold-window denial increments nothing (deny path doesn't bump
	// val), so val stays 0 and the utilization filter skips the flush.
	require.Never(t, func() bool {
		return env.hasRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())
	}, 2*time.Second, 100*time.Millisecond,
		"cold oversized denial must not write a global-counters row")
}

// TestGlobal_EntriesCreatedOnSync asserts that the pull goroutine
// materializes a local counterEntry when it sees a row for a key the
// region has not seen any traffic for. The new entry's globalCount
// carries the remote sum; subsequent local requests on this identifier
// deny correctly even though the region's val starts at zero.
func TestGlobal_EntriesCreatedOnSync(t *testing.T) {
	t.Parallel()

	env := newIntegrationTestEnv(t)
	clk := clock.NewTestClock()
	regionA := env.newRegionAs(clk, "region-a")
	regionB := env.newRegionAs(clk, "region-b")

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-cold-import"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	for range 6 {
		_, err := regionA.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
	}
	regionA.runGlobalPushOnce()
	env.waitForRow(workspaceID, namespace, identifier, "region-a", duration.Milliseconds())

	// Region B has no local entry for this key. Sync must materialize one
	// with globalCount=6.
	regionB.runGlobalPullOnce()

	curKey := counterKey{
		workspaceID: workspaceID, namespace: namespace, identifier: identifier,
		durationMs: duration.Milliseconds(),
		sequence:   calculateSequence(clk.Now(), duration),
	}
	entry, ok := regionB.counters.Load(curKey)
	require.True(t, ok, "sync must create the entry on B even with no local traffic")
	require.Equal(t, int64(6), entry.(*counterEntry).globalCount.Load())
}
