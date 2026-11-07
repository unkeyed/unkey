package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
)

// TestRerollKeyConcurrentRerollsMintOneKey pins the money rule: two concurrent
// rerolls of the same key with immediate expiration must produce exactly one
// replacement key. The loser must be refused instead of minting a second key
// that copies the same remaining credits, which would make the balance
// spendable twice.
func TestRerollKeyConcurrentRerollsMintOneKey(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Remaining:   new(int64(10)),
	})

	rootKey := h.CreateRootKey(workspace.ID, "api.*.create_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}

	body, err := json.Marshal(handler.Request{
		KeyId:      key.KeyID,
		Expiration: nullable.NewNullableWithValue(int64(0)),
	})
	require.NoError(t, err)

	// Hold the source row so both rerolls are in flight at the same time and
	// have to queue behind the same lock, the way two real callers would.
	lock, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = lock.Rollback() })
	_, err = db.Query.LockKeyForUpdate(ctx, lock, key.KeyID)
	require.NoError(t, err)

	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 2)
	for range 2 {
		wg.Go(func() {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(route.Method(), route.Path(), bytes.NewReader(body))
			req.Header = headers
			h.Mux().ServeHTTP(rr, req)
			responses <- rr
		})
	}

	require.NoError(t, lock.Commit())

	var (
		winner   handler.Response
		statuses = map[int]int{}
	)
	for range 2 {
		select {
		case rr := <-responses:
			statuses[rr.Code]++
			if rr.Code == http.StatusOK {
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &winner))
			}
		case <-time.After(10 * time.Second):
			t.Fatal("reroll did not finish after the source row lock was released")
		}
	}
	wg.Wait()

	require.Equal(t, 1, statuses[http.StatusOK], "exactly one reroll must win, statuses: %v", statuses)
	require.Equal(t, 1, statuses[http.StatusBadRequest], "the losing reroll must be refused, statuses: %v", statuses)

	count, err := db.Query.CountKeysByWorkspace(ctx, h.DB.RO(), workspace.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), count, "the workspace must hold the source key and exactly one replacement")

	replacement, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), winner.Data.KeyId)
	require.NoError(t, err)
	require.Equal(t, int64(10), replacement.KeyRemainingRequests.Int64)

	source, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), key.KeyID)
	require.NoError(t, err)
	require.True(t, source.KeyExpires.Valid, "the source key must expire")
	require.False(t, source.KeyExpires.Time.After(time.Now()), "the source key must be expired")
}
