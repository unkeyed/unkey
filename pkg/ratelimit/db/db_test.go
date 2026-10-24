package db_test

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"math/rand/v2"
	"slices"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ratelimit/db"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func openMySQL(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("mysql", containers.MySQL(t).DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	return conn
}

// newFingerprints returns n random fingerprints in ascending byte order, the
// order in which MySQL sorts binary columns.
func newFingerprints(t *testing.T, n int) [][]byte {
	t.Helper()
	fingerprints := make([][]byte, n)
	for i := range fingerprints {
		hash := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
		fingerprints[i] = hash[:]
	}
	slices.SortFunc(fingerprints, bytes.Compare)
	return fingerprints
}

func listSessions(t *testing.T, conn db.DBTX, fingerprint []byte, limit int32) []db.ListSessionsRow {
	t.Helper()
	sessions, err := db.Query.ListSessions(t.Context(), conn, db.ListSessionsParams{
		Fingerprint: fingerprint,
		Limit:       limit,
	})
	require.NoError(t, err)
	return sessions
}

// TestUpsertSession_KeepsGreatestConsumedAndFirstValues guarantees that a
// retried or reordered write cannot move consumed_ns backwards, and that no
// write changes anchor_at_ns or compact_after_ms after the row exists.
func TestUpsertSession_KeepsGreatestConsumedAndFirstValues(t *testing.T) {
	t.Parallel()
	conn := openMySQL(t)
	ctx := t.Context()
	fingerprint := newFingerprints(t, 1)[0]
	sessionID := uid.New(uid.TestPrefix)

	err := db.BulkQuery.UpsertSession(ctx, conn, []db.UpsertSessionParams{
		{Fingerprint: fingerprint, SessionID: sessionID, AnchorAtNs: 50, ConsumedNs: 8, CompactAfterMs: 100},
	})
	require.NoError(t, err)
	err = db.Query.UpsertSession(ctx, conn, db.UpsertSessionParams{
		Fingerprint: fingerprint, SessionID: sessionID, AnchorAtNs: 90, ConsumedNs: 7, CompactAfterMs: 200,
	})
	require.NoError(t, err)
	err = db.Query.UpsertSession(ctx, conn, db.UpsertSessionParams{
		Fingerprint: fingerprint, SessionID: sessionID, AnchorAtNs: 10, ConsumedNs: 10, CompactAfterMs: 50,
	})
	require.NoError(t, err)

	require.Equal(t, []db.ListSessionsRow{
		{SessionID: sessionID, AnchorAtNs: 50, ConsumedNs: 10, CompactAfterMs: 100},
	}, listSessions(t, conn, fingerprint, 10))
}

// TestListSessions_ReturnsNewestFirstWithinLimit guarantees that a caller gets
// the newest sessions of one fingerprint first, and that a result cut off by
// the LIMIT has exactly LIMIT rows, so the caller can detect it.
func TestListSessions_ReturnsNewestFirstWithinLimit(t *testing.T) {
	t.Parallel()
	conn := openMySQL(t)
	ctx := t.Context()
	fingerprints := newFingerprints(t, 2)
	fingerprint, other := fingerprints[0], fingerprints[1]
	oldest, middle, newest := uid.New(uid.TestPrefix), uid.New(uid.TestPrefix), uid.New(uid.TestPrefix)

	err := db.BulkQuery.UpsertSession(ctx, conn, []db.UpsertSessionParams{
		{Fingerprint: fingerprint, SessionID: middle, AnchorAtNs: 20, ConsumedNs: 2, CompactAfterMs: 1},
		{Fingerprint: fingerprint, SessionID: newest, AnchorAtNs: 30, ConsumedNs: 3, CompactAfterMs: 1},
		{Fingerprint: fingerprint, SessionID: oldest, AnchorAtNs: 10, ConsumedNs: 1, CompactAfterMs: 1},
		{Fingerprint: other, SessionID: uid.New(uid.TestPrefix), AnchorAtNs: 40, ConsumedNs: 4, CompactAfterMs: 1},
	})
	require.NoError(t, err)

	require.Equal(t, []db.ListSessionsRow{
		{SessionID: newest, AnchorAtNs: 30, ConsumedNs: 3, CompactAfterMs: 1},
		{SessionID: middle, AnchorAtNs: 20, ConsumedNs: 2, CompactAfterMs: 1},
		{SessionID: oldest, AnchorAtNs: 10, ConsumedNs: 1, CompactAfterMs: 1},
	}, listSessions(t, conn, fingerprint, 10))

	require.Equal(t, []db.ListSessionsRow{
		{SessionID: newest, AnchorAtNs: 30, ConsumedNs: 3, CompactAfterMs: 1},
		{SessionID: middle, AnchorAtNs: 20, ConsumedNs: 2, CompactAfterMs: 1},
	}, listSessions(t, conn, fingerprint, 2))

	require.Empty(t, listSessions(t, conn, newFingerprints(t, 1)[0], 10))
}

// TestListRowsToCompact_PagesThroughEveryDueFingerprint guarantees that
// paging with the cursor skips no fingerprint whose compact_after_ms passed,
// also when several fingerprints share a value and a page ends between them.
func TestListRowsToCompact_PagesThroughEveryDueFingerprint(t *testing.T) {
	t.Parallel()
	conn := openMySQL(t)
	ctx := t.Context()
	compactAfterMsBase := rand.Int64N(1<<40) + 1<<41
	fingerprints := newFingerprints(t, 4)
	a, b, c, d := fingerprints[0], fingerprints[1], fingerprints[2], fingerprints[3]

	rows := []db.UpsertSessionParams{
		{Fingerprint: a, SessionID: uid.New(uid.TestPrefix), CompactAfterMs: compactAfterMsBase + 100},
		{Fingerprint: a, SessionID: uid.New(uid.TestPrefix), CompactAfterMs: compactAfterMsBase + 100},
		{Fingerprint: b, SessionID: uid.New(uid.TestPrefix), CompactAfterMs: compactAfterMsBase + 100},
		{Fingerprint: c, SessionID: uid.New(uid.TestPrefix), CompactAfterMs: compactAfterMsBase + 100},
		{Fingerprint: a, SessionID: uid.New(uid.TestPrefix), CompactAfterMs: compactAfterMsBase + 200},
		{Fingerprint: d, SessionID: uid.New(uid.TestPrefix), CompactAfterMs: compactAfterMsBase + 300},
	}
	for i := range rows {
		rows[i].AnchorAtNs = 1
		rows[i].ConsumedNs = 1
	}
	require.NoError(t, db.BulkQuery.UpsertSession(ctx, conn, rows))

	var visited []db.ListRowsToCompactRow
	cursor := db.ListRowsToCompactRow{Fingerprint: []byte{}, CompactAfterMs: compactAfterMsBase}
	for range 10 {
		page, err := db.Query.ListRowsToCompact(ctx, conn, db.ListRowsToCompactParams{
			NowMs:               compactAfterMsBase + 300,
			AfterCompactAfterMs: cursor.CompactAfterMs,
			AfterFingerprint:    cursor.Fingerprint,
			Limit:               2,
		})
		require.NoError(t, err)
		if len(page) == 0 {
			break
		}
		visited = append(visited, page...)
		cursor = page[len(page)-1]
	}

	require.Equal(t, []db.ListRowsToCompactRow{
		{Fingerprint: a, CompactAfterMs: compactAfterMsBase + 100},
		{Fingerprint: a, CompactAfterMs: compactAfterMsBase + 100},
		{Fingerprint: b, CompactAfterMs: compactAfterMsBase + 100},
		{Fingerprint: c, CompactAfterMs: compactAfterMsBase + 100},
		{Fingerprint: a, CompactAfterMs: compactAfterMsBase + 200},
	}, visited)
}

// TestDeleteSession_KeepsRowChangedAfterRead guarantees that cleanup never
// deletes a value it did not check: a write that raised consumed_ns after
// cleanup read the row keeps the row.
func TestDeleteSession_KeepsRowChangedAfterRead(t *testing.T) {
	t.Parallel()
	conn := openMySQL(t)
	ctx := t.Context()
	fingerprint := newFingerprints(t, 1)[0]
	sessionID := uid.New(uid.TestPrefix)
	upsert := func(consumedNs int64) {
		t.Helper()
		require.NoError(t, db.Query.UpsertSession(ctx, conn, db.UpsertSessionParams{
			Fingerprint: fingerprint, SessionID: sessionID, AnchorAtNs: 1, ConsumedNs: consumedNs, CompactAfterMs: 1,
		}))
	}

	upsert(5)
	read := listSessions(t, conn, fingerprint, 10)
	require.Len(t, read, 1)
	upsert(6)

	deleted, err := db.Query.DeleteSession(ctx, conn, db.DeleteSessionParams{
		Fingerprint: fingerprint, SessionID: sessionID, ConsumedNs: read[0].ConsumedNs,
	})
	require.NoError(t, err)
	require.Equal(t, int64(0), deleted)
	require.Len(t, listSessions(t, conn, fingerprint, 10), 1)

	deleted, err = db.Query.DeleteSession(ctx, conn, db.DeleteSessionParams{
		Fingerprint: fingerprint, SessionID: sessionID, ConsumedNs: 6,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
	require.Empty(t, listSessions(t, conn, fingerprint, 10))
}
