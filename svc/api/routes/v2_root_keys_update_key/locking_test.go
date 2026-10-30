package handler_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

func TestRootKeyMutationPreservesDatabaseErrors(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	key := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := db.LockUnkeyRootKeys(ctx, h.DB.RW(), key.KeyID)
	require.ErrorIs(t, err, context.Canceled)
	_, coded := fault.GetCode(err)
	require.False(t, coded, "database failures must not become authorization failures")
}

// Reciprocal mutations must complete without relying on deadlock retries.
func TestReciprocalRootKeyMutationsDoNotDeadlock(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.Resources().UserWorkspace
	a := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	b := h.CreateUnkeyRootKey(seed.CreateUnkeyRootKeyRequest{WorkspaceID: workspace.ID})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	for range 32 {
		start := make(chan struct{})
		ready := make(chan struct{}, 2)
		results := make(chan error, 2)
		for _, pair := range [][2]string{{a.KeyID, b.KeyID}, {b.KeyID, a.KeyID}} {
			go func() {
				results <- db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
					ready <- struct{}{}
					select {
					case <-start:
					case <-ctx.Done():
						return ctx.Err()
					}
					keys, err := db.LockUnkeyRootKeys(ctx, tx, pair[0], pair[1])
					if err != nil {
						return err
					}
					return db.Query.UpdateUnkeyRootKey(ctx, tx, db.UpdateUnkeyRootKeyParams{
						ID:               keys[pair[1]].ID,
						WorkspaceID:      workspace.ID,
						EnabledSpecified: 1,
						Enabled:          sql.NullBool{Bool: true, Valid: true},
					})
				})
			}()
		}
		for range 2 {
			select {
			case <-ready:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}
		close(start)
		first, second := <-results, <-results
		require.NoError(t, first)
		require.NoError(t, second)
	}
}
