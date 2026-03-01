package keylastusedsync_test

import (
	"context"
	"fmt"
	"strings"
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
		resp, err := callRunSync(h, fmt.Sprintf("test-basic-%s", uid.New("", 8)))
		require.NoError(t, err)
		require.Equal(t, int32(3), resp.GetKeysSynced())
		require.False(t, resp.GetHasMore())

		// Verify MySQL was updated
		for i, keyID := range keyIDs {
			key, keyErr := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), keyID)
			require.NoError(t, keyErr)
			require.True(t, key.LastUsedAt.Valid, "key %s should have last_used_at set", keyID)
			require.Equal(t, chRows[i].Time, key.LastUsedAt.Int64)
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
		runKey := fmt.Sprintf("test-idempotent-%s", uid.New("", 8))
		_, err := callRunSync(h, runKey)
		require.NoError(t, err)

		// Verify
		key, err := db.Query.FindKeyByID(h.Ctx, h.DB.RO(), resp.KeyID)
		require.NoError(t, err)
		require.Equal(t, newerTime, key.LastUsedAt.Int64)

		// Now manually set MySQL to an even newer timestamp
		evenNewer := newerTime + 999999
		_, err = h.DB.RW().ExecContext(h.Ctx,
			"UPDATE `keys` SET last_used_at = ? WHERE id = ?",
			evenNewer, resp.KeyID,
		)
		require.NoError(t, err)

		// Re-sync with a new key (CH still has newerTime which is older than evenNewer)
		runKey2 := fmt.Sprintf("test-idempotent2-%s", uid.New("", 8))
		_, err = callRunSync(h, runKey2)
		require.NoError(t, err)

		// MySQL should still have the newer value
		key, err = db.Query.FindKeyByID(h.Ctx, h.DB.RO(), resp.KeyID)
		require.NoError(t, err)
		require.Equal(t, evenNewer, key.LastUsedAt.Int64, "sync should not overwrite a newer MySQL timestamp")
	})
}

func TestRunSync_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	h := harness.New(t, harness.WithDiskMySQL())

	// Long-running context for the performance test (harness ctx is only 120s)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	t.Cleanup(cancel)

	const totalKeys = 10_000_000
	const eventsPerKey = 100
	const totalEvents = totalKeys * eventsPerKey // 1B

	ws := h.Seed.CreateWorkspace(h.Ctx)
	api := h.Seed.CreateAPI(h.Ctx, seed.CreateApiRequest{
		WorkspaceID: ws.ID,
	})

	// ── Phase 1: Seed MySQL with 10M keys ────────────────────────────────
	t.Logf("Phase 1: Seeding %d keys into MySQL...", totalKeys)
	mysqlStart := time.Now()
	keyIDs := bulkInsertMySQLKeys(t, ctx, h.DB, ws.ID, api.KeyAuthID.String, totalKeys)
	t.Logf("Phase 1 complete: %d keys in %s (%.0f keys/sec)",
		totalKeys, time.Since(mysqlStart), float64(totalKeys)/time.Since(mysqlStart).Seconds())

	// ── Phase 2: Seed ClickHouse with 1B raw verification events ─────────
	t.Logf("Phase 2: Seeding %d raw verification events into ClickHouse (%d keys × %d events/key, 8 workers)...",
		totalEvents, totalKeys, eventsPerKey)
	chStart := time.Now()
	h.ClickHouseSeed.InsertRawVerificationsForKeys(ctx, ws.ID, api.KeyAuthID.String, keyIDs, eventsPerKey)
	chDuration := time.Since(chStart)
	t.Logf("Phase 2 complete: %d events in %s (%.0f events/sec)",
		totalEvents, chDuration, float64(totalEvents)/chDuration.Seconds())

	// ── Phase 3: Force ClickHouse merge for realistic read perf ──────────
	t.Logf("Phase 3: OPTIMIZE TABLE key_last_used_v1 FINAL...")
	optimizeStart := time.Now()
	require.NoError(t, h.ClickHouseConn.Exec(ctx, "OPTIMIZE TABLE default.key_last_used_v1 FINAL"))
	t.Logf("Phase 3 complete: OPTIMIZE in %s", time.Since(optimizeStart))

	// ── Phase 4: Run sync until all 10M keys are synced ──────────────────
	t.Logf("Phase 4: Running sync...")
	syncStart := time.Now()
	var totalSynced int32
	invocations := 0
	for {
		runKey := fmt.Sprintf("perf-%s-%d", uid.New("", 8), invocations)
		resp, err := callRunSyncCtx(ctx, h, runKey)
		require.NoError(t, err)

		totalSynced += resp.GetKeysSynced()
		invocations++

		if invocations%10 == 0 || !resp.GetHasMore() {
			elapsed := time.Since(syncStart)
			t.Logf("  invocation %d: synced %d (total: %d/%d, %.1f%%, %.0f keys/sec)",
				invocations, resp.GetKeysSynced(), totalSynced, totalKeys,
				float64(totalSynced)/float64(totalKeys)*100,
				float64(totalSynced)/elapsed.Seconds())
		}

		if !resp.GetHasMore() {
			break
		}
	}
	syncDuration := time.Since(syncStart)
	t.Logf("Phase 4 complete: %d keys in %s across %d invocations (%.0f keys/sec)",
		totalSynced, syncDuration, invocations, float64(totalSynced)/syncDuration.Seconds())

	require.Equal(t, int32(totalKeys), totalSynced)

	// ── Spot-check ───────────────────────────────────────────────────────
	const sampleSize = 1000
	t.Logf("Spot-checking %d keys...", sampleSize)
	for i := range sampleSize {
		idx := i * (totalKeys / sampleSize)
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyIDs[idx])
		require.NoError(t, err)
		require.True(t, key.LastUsedAt.Valid, "key %s (index %d) should have last_used_at", keyIDs[idx], idx)
	}
	t.Logf("All %d spot-checks passed", sampleSize)
}

// bulkInsertMySQLKeys inserts keys directly via raw SQL in batches of 10,000.
func bulkInsertMySQLKeys(t *testing.T, ctx context.Context, database db.Database, workspaceID, keySpaceID string, count int) []string {
	t.Helper()

	keyIDs := make([]string, count)
	for i := range keyIDs {
		keyIDs[i] = uid.New(uid.KeyPrefix, 12)
	}

	const batchSize = 9_000 // 9K × 7 columns = 63K placeholders (MySQL limit is 65,535)
	now := time.Now().UnixMilli()

	for i := 0; i < count; i += batchSize {
		end := min(i+batchSize, count)
		chunk := keyIDs[i:end]

		var queryBuilder strings.Builder
		queryBuilder.WriteString("INSERT INTO `keys` (id, key_auth_id, hash, start, workspace_id, created_at_m, enabled) VALUES ")

		args := make([]any, 0, len(chunk)*7)
		for j, keyID := range chunk {
			if j > 0 {
				queryBuilder.WriteString(", ")
			}
			queryBuilder.WriteString("(?, ?, ?, ?, ?, ?, ?)")
			args = append(args,
				keyID,
				keySpaceID,
				hash.Sha256(keyID),
				keyID[:8],
				workspaceID,
				now,
				true,
			)
		}

		_, err := database.RW().ExecContext(ctx, queryBuilder.String(), args...)
		require.NoError(t, err)

		if (i/batchSize)%100 == 0 {
			t.Logf("  MySQL: inserted %d/%d keys", end, count)
		}
	}

	return keyIDs
}

func callRunSync(h *harness.Harness, runKey string) (*hydrav1.RunSyncResponse, error) {
	client := hydrav1.NewKeyLastUsedSyncServiceIngressClient(h.Restate, runKey)
	return client.RunSync().Request(h.Ctx, &hydrav1.RunSyncRequest{})
}

func callRunSyncCtx(ctx context.Context, h *harness.Harness, runKey string) (*hydrav1.RunSyncResponse, error) {
	client := hydrav1.NewKeyLastUsedSyncServiceIngressClient(h.Restate, runKey)
	return client.RunSync().Request(ctx, &hydrav1.RunSyncRequest{})
}
