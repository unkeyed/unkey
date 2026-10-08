package cron_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
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
		require.NoError(t, pkgdb.Query.InsertUnkeyRootKey(h.Ctx, h.DB.RW(), pkgdb.InsertUnkeyRootKeyParams{
			ID:          rootKeyID,
			WorkspaceID: ws.ID,
			Hash:        uid.New("hash"),
			Name:        sql.NullString{Valid: false},
			Prefix:      "unkey",
			Start:       "abcd",
			End:         "wxyz",
			Enabled:     true,
			Expires:     sql.NullInt64{Valid: false},
			CreatedAt:   time.Now().UnixMilli(),
		}))

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

		_, err := callRunKeyLastUsedSync(h)
		require.NoError(t, err)

		for i, keyID := range keyIDs {
			key, kErr := h.DB.FindKeyByID(h.Ctx, keyID)
			require.NoError(t, kErr)
			expectedMinute := ((now - int64((len(keyIDs)-i)*1000)) / 60_000) * 60_000
			require.GreaterOrEqual(t, int64(key.LastUsedAt), expectedMinute,
				"key %s last_used_at %d should match ClickHouse Time floor %d", keyID, key.LastUsedAt, expectedMinute)
		}

		rootKey, err := pkgdb.Query.FindUnkeyRootKeyByID(h.Ctx, h.DB.RO(), rootKeyID)
		require.NoError(t, err)
		expectedMinute := (now / 60_000) * 60_000
		require.GreaterOrEqual(t, int64(rootKey.LastUsedAt), expectedMinute)
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
