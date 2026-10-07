package cron_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestRunKeyLastUsedSync_Integration(t *testing.T) {
	h := harness.New(t)

	t.Run("syncs legacy and root key last_used_at from ClickHouse to MySQL", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})
		rootKeyID := uid.New(uid.KeyPrefix)
		_, err := h.DB.RW().ExecContext(h.Ctx, `
			INSERT INTO unkey_root_keys (id, workspace_id, hash, name, prefix, start, end, enabled, expires, created_at)
			VALUES (?, ?, ?, NULL, ?, ?, ?, true, NULL, ?)
		`, rootKeyID, ws.ID, uid.New("hash"), "unkey", "abcd", "wxyz", time.Now().UnixMilli())
		require.NoError(t, err)

		keyIDs := make([]string, 3)
		for i := range keyIDs {
			resp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
				WorkspaceID: ws.ID,
				KeySpaceID:  api.KeyAuthID.String,
			})
			keyIDs[i] = resp.KeyID
		}

		now := time.Now().UnixMilli()
		chRows := make([]seed.KeyLastUsedRow, len(keyIDs), len(keyIDs)+1)
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
		chRows = append(chRows, seed.KeyLastUsedRow{
			WorkspaceID: "",
			KeySpaceID:  "",
			KeyID:       rootKeyID,
			IdentityID:  "",
			Time:        now,
			RequestID:   uid.New(uid.RequestPrefix),
			Outcome:     "VALID",
			Tags:        []string{},
		})
		h.ClickHouseSeed.InsertKeyLastUsed(h.Ctx, chRows)

		_, err = callRunKeyLastUsedSync(h)
		require.NoError(t, err)

		for i, keyID := range keyIDs {
			key, kErr := h.DB.FindKeyByID(h.Ctx, keyID)
			require.NoError(t, kErr)
			expectedMinute := ((now - int64((len(keyIDs)-i)*1000)) / 60_000) * 60_000
			require.GreaterOrEqual(t, int64(key.LastUsedAt), expectedMinute,
				"key %s last_used_at %d should match ClickHouse Time floor %d", keyID, key.LastUsedAt, expectedMinute)
		}

		var lastUsedAt uint64
		err = h.DB.RW().QueryRowContext(h.Ctx, "SELECT last_used_at FROM unkey_root_keys WHERE id = ?", rootKeyID).Scan(&lastUsedAt)
		require.NoError(t, err)
		expectedMinute := (now / 60_000) * 60_000
		require.GreaterOrEqual(t, int64(lastUsedAt), expectedMinute)
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
		require.NoError(t, h.DB.UpdateKeysLastUsed(h.Ctx, db.UpdateKeysLastUsedParams{
			LastUsedAt: uint64(evenNewer),
			KeyIds:     []string{resp.KeyID},
		}))

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
