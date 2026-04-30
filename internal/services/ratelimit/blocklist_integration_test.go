package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"

	rldb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
)

// blocklistTestEnv bundles a per-test MySQL container plus both a pkg/db
// database (for handing to [ratelimit.New] under the [DB] interface) and a
// wrapped ratelimit DB (for direct query assertions inside the tests). Each
// test gets independent service instances against the same data plane —
// the multi-region scenario we are asserting.
type blocklistTestEnv struct {
	t    *testing.T
	db   DB
	rldb *rldb.Database
}

func newBlocklistTestEnv(t *testing.T) *blocklistTestEnv {
	t.Helper()

	cfg := dockertest.MySQL(t)
	database, err := pkgdb.New(pkgdb.Config{
		PrimaryDSN:  cfg.DSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	return &blocklistTestEnv{
		t:    t,
		db:   database,
		rldb: rldb.New(database.RW(), database.RO()),
	}
}

// newRegion builds a ratelimit service wired into the shared DB. Each "region"
// in the test is one of these: same DB, independent in-memory state, separate
// sync goroutine. The Counter is per-region so we exercise the propagation
// channel rather than letting Redis fan-out hide the bug we are looking for.
func (e *blocklistTestEnv) newRegion(clk clock.Clock) *service {
	e.t.Helper()
	svc, err := New(Config{
		Clock:   clk,
		Counter: counter.NewMemory(),
		DB:      e.db,
	})
	require.NoError(e.t, err)
	e.t.Cleanup(func() { _ = svc.Close() })
	return svc
}

// waitForRow blocks until the propagation row for (workspace, namespace,
// identifier, duration) is visible in MySQL. The batch processor flushes every
// second, so polling for up to 3s gives deterministic propagation in tests
// without coupling to internal timing knobs.
func (e *blocklistTestEnv) waitForRow(workspaceID, namespace, identifier string, durationMs int64) {
	e.t.Helper()
	require.Eventually(e.t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		rows, err := e.rldb.RO().BlocklistListActive(ctx, 0)
		if err != nil {
			return false
		}
		for _, r := range rows {
			if r.WorkspaceID == workspaceID && r.Namespace == namespace &&
				r.Identifier == identifier && int64(r.DurationMs) == durationMs {
				return true
			}
		}
		return false
	}, 3*time.Second, 50*time.Millisecond, "expected propagation row in MySQL")
}

// TestBlocklist_PropagatesDenialAcrossRegions is the headline scenario: a
// denial in one region must shut down the same identifier in a different
// region without the second region having seen the abusive traffic firsthand.
//
// The proof is end-to-end: take a request that *would have passed* if region B
// were judging on its own local counters, and show it is denied after region A
// finishes propagating.
func TestBlocklist_PropagatesDenialAcrossRegions(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	clk := clock.NewTestClock()
	regionA := env.newRegion(clk)
	regionB := env.newRegion(clk)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-rotating"
		limit       = int64(2)
	)
	duration := time.Minute

	ctx := context.Background()
	mkReq := func() RatelimitRequest {
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

	// Region A: burn through the limit, then trip a denial. The denial flushes
	// a propagation row to MySQL.
	for i := int64(0); i < limit; i++ {
		resp, err := regionA.Ratelimit(ctx, mkReq())
		require.NoError(t, err)
		require.True(t, resp.Success, "request %d to region A should pass", i+1)
	}
	resp, err := regionA.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	require.False(t, resp.Success, "limit+1 in region A should deny")

	// Wait for the batch processor to land the row. This is the only place we
	// wait on async timing; everything else in the test is deterministic.
	env.waitForRow(workspaceID, namespace, identifier, duration.Milliseconds())

	// Region B has seen zero traffic. Without propagation, its local counter
	// for this identifier is 0 and the next request would pass. Force-sync
	// from MySQL and the next request must instead deny — that is the whole
	// feature.
	regionB.runBlocklistSyncOnce()

	resp, err = regionB.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	require.False(t, resp.Success,
		"region B should deny after sync — propagation is doing nothing if this passes")
}

// TestBlocklist_DedupsConcurrentDenialsToOneRow asserts that sustained abuse
// against the same identifier within a single window collapses to one MySQL
// row across the fleet. Without this the propagation channel would amplify
// abuse traffic into a write storm.
func TestBlocklist_DedupsConcurrentDenialsToOneRow(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	clk := clock.NewTestClock()
	regionA := env.newRegion(clk)
	regionB := env.newRegion(clk)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-burst"
		limit       = int64(1)
	)
	duration := time.Minute

	ctx := context.Background()
	mkReq := func() RatelimitRequest {
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

	// Both regions independently burn the limit and start denying. Each region
	// sees its denial as the "first" locally, but the second writer's row must
	// collapse onto the first via ON DUPLICATE KEY UPDATE.
	_, err := regionA.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	_, err = regionA.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	_, err = regionB.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	_, err = regionB.Ratelimit(ctx, mkReq())
	require.NoError(t, err)

	env.waitForRow(workspaceID, namespace, identifier, duration.Milliseconds())

	// More denials in the same window — strictUntil is already in the future
	// in both regions, so setStrictUntil should NOT enqueue more events. The
	// row count must stay at exactly one. require.Never polls past the
	// flush interval (1s + slack) so even if extra events were buffered,
	// we'd see the row count grow.
	for range 5 {
		_, _ = regionA.Ratelimit(ctx, mkReq())
		_, _ = regionB.Ratelimit(ctx, mkReq())
	}

	countMatching := func() int {
		rows, err := env.rldb.RO().BlocklistListActive(context.Background(), 0)
		if err != nil {
			return -1
		}
		n := 0
		for _, r := range rows {
			if r.WorkspaceID == workspaceID && r.Namespace == namespace && r.Identifier == identifier {
				n++
			}
		}
		return n
	}

	require.Never(t, func() bool {
		return countMatching() > 1
	}, 2*time.Second, 100*time.Millisecond,
		"sustained denial across multiple regions must collapse to one row")
	require.Equal(t, 1, countMatching(), "expected exactly one row to remain after dedup")
}

// TestBlocklist_SyncInflatedDenialsDontEcho asserts that once a region
// receives a propagation row and its sync inflates the local counter,
// subsequent denials caused by that inflation do NOT emit echo events back
// to MySQL. Without this gate, every region that sees traffic for an
// already-blocked identifier would write an extra row per window —
// turning each denial into an N-region write storm.
func TestBlocklist_SyncInflatedDenialsDontEcho(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	clk := clock.NewTestClock()
	regionA := env.newRegion(clk)
	regionB := env.newRegion(clk)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-noecho"
		limit       = int64(1)
	)
	duration := time.Minute

	ctx := context.Background()
	mkReq := func() RatelimitRequest {
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

	// Region A originates the denial and writes the row.
	_, err := regionA.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	_, err = regionA.Ratelimit(ctx, mkReq())
	require.NoError(t, err)

	env.waitForRow(workspaceID, namespace, identifier, duration.Milliseconds())

	originalExpires := func() uint64 {
		rows, listErr := env.rldb.RO().BlocklistListActive(context.Background(), 0)
		require.NoError(t, listErr)
		for _, r := range rows {
			if r.WorkspaceID == workspaceID && r.Namespace == namespace && r.Identifier == identifier {
				return r.ExpiresAt
			}
		}
		t.Fatalf("expected row to exist after waitForRow")
		return 0
	}()

	// Region B syncs from MySQL — its counter is now sync-inflated and the
	// blocked flag is set, so its denials below should be silent from the
	// propagation channel's perspective.
	regionB.runBlocklistSyncOnce()

	// Drive lots of traffic into B. Each request hits a sync-inflated
	// counter and denies, but every CAS on blocked must fail because sync
	// already flipped it to true.
	for range 10 {
		resp, ratelimitErr := regionB.Ratelimit(ctx, mkReq())
		require.NoError(t, ratelimitErr)
		require.False(t, resp.Success, "B should deny on sync-inflated counter")
	}

	// If B had emitted any events, the row's expires_at would have moved
	// forward via GREATEST. Poll past one full flush interval to catch any
	// delayed write.
	require.Never(t, func() bool {
		rows, listErr := env.rldb.RO().BlocklistListActive(context.Background(), 0)
		if listErr != nil {
			return false
		}
		for _, r := range rows {
			if r.WorkspaceID == workspaceID && r.Namespace == namespace && r.Identifier == identifier {
				return r.ExpiresAt != originalExpires
			}
		}
		return false
	}, 2*time.Second, 100*time.Millisecond,
		"sync-inflated region must not echo write — expires_at must not move from %d", originalExpires)
}

// TestBlocklist_StoredSequenceWinsOverClock asserts the sync inflates the
// counter for the originating denial's sequence, not whichever sequence the
// receiving region's clock currently maps to.
//
// We pin the row's sequence to a value far behind the region's current
// sequence. If the implementation recomputed sequence from its own clock it
// would inflate the current sequence's counter — over-blocking by many
// full windows. Storing sequence means we inflate the recorded sequence,
// which is far in the past and therefore harmless.
//
// expires_at is set to far future explicitly so the test asserts only the
// "which sequence gets inflated" property, independent of cleanup timing.
func TestBlocklist_StoredSequenceWinsOverClock(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	clk := clock.NewTestClock()
	region := env.newRegion(clk)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-frozen"
		limit       = int64(10)
		durationMs  = int64(60_000)
	)

	currentSequence := clk.Now().UnixMilli() / durationMs
	originalSequence := currentSequence - 5
	farFutureExpiresAt := clk.Now().Add(24 * time.Hour).UnixMilli()

	require.NoError(t, env.rldb.RW().BlocklistInsert(context.Background(), rldb.BlocklistInsertParams{
		WorkspaceID: workspaceID,
		Namespace:   namespace,
		Identifier:  identifier,
		DurationMs:  uint64(durationMs),
		Sequence:    originalSequence,
		Limit:       uint64(limit),
		ExpiresAt:   uint64(farFutureExpiresAt),
	}))

	region.runBlocklistSyncOnce()

	// Sync must have inflated counter at originalSequence, not the current
	// sequence. If the implementation recomputed from clock, we'd see the
	// inflation on the current sequence and the next request would deny.
	require.NotEqual(t, originalSequence, currentSequence,
		"test setup invariant: clock must be on a different sequence")

	originalKey := counterKey{
		workspaceID: workspaceID,
		namespace:   namespace,
		identifier:  identifier,
		durationMs:  durationMs,
		sequence:    originalSequence,
	}
	currentKey := counterKey{
		workspaceID: workspaceID,
		namespace:   namespace,
		identifier:  identifier,
		durationMs:  durationMs,
		sequence:    currentSequence,
	}

	originalEntry, ok := region.counters.Load(originalKey)
	require.True(t, ok, "sync should have created an entry at the originating sequence")
	require.Equal(t, limit, originalEntry.(*counterEntry).val.Load(),
		"originating sequence's counter should be inflated to limit")

	_, currentExists := region.counters.Load(currentKey)
	require.False(t, currentExists,
		"sync must NOT have created an entry at the receiving region's current sequence")
}
