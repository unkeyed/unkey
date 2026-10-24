package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keydb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestRefillKeysByIDsUsesDurableRefillTime(t *testing.T) {
	tx := beginRollbackTx(t, openTestDatabase(t))
	q := NewQueries(tx)
	workspaceID, keyID := uid.New(uid.WorkspacePrefix), uid.New(uid.KeyPrefix)
	insertTestWorkspace(t, q, workspaceID, uid.DNS1035())
	require.NoError(t, q.InsertKey(t.Context(), InsertKeyParams{
		ID: keyID, WorkspaceID: workspaceID, KeySpaceID: uid.New(uid.KeySpacePrefix),
		Hash: keyID, Enabled: true,
		RemainingRequests: sql.NullInt64{Int64: 17, Valid: true},
		RefillAmount:      sql.NullInt64{Int64: 123, Valid: true},
	}))
	refillTime := time.Date(2026, 1, 2, 3, 4, 5, 678000000, time.UTC)
	params := RefillKeysByIDsParams{
		RefillAt: sql.NullTime{Time: refillTime, Valid: true},
		Now:      sql.NullInt64{Int64: refillTime.UnixMilli(), Valid: true}, Ids: []string{keyID},
	}
	updated, err := q.RefillKeysByIDs(t.Context(), params)
	require.NoError(t, err)
	require.Equal(t, int64(1), updated)
	stored, err := q.FindKeyByID(t.Context(), keyID)
	require.NoError(t, err)
	require.Equal(t, int64(123), stored.RemainingRequests.Int64)
	require.True(t, stored.LastRefillAt.Valid)
	require.Equal(t, refillTime.UnixMilli(), stored.LastRefillAt.Time.UnixMilli())

	require.NoError(t, keydb.Query.UpdateKeyCreditsDecrement(t.Context(), tx, keydb.UpdateKeyCreditsDecrementParams{
		ID: keyID, Credits: sql.NullInt64{Int64: 9, Valid: true},
	}))
	updated, err = q.RefillKeysByIDs(t.Context(), params)
	require.NoError(t, err)
	require.Zero(t, updated)
	stored, err = q.FindKeyByID(t.Context(), keyID)
	require.NoError(t, err)
	require.Equal(t, int64(114), stored.RemainingRequests.Int64, "replay must not restore credits spent after commit")
	params.RefillAt.Time = refillTime.Add(-time.Millisecond)
	params.Now.Int64--
	updated, err = q.RefillKeysByIDs(t.Context(), params)
	require.NoError(t, err)
	require.Zero(t, updated, "an older invocation must not overwrite a newer refill")
	params.RefillAt.Time = refillTime.Add(time.Millisecond)
	params.Now.Int64 = refillTime.UnixMilli() + 1
	updated, err = q.RefillKeysByIDs(t.Context(), params)
	require.NoError(t, err)
	require.Equal(t, int64(1), updated, "a later refill must still be allowed")
}
