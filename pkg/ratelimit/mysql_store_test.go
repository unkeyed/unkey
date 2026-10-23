package ratelimit

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	shareddb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestMySQLStoreRoundTrip(t *testing.T) {
	ctx := t.Context()
	store := newMySQLTestStore(t)

	fingerprint := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	older := StoredSession{Fingerprint: fingerprint, SessionID: uid.New("", 16), AnchorAtNs: 100, ConsumedNs: 7, CompactAfterMs: 1}
	newer := StoredSession{Fingerprint: fingerprint, SessionID: uid.New("", 16), AnchorAtNs: 200, ConsumedNs: 5, CompactAfterMs: 2}
	require.NoError(t, store.WriteSessions(ctx, []StoredSession{older, newer}))
	stale := older
	stale.ConsumedNs = 3
	require.NoError(t, store.WriteSessions(ctx, []StoredSession{stale}))

	sessions, err := store.ReadSessions(ctx, fingerprint, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredSession{newer, older}, sessions)

	sessions, err = store.ReadSessions(ctx, fingerprint, 1)
	require.NoError(t, err)
	require.Equal(t, []StoredSession{newer}, sessions)
}

// TestMySQLStoreCleanup guarantees the two promises that cleanup relies on:
// due rows page in `ListRowsToCompact` order, and a cleanup whose rows
// changed after it read them changes nothing.
func TestMySQLStoreCleanup(t *testing.T) {
	ctx := t.Context()
	store := newMySQLTestStore(t)

	// The database outlives test runs, so each run lists its own range.
	dueMs := time.Now().UnixNano()
	beforeDue := DueRow{Fingerprint: lastFingerprint, CompactAfterMs: dueMs - 1}
	first := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	second := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	if bytes.Compare(first[:], second[:]) > 0 {
		first, second = second, first
	}
	a := StoredSession{Fingerprint: second, SessionID: uid.New("", 16), AnchorAtNs: 100, ConsumedNs: 10, CompactAfterMs: dueMs}
	b := StoredSession{Fingerprint: second, SessionID: uid.New("", 16), AnchorAtNs: 105, ConsumedNs: 5, CompactAfterMs: dueMs}
	c := StoredSession{Fingerprint: first, SessionID: uid.New("", 16), AnchorAtNs: 100, ConsumedNs: 1, CompactAfterMs: dueMs}
	require.NoError(t, store.WriteSessions(ctx, []StoredSession{a, b, c}))

	page, err := store.ListDueRows(ctx, dueMs+1, beforeDue, 1)
	require.NoError(t, err)
	require.Equal(t, []DueRow{{Fingerprint: first, CompactAfterMs: dueMs}}, page)
	page, err = store.ListDueRows(ctx, dueMs+1, page[0], 3)
	require.NoError(t, err)
	require.Equal(t, []DueRow{{Fingerprint: second, CompactAfterMs: dueMs}, {Fingerprint: second, CompactAfterMs: dueMs}}, page)
	page, err = store.ListDueRows(ctx, dueMs+1, page[1], 3)
	require.NoError(t, err)
	require.Empty(t, page, "the next page starts after the last fingerprint")
	page, err = store.ListDueRows(ctx, dueMs, beforeDue, 2)
	require.NoError(t, err)
	require.Empty(t, page)

	merged := StoredSession{Fingerprint: second, SessionID: uid.New("", 16), AnchorAtNs: 100, ConsumedNs: 15, CompactAfterMs: dueMs + 1}
	stale := b
	stale.ConsumedNs = 4
	applied, err := store.ApplyCleanup(ctx, []StoredSession{a, stale}, []StoredSession{merged})
	require.NoError(t, err)
	require.False(t, applied)
	sessions, err := store.ReadSessionsFromPrimary(ctx, second, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredSession{b, a}, sessions, "a failed delete must roll back the earlier delete")

	applied, err = store.ApplyCleanup(ctx, []StoredSession{a, b}, []StoredSession{merged})
	require.NoError(t, err)
	require.True(t, applied)
	sessions, err = store.ReadSessionsFromPrimary(ctx, second, 10)
	require.NoError(t, err)
	require.Equal(t, []StoredSession{merged}, sessions)
}

// newMySQLTestStore returns a store on the test container through `pkg/db`,
// as the API builds it.
func newMySQLTestStore(t *testing.T) Store {
	t.Helper()
	database, err := shareddb.New(shareddb.Config{PrimaryDSN: containers.MySQL(t).DSN, ReadOnlyDSN: "", Tags: sqlcomment.Static{}})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return NewMySQLStore(database.RW(), database.RO())
}
