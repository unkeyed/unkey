package ratelimit

import (
	"crypto/sha256"
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestMySQLStoreRoundTrip(t *testing.T) {
	ctx := t.Context()
	conn, err := sql.Open("mysql", containers.MySQL(t).DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	store := NewMySQLStore(conn, conn)

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
