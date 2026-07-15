package cron_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

func TestRunKeyLastUsedSync_Integration(t *testing.T) {
	h := harness.New(t)

	t.Run("syncs last_used_at from ClickHouse to MySQL", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		keyIDs := make([]string, 3)
		for i := range keyIDs {
			resp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
				WorkspaceID: ws.ID,
				KeySpaceID:  api.KeyAuthID.String,
			})
			keyIDs[i] = resp.KeyID
		}

		now := time.Now().UnixMilli()
		chRows := make([]seed.KeyLastUsedRow, len(keyIDs))
		for i, keyID := range keyIDs {
			chRows[i] = seed.KeyLastUsedRow{
				WorkspaceID: ws.ID,
				KeySpaceID:  api.KeyAuthID.String,
				KeyID:       keyID,
				IdentityID:  "",
				Time:        now - int64((len(keyIDs)-i)*1000),
				RequestID:   uid.New(uid.RequestPrefix),
				Outcome:     "VALID",
				Tags:        []string{},
			}
		}
		h.ClickHouseSeed.InsertKeyLastUsed(h.Ctx, chRows)

		_, err := callRunKeyLastUsedSync(h)
		require.NoError(t, err)

		for i, keyID := range keyIDs {
			key, kErr := h.DB.FindKeyByID(h.Ctx, keyID)
			require.NoError(t, kErr)
			expectedMinute := ((now - int64((len(keyIDs)-i)*1000)) / 60_000) * 60_000
			require.GreaterOrEqual(t, int64(key.LastUsedAt), expectedMinute,
				"key %s last_used_at %d should match ClickHouse Time floor %d", keyID, key.LastUsedAt, expectedMinute)
		}
	})

	t.Run("does not regress last_used_at when MySQL is newer", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		resp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID: ws.ID,
			KeySpaceID:  api.KeyAuthID.String,
		})

		now := time.Now().UnixMilli()
		chRows := []seed.KeyLastUsedRow{{
			WorkspaceID: ws.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       resp.KeyID,
			IdentityID:  "",
			Time:        now,
			RequestID:   uid.New(uid.RequestPrefix),
			Outcome:     "VALID",
			Tags:        []string{},
		}}
		h.ClickHouseSeed.InsertKeyLastUsed(h.Ctx, chRows)

		_, err := callRunKeyLastUsedSync(h)
		require.NoError(t, err)

		evenNewer := now + 60_000
		_, err = h.DB.RW().ExecContext(h.Ctx,
			"UPDATE `keys` SET last_used_at = ? WHERE id = ?",
			evenNewer, resp.KeyID,
		)
		require.NoError(t, err)

		_, err = callRunKeyLastUsedSync(h)
		require.NoError(t, err)

		key, err := h.DB.FindKeyByID(h.Ctx, resp.KeyID)
		require.NoError(t, err)
		require.Equal(t, evenNewer, int64(key.LastUsedAt), "sync should not overwrite a newer MySQL timestamp")
	})
}

func callRunKeyLastUsedSync(h *harness.Harness) (*hydrav1.RunKeyLastUsedSyncResponse, error) {
	// VO key is ignored by RunKeyLastUsedSync; "default" pins concurrent
	// triggers into one queue.
	client := hydrav1.NewCronServiceIngressClient(h.Restate, "default")
	return client.RunKeyLastUsedSync().Request(h.Ctx, &hydrav1.RunKeyLastUsedSyncRequest{})
}
