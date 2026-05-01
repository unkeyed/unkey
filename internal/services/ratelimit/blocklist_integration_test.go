package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/mysql"

	rldb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
)

// blocklistTestEnv bundles a per-test MySQL container plus both a pkg/mysql
// database (for handing to [ratelimit.New] under the [DB] interface) and a
// wrapped ratelimit DB (for direct query assertions inside the tests). Each
// test gets independent service instances against the same data plane;
// that's the multi-region scenario we are asserting.
//
// Uses dockertest.MySQL rather than containers.MySQL so each test gets its
// own isolated table state. The integration tests assert on row counts and
// table contents, which would race under a shared database.
type blocklistTestEnv struct {
	t    *testing.T
	db   DB
	rldb *rldb.Database
}

func newBlocklistTestEnv(t *testing.T) *blocklistTestEnv {
	t.Helper()

	cfg := dockertest.MySQL(t)
	database, err := mysql.New(mysql.Config{
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

	require.NoError(t, env.rldb.BulkInsertBlocklist(context.Background(), []rldb.BlocklistInsertParams{{
		WorkspaceID: workspaceID,
		Namespace:   namespace,
		Identifier:  identifier,
		DurationMs:  uint64(durationMs),
		Sequence:    originalSequence,
		Limit:       uint64(limit),
		ExpiresAt:   uint64(farFutureExpiresAt),
	}}))

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

// TestBlocklist_DecaysAcrossWindowBoundary asserts that a propagated denial
// bleeds into the next window via sliding-window math and is gone the window
// after that. This is the central behavioral claim of the design and is
// independent of the propagation channel: once region B has the inflated
// counter at sequence S, the receiver's own sliding-window math takes over.
//
// The clock is pinned to the exact start of a window so windowElapsed is
// deterministic at each assertion point. Limit is 10 so the decay shape is
// visible: prev contributes limit*(1-elapsed), which goes from limit at
// elapsed=0 down to ~0 at elapsed=1.
func TestBlocklist_DecaysAcrossWindowBoundary(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	duration := time.Minute
	durationMs := duration.Milliseconds()
	// Snap to the start of the current window so elapsed math is exact at
	// every assertion. Using the wall clock keeps the test from depending on
	// any specific epoch.
	windowStart := time.UnixMilli((time.Now().UnixMilli() / durationMs) * durationMs)
	clk := clock.NewTestClock(windowStart)

	regionA := env.newRegion(clk)
	regionB := env.newRegion(clk)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-decay"
		limit       = int64(10)
	)

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

	// Region A burns through the limit and trips a denial. Row written.
	for i := int64(0); i < limit; i++ {
		resp, err := regionA.Ratelimit(ctx, mkReq())
		require.NoError(t, err)
		require.True(t, resp.Success, "A request %d/%d should pass", i+1, limit)
	}
	resp, err := regionA.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	require.False(t, resp.Success, "A's limit+1 request should deny")

	env.waitForRow(workspaceID, namespace, identifier, durationMs)
	regionB.runBlocklistSyncOnce()

	// Sequence S (current window): B's cur is sync-inflated to limit. A
	// fresh request adds cost=1, tripping the limit.
	resp, err = regionB.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	require.False(t, resp.Success,
		"B at sequence S should deny — sync inflated cur to limit")

	// Start of S+1 (elapsed=0). The inflated entry at S becomes the prev
	// counter for new requests. Sliding-window math: cur=0 + prev*(1-0) + cost
	// = 0 + 10 + 1 = 11 > 10 → still deny. Without bleed-into-next-window the
	// receiver would let traffic through the moment the window flips.
	clk.Set(windowStart.Add(duration))
	resp, err = regionB.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	require.False(t, resp.Success,
		"B at start of S+1 should still deny — prev=limit contributes 100%% via sliding window")

	// Mid S+1 (elapsed=0.5). prev's contribution decays to limit*0.5 = 5.
	// 0 + 5 + 1 = 6 ≤ 10 → allow. This is the decay smoking gun.
	clk.Set(windowStart.Add(duration + duration/2))
	resp, err = regionB.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	require.True(t, resp.Success,
		"B mid S+1 should allow — prev's contribution decays to limit/2")

	// Start of S+2. The inflated counter at S is no longer in the
	// sliding-window math at all (we look at cur=S+2 and prev=S+1, not S).
	// prev=S+1 contains the single allowed request from the previous step
	// (val=1), so effective = 0 + 1 + 1 = 2 ≤ 10 → allow.
	clk.Set(windowStart.Add(2 * duration))
	resp, err = regionB.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	require.True(t, resp.Success,
		"B at S+2 should allow — inflated counter at S is no longer in the sliding window")
}

// TestBlocklist_CleanupDeletesExpiredOnly asserts that BlocklistDeleteExpired
// removes only rows past the cutoff, preserves rows at exactly the cutoff
// (the query uses strict `<`, not `<=`), and reports the correct delete count.
//
// The Restate cron drives this query in production. If it gets the boundary
// wrong, the propagation channel either retains stale denials past their
// useful lifetime (read amplification) or evicts still-active rows (denials
// drop out of sync before the originating window decays). Both are silent
// failures; this test catches them.
func TestBlocklist_CleanupDeletesExpiredOnly(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		durationMs  = uint64(60_000)
		limit       = uint64(10)
	)

	// Three rows differing only by identifier (so the unique key doesn't
	// collapse them) and expires_at: one before cutoff, one exactly at
	// cutoff, one after.
	const cutoff uint64 = 200_000
	rows := []rldb.BlocklistInsertParams{
		{WorkspaceID: workspaceID, Namespace: namespace, Identifier: "expired", DurationMs: durationMs, Sequence: 1, Limit: limit, ExpiresAt: cutoff - 1},
		{WorkspaceID: workspaceID, Namespace: namespace, Identifier: "boundary", DurationMs: durationMs, Sequence: 1, Limit: limit, ExpiresAt: cutoff},
		{WorkspaceID: workspaceID, Namespace: namespace, Identifier: "future", DurationMs: durationMs, Sequence: 1, Limit: limit, ExpiresAt: cutoff + 1},
	}
	require.NoError(t, env.rldb.BulkInsertBlocklist(context.Background(), rows))

	deleted, err := env.rldb.RW().BlocklistDeleteExpired(context.Background(), cutoff)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted, "only the row strictly before cutoff should be deleted")

	// Surviving rows: list everything with expires_at > 0 (i.e., everything
	// still in the table). Asserts the boundary row is preserved.
	remaining, err := env.rldb.RO().BlocklistListActive(context.Background(), 0)
	require.NoError(t, err)
	require.Len(t, remaining, 2, "boundary and future rows must survive the delete")

	identifiers := make(map[string]uint64, len(remaining))
	for _, r := range remaining {
		identifiers[r.Identifier] = r.ExpiresAt
	}
	require.Equal(t, cutoff, identifiers["boundary"],
		"row at exactly cutoff must remain — query is strict `<`, not `<=`")
	require.Equal(t, cutoff+1, identifiers["future"],
		"row past cutoff must remain")
	_, expiredStillThere := identifiers["expired"]
	require.False(t, expiredStillThere, "row before cutoff must be gone")
}

// TestBlocklist_IsolatesAcrossWorkspaces asserts that two workspaces using
// the same namespace, identifier, and duration do not pollute each other
// through propagation rows or sync inflation. The unique index includes
// workspace_id and the in-memory counter keys do too; this test makes sure
// neither layer collapses across workspace boundaries.
//
// Without this isolation, a noisy customer in workspace A could starve a
// well-behaved customer in workspace B that happens to use the same
// identifier string ("user_42", "pro-tier", whatever).
func TestBlocklist_IsolatesAcrossWorkspaces(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	clk := clock.NewTestClock()
	region := env.newRegion(clk)

	const (
		workspaceA = "ws_a"
		workspaceB = "ws_b"
		namespace  = "ns"
		identifier = "shared"
		limit      = int64(2)
	)
	duration := time.Minute

	ctx := context.Background()
	mkReq := func(ws string) RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: ws,
			Namespace:   namespace,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    duration,
			Cost:        1,
			Time:        clk.Now(),
		}
	}

	// Workspace A burns through its limit and denies — row written for A only.
	for i := int64(0); i < limit; i++ {
		resp, err := region.Ratelimit(ctx, mkReq(workspaceA))
		require.NoError(t, err)
		require.True(t, resp.Success, "A request %d/%d should pass", i+1, limit)
	}
	resp, err := region.Ratelimit(ctx, mkReq(workspaceA))
	require.NoError(t, err)
	require.False(t, resp.Success, "A's limit+1 should deny")

	env.waitForRow(workspaceA, namespace, identifier, duration.Milliseconds())

	// Verify exactly one row exists, and it belongs to workspace A.
	rows, err := env.rldb.RO().BlocklistListActive(ctx, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1, "only workspace A should have written a propagation row")
	require.Equal(t, workspaceA, rows[0].WorkspaceID)

	// Workspace B has not seen any traffic yet. Its first request must pass
	// even though A is currently being blocked on the same identifier — B's
	// counter is independent.
	resp, err = region.Ratelimit(ctx, mkReq(workspaceB))
	require.NoError(t, err)
	require.True(t, resp.Success,
		"workspace B should not be affected by A's denial on the same identifier")

	// Force a sync. If the implementation collapsed the workspace dimension
	// in counter keys, A's inflated counter would poison B. Drive enough
	// traffic into B to fully exercise its limit; nothing here should deny
	// because B's counter is independent.
	region.runBlocklistSyncOnce()

	// B has already used 1 of its limit above, so it should still allow
	// limit-1 more requests.
	for i := int64(0); i < limit-1; i++ {
		resp, err = region.Ratelimit(ctx, mkReq(workspaceB))
		require.NoError(t, err)
		require.True(t, resp.Success,
			"workspace B request %d should pass — B's counter is independent of A's", i+2)
	}
}

// TestBlocklist_DifferentDurationsAreSeparateRows asserts that the same
// identifier hitting two policies with different windows produces two
// distinct propagation rows. duration_ms is part of the unique key precisely
// so a customer running both a 1m and a 5m limit on the same identifier sees
// independent enforcement on each.
//
// If duration_ms were collapsed out of the unique key, one policy's denial
// would clobber the other's row via ON DUPLICATE KEY UPDATE and receivers
// would only inflate one of the two counters.
func TestBlocklist_DifferentDurationsAreSeparateRows(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	clk := clock.NewTestClock()
	region := env.newRegion(clk)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "shared"
		limit       = int64(2)
	)
	shortDuration := time.Minute
	longDuration := 2 * time.Minute

	ctx := context.Background()
	mkReq := func(d time.Duration) RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID,
			Namespace:   namespace,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    d,
			Cost:        1,
			Time:        clk.Now(),
		}
	}

	// Trip a denial under the short-window policy.
	for i := int64(0); i < limit; i++ {
		resp, err := region.Ratelimit(ctx, mkReq(shortDuration))
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	resp, err := region.Ratelimit(ctx, mkReq(shortDuration))
	require.NoError(t, err)
	require.False(t, resp.Success, "short-duration limit+1 should deny")

	// Trip a denial under the long-window policy. Counter is independent —
	// fresh limit available.
	for i := int64(0); i < limit; i++ {
		resp, err := region.Ratelimit(ctx, mkReq(longDuration))
		require.NoError(t, err)
		require.True(t, resp.Success,
			"long-duration request %d/%d should pass — counter is independent of short-duration's", i+1, limit)
	}
	resp, err = region.Ratelimit(ctx, mkReq(longDuration))
	require.NoError(t, err)
	require.False(t, resp.Success, "long-duration limit+1 should deny")

	env.waitForRow(workspaceID, namespace, identifier, shortDuration.Milliseconds())
	env.waitForRow(workspaceID, namespace, identifier, longDuration.Milliseconds())

	// Both rows must exist independently. Filter by identifier (other rows
	// could exist from background noise; in this isolated container there's
	// nothing else, but assert by attribute rather than by total count).
	rows, err := env.rldb.RO().BlocklistListActive(ctx, 0)
	require.NoError(t, err)

	durations := make(map[int64]bool)
	for _, r := range rows {
		if r.WorkspaceID == workspaceID && r.Namespace == namespace && r.Identifier == identifier {
			durations[int64(r.DurationMs)] = true
		}
	}
	require.True(t, durations[shortDuration.Milliseconds()],
		"row for short duration must exist")
	require.True(t, durations[longDuration.Milliseconds()],
		"row for long duration must exist — without duration_ms in the unique key, this would have been clobbered")
	require.Len(t, durations, 2, "exactly two rows for this identifier — one per policy")
}

// TestBlocklist_PropagatesFromRatelimitMany asserts that a denial via the
// batch path also writes a propagation row, and that in a mixed batch only
// the failing entry emits. Ratelimit and RatelimitMany reach denial through
// different control flow (single CAS loop vs optimistic-Add-then-rollback),
// but both must call activateStrictMode for failed entries — and only failed
// entries. A refactor of the batch rollback path could silently disable
// propagation, or worse, emit for entries that actually passed.
//
// The failing entry crosses the 50% usage threshold so the propagation
// filter doesn't suppress its emit (see activateStrictMode for the rule).
func TestBlocklist_PropagatesFromRatelimitMany(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	clk := clock.NewTestClock()
	region := env.newRegion(clk)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		passingID   = "user-passing"
		failingID   = "user-failing"
		limit       = int64(10)
	)
	duration := time.Minute
	ctx := context.Background()

	// Pre-fill the failing identifier to 6/10 so the eventual denial's
	// preCost (6) is at or above limit/2 (5) and the propagation filter
	// allows the emit through.
	for range 6 {
		resp, err := region.Ratelimit(ctx, RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: failingID,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		})
		require.NoError(t, err)
		require.True(t, resp.Success)
	}

	// Mixed batch: one entry well under its limit (passing), one whose
	// post-Add count crosses the limit (failing). RatelimitMany rolls back
	// both counter increments (all-or-nothing side effects) but per-entry
	// responses report individual pass/fail. activateStrictMode fires for
	// the failing entry only.
	reqs := []RatelimitRequest{
		{WorkspaceID: workspaceID, Namespace: namespace, Identifier: passingID, Limit: limit, Duration: duration, Cost: 1, Time: clk.Now()},
		{WorkspaceID: workspaceID, Namespace: namespace, Identifier: failingID, Limit: limit, Duration: duration, Cost: 5, Time: clk.Now()},
	}

	resps, err := region.RatelimitMany(ctx, reqs)
	require.NoError(t, err)
	require.Len(t, resps, 2)
	require.True(t, resps[0].Success, "passing entry reports individual pass even when batch rolls back")
	require.False(t, resps[1].Success, "failing entry exceeds its limit (6+5 > 10)")

	// Wait for the failing entry's row to land — that's the smoke test.
	env.waitForRow(workspaceID, namespace, failingID, duration.Milliseconds())

	// Assert no spurious row for the passing entry. Both inserts would have
	// hit the same flush, so once the failing row is visible we have the
	// definitive picture; require.Never adds a margin in case of any delayed
	// write.
	require.Never(t, func() bool {
		rows, listErr := env.rldb.RO().BlocklistListActive(ctx, 0)
		if listErr != nil {
			return false
		}
		for _, r := range rows {
			if r.Identifier == passingID {
				return true
			}
		}
		return false
	}, 2*time.Second, 100*time.Millisecond,
		"batch entry that satisfied its limit must not emit a propagation row")
}

// TestBlocklist_DoesNotPropagateBelowFiftyPercentUsage asserts the propagation
// filter that skips emits when the user has consumed less than half their
// limit at the time of denial. The classic case: a single oversized request
// (cost > limit) on a fresh user. Locally we deny correctly, but propagating
// would block other regions for someone who still has full legitimate quota.
func TestBlocklist_DoesNotPropagateBelowFiftyPercentUsage(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	clk := clock.NewTestClock()
	region := env.newRegion(clk)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-oversized"
		limit       = int64(10)
	)
	duration := time.Minute

	resp, err := region.Ratelimit(context.Background(), RatelimitRequest{
		WorkspaceID: workspaceID,
		Namespace:   namespace,
		Identifier:  identifier,
		Limit:       limit,
		Duration:    duration,
		Cost:        limit + 1,
		Time:        clk.Now(),
	})
	require.NoError(t, err)
	require.False(t, resp.Success, "cost > limit must deny locally")

	// preCost is 0 at the time of denial; 0 is below limit/2, so the filter
	// suppresses the emit. Watch past one full flush interval to be sure.
	require.Never(t, func() bool {
		rows, listErr := env.rldb.RO().BlocklistListActive(context.Background(), 0)
		if listErr != nil {
			return false
		}
		for _, r := range rows {
			if r.Identifier == identifier {
				return true
			}
		}
		return false
	}, 2*time.Second, 100*time.Millisecond,
		"denial below 50%% usage must not write a propagation row")
}

// TestBlocklist_DoesNotPropagateForSubMinuteDurations asserts the duration
// filter that skips emits for windows shorter than one minute. Below that,
// the 1s flush + 10s sync pipeline takes longer than the window itself, so
// the row arrives at receivers after the originating window has rotated and
// has no enforcement value.
func TestBlocklist_DoesNotPropagateForSubMinuteDurations(t *testing.T) {
	t.Parallel()

	env := newBlocklistTestEnv(t)

	clk := clock.NewTestClock()
	region := env.newRegion(clk)

	const (
		workspaceID = "ws_test"
		namespace   = "ns"
		identifier  = "user-burst"
		limit       = int64(2)
	)
	// 30s is shorter than the minPropagationDuration floor.
	duration := 30 * time.Second
	ctx := context.Background()

	mkReq := func() RatelimitRequest {
		return RatelimitRequest{
			WorkspaceID: workspaceID, Namespace: namespace, Identifier: identifier,
			Limit: limit, Duration: duration, Cost: 1, Time: clk.Now(),
		}
	}

	for i := int64(0); i < limit; i++ {
		resp, err := region.Ratelimit(ctx, mkReq())
		require.NoError(t, err)
		require.True(t, resp.Success)
	}
	resp, err := region.Ratelimit(ctx, mkReq())
	require.NoError(t, err)
	require.False(t, resp.Success, "limit+1 must deny locally regardless of duration")

	// Even though preCost=limit (would normally propagate), the duration
	// filter blocks the emit because the row would be useless by the time
	// receivers see it.
	require.Never(t, func() bool {
		rows, listErr := env.rldb.RO().BlocklistListActive(ctx, 0)
		if listErr != nil {
			return false
		}
		for _, r := range rows {
			if r.Identifier == identifier {
				return true
			}
		}
		return false
	}, 2*time.Second, 100*time.Millisecond,
		"sub-minute window must not write a propagation row")
}
