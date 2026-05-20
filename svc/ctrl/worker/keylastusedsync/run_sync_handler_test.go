package keylastusedsync_test

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

func TestRunSync_Integration(t *testing.T) {
	h := harness.New(t)

	t.Run("syncs last_used_at from ClickHouse to MySQL", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// Create 3 keys in MySQL
		keyIDs := make([]string, 3)
		for i := range keyIDs {
			resp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
				WorkspaceID: ws.ID,
				KeySpaceID:  api.KeyAuthID.String,
			})
			keyIDs[i] = resp.KeyID
		}

		// Seed ClickHouse with last-used timestamps
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

		// Run the sync
		resp, err := callRunSync(h)
		require.NoError(t, err)
		require.Equal(t, int32(3), resp.GetKeysSynced())

		// Verify MySQL was updated
		for i, keyID := range keyIDs {
			key, keyErr := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), keyID)
			require.NoError(t, keyErr)
			require.Greater(t, key.LastUsedAt, uint64(0), "key %s should have last_used_at set", keyID)
			// Timestamps are truncated to minute granularity
			expectedMinute := (chRows[i].Time / 60_000) * 60_000
			require.Equal(t, expectedMinute, int64(key.LastUsedAt))
		}
	})

	t.Run("is idempotent — does not overwrite with older timestamps", func(t *testing.T) {
		ws := h.Seed.CreateWorkspace(h.Ctx)
		api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		resp := h.Seed.CreateKey(h.Ctx, seed.CreateKeyRequest{
			WorkspaceID: ws.ID,
			KeySpaceID:  api.KeyAuthID.String,
		})

		newerTime := time.Now().UnixMilli()

		// Seed CH with a timestamp
		h.ClickHouseSeed.InsertKeyLastUsed(h.Ctx, []seed.KeyLastUsedRow{{
			WorkspaceID: ws.ID,
			KeySpaceID:  api.KeyAuthID.String,
			KeyID:       resp.KeyID,
			Time:        newerTime,
			RequestID:   uid.New(uid.RequestPrefix),
			Outcome:     "VALID",
			Tags:        []string{},
		}})

		// Sync once
		_, err := callRunSync(h)
		require.NoError(t, err)

		// Verify
		key, err := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), resp.KeyID)
		require.NoError(t, err)
		newerMinute := (newerTime / 60_000) * 60_000
		require.Equal(t, newerMinute, int64(key.LastUsedAt))

		// Now manually set MySQL to an even newer timestamp
		evenNewer := newerTime + 999999
		_, err = h.DB.RW().ExecContext(h.Ctx,
			"UPDATE `keys` SET last_used_at = ? WHERE id = ?",
			evenNewer, resp.KeyID,
		)
		require.NoError(t, err)

		// Re-sync (CH still has newerTime which is older than evenNewer)
		_, err = callRunSync(h)
		require.NoError(t, err)

		// MySQL should still have the newer value
		key, err = db.Query.FindKeyByID(h.Ctx, h.DB.RO(), resp.KeyID)
		require.NoError(t, err)
		require.Equal(t, evenNewer, int64(key.LastUsedAt), "sync should not overwrite a newer MySQL timestamp")
	})
}

// TestRunSync_Performance exercises the full sync path end-to-end with a
// realistic data volume, then verifies that a follow-up incremental sync
// resumes from the persisted partition cursors instead of rescanning the
// full dataset. Both behaviours share a single harness so the Restate
// cursor state established by the first sync is visible to the second.
func TestRunSync_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	h := harness.New(t,
		harness.WithDiskMySQL(),
		harness.WithTimeout(60*time.Minute),
	)

	const totalKeys = 1_500_000
	const eventsPerKey = 100
	const totalEvents = totalKeys * eventsPerKey

	ws := h.Seed.CreateWorkspace(h.Ctx)
	api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
		WorkspaceID: ws.ID,
	})

	// Phase 1: Seed MySQL with keys
	t.Logf("Phase 1: Seeding %d keys into MySQL...", totalKeys)
	mysqlStart := time.Now()
	keyIDs := bulkInsertMySQLKeys(t, h.Ctx, h.DB, ws.ID, api.KeyAuthID.String, totalKeys)
	t.Logf("Phase 1 complete: %d keys in %s (%.0f keys/sec)",
		totalKeys, time.Since(mysqlStart), float64(totalKeys)/time.Since(mysqlStart).Seconds())

	// Phase 2: Seed ClickHouse with raw verification events
	t.Logf("Phase 2: Seeding %d raw verification events into ClickHouse (%d keys × %d events/key, 8 workers)...",
		totalEvents, totalKeys, eventsPerKey)
	chStart := time.Now()
	h.ClickHouseSeed.InsertRawVerificationsForKeys(h.Ctx, ws.ID, api.KeyAuthID.String, keyIDs, eventsPerKey)
	chDuration := time.Since(chStart)
	t.Logf("Phase 2 complete: %d events in %s (%.0f events/sec)",
		totalEvents, chDuration, float64(totalEvents)/chDuration.Seconds())

	// Phase 3: Force ClickHouse merge so the sync reads from a single part
	t.Logf("Phase 3: OPTIMIZE TABLE key_last_used_v1 FINAL...")
	optimizeStart := time.Now()
	require.NoError(t, h.ClickHouseConn.Exec(h.Ctx, "OPTIMIZE TABLE default.key_last_used_v1 FINAL"))
	t.Logf("Phase 3 complete: OPTIMIZE in %s", time.Since(optimizeStart))

	t.Run("full sync covers every seeded key", func(t *testing.T) {
		t.Logf("Running sync (partitioned fan-out)...")
		syncStart := time.Now()
		resp, err := callRunSync(h)
		require.NoError(t, err)

		syncDuration := time.Since(syncStart)
		t.Logf("Sync complete: %d keys in %s (%.0f keys/sec)",
			resp.GetKeysSynced(), syncDuration, float64(resp.GetKeysSynced())/syncDuration.Seconds())

		require.Equal(t, int32(len(keyIDs)), resp.GetKeysSynced())

		// Spot-check a sample of keys to confirm last_used_at landed
		const sampleSize = 1000
		t.Logf("Spot-checking %d keys...", sampleSize)
		for i := range sampleSize {
			idx := i * (totalKeys / sampleSize)
			key, kErr := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), keyIDs[idx])
			require.NoError(t, kErr)
			require.Greater(t, key.LastUsedAt, uint64(0), "key %s (index %d) should have last_used_at", keyIDs[idx], idx)
		}
		t.Logf("All %d spot-checks passed", sampleSize)
	})

	t.Run("incremental sync resumes from persisted partition cursors", func(t *testing.T) {
		// Inserting newer ClickHouse rows for a small subset and resyncing
		// proves the cursor advanced — only the freshly written rows should
		// be scanned, not the original 150M.
		const updateCount = 50_000
		updateKeys := keyIDs[:updateCount]

		oldTimestamps := make(map[string]int64, updateCount)
		for _, kid := range updateKeys {
			k, kErr := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), kid)
			require.NoError(t, kErr)
			oldTimestamps[kid] = int64(k.LastUsedAt)
		}

		newerTime := time.Now().UnixMilli()
		t.Logf("Inserting newer ClickHouse data for %d keys (time=%d)...", updateCount, newerTime)

		chRows := make([]seed.KeyLastUsedRow, updateCount)
		for i, kid := range updateKeys {
			chRows[i] = seed.KeyLastUsedRow{
				WorkspaceID: ws.ID,
				KeySpaceID:  api.KeyAuthID.String,
				KeyID:       kid,
				Time:        newerTime + int64(i), // unique timestamps so cursor advances
				RequestID:   uid.New(uid.RequestPrefix),
				Outcome:     "VALID",
				Tags:        []string{},
			}
		}
		h.ClickHouseSeed.InsertKeyLastUsed(h.Ctx, chRows)

		t.Logf("Running incremental sync (cursors carry over from previous sync)...")
		syncStart := time.Now()
		resp, err := callRunSync(h)
		require.NoError(t, err)

		t.Logf("Incremental sync complete: %d keys in %s (expected ~%d)",
			resp.GetKeysSynced(), time.Since(syncStart), updateCount)

		require.Greater(t, resp.GetKeysSynced(), int32(0), "should have synced some keys")
		require.LessOrEqual(t, resp.GetKeysSynced(), int32(updateCount*2),
			"incremental sync should only process the newly inserted keys, not the full dataset")

		// Verify the updated keys advanced past their previous timestamp
		for _, kid := range updateKeys {
			k, kErr := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), kid)
			require.NoError(t, kErr)
			require.Greater(t, k.LastUsedAt, uint64(0))
			newerMinute := (newerTime / 60_000) * 60_000
			require.GreaterOrEqual(t, int64(k.LastUsedAt), newerMinute,
				"key %s should have the newer timestamp after incremental sync", kid)
			require.Greater(t, int64(k.LastUsedAt), oldTimestamps[kid],
				"key %s last_used_at should have advanced from %d", kid, oldTimestamps[kid])
		}

		// Cursor resume proof: the second sync touched far fewer keys than total.
		require.Less(t, resp.GetKeysSynced(), int32(len(keyIDs)),
			"incremental sync should process fewer keys than total (%d) — proves cursor resume", len(keyIDs))

		t.Logf("Incremental sync verified: %d/%d keys synced, all %d updated keys have newer timestamps",
			resp.GetKeysSynced(), len(keyIDs), updateCount)
	})
}

// bulkInsertMySQLKeys inserts keys via raw SQL using 8 parallel workers.
func bulkInsertMySQLKeys(t *testing.T, ctx context.Context, database db.Database, workspaceID, keySpaceID string, count int) []string {
	t.Helper()

	keyIDs := make([]string, count)
	for i := range keyIDs {
		keyIDs[i] = uid.New(uid.KeyPrefix, 12)
	}

	const batchSize = 9_000 // 9K × 7 columns = 63K placeholders (MySQL limit is 65,535)
	const workers = 8
	now := time.Now().UnixMilli()

	chunkSize := (count + workers - 1) / workers
	var wg sync.WaitGroup
	inserted := &atomic.Int64{}
	errCh := make(chan error, workers)

	for w := range workers {
		start := w * chunkSize
		end := min(start+chunkSize, count)
		if start >= count {
			break
		}
		workerKeys := keyIDs[start:end]

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < len(workerKeys); i += batchSize {
				batchEnd := min(i+batchSize, len(workerKeys))
				chunk := workerKeys[i:batchEnd]

				var queryBuilder strings.Builder
				queryBuilder.WriteString("INSERT INTO `keys` (id, key_auth_id, hash, start, workspace_id, created_at_m, enabled) VALUES ")

				args := make([]any, 0, len(chunk)*7)
				for j, keyID := range chunk {
					if j > 0 {
						queryBuilder.WriteString(", ")
					}
					queryBuilder.WriteString("(?, ?, ?, ?, ?, ?, ?)")
					args = append(args, keyID, keySpaceID, hash.Sha256(keyID), keyID[:8], workspaceID, now, true)
				}

				if _, err := database.RW().ExecContext(ctx, queryBuilder.String(), args...); err != nil {
					errCh <- err
					return
				}

				n := inserted.Add(int64(len(chunk)))
				if n%(batchSize*100) < int64(len(chunk)) {
					t.Logf("  MySQL: inserted %d/%d keys", n, count)
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}

	return keyIDs
}

func callRunSync(h *harness.Harness) (*hydrav1.RunSyncResponse, error) {
	client := hydrav1.NewKeyLastUsedSyncServiceIngressClient(h.Restate)
	return client.RunSync().Request(h.Ctx, &hydrav1.RunSyncRequest{})
}
