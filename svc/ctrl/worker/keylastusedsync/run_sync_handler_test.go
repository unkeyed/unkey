package keylastusedsync_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/restatedev/sdk-go/ingress"
	restateServer "github.com/restatedev/sdk-go/server"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/worker/keylastusedsync"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

func TestRunSync_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	// Use the already-running dev docker compose containers instead of spinning
	// up fresh ones. This saves minutes of container startup time.
	// Requires: docker compose -f dev/docker-compose.yaml up -d mysql clickhouse restate
	h := newDevComposeHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	t.Cleanup(cancel)

	const totalKeys = 1_500_000
	const eventsPerKey = 100
	const totalEvents = totalKeys * eventsPerKey // 150M

	// Check if data is already seeded from a previous run
	keyIDs, alreadySeeded := checkExistingPerfData(t, ctx, h)

	if alreadySeeded {
		t.Logf("Skipping seeding — found %d keys already in MySQL + ClickHouse", len(keyIDs))
	} else {
		ws := h.Seed.CreateWorkspace(ctx)
		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: ws.ID,
		})

		// ── Phase 1: Seed MySQL with 10M keys ────────────────────────────────
		t.Logf("Phase 1: Seeding %d keys into MySQL...", totalKeys)
		mysqlStart := time.Now()
		keyIDs = bulkInsertMySQLKeys(t, ctx, h.DB, ws.ID, api.KeyAuthID.String, totalKeys)
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
	}

	// Reset last_used_at so the sync has work to do
	t.Logf("Resetting last_used_at for all keys...")
	resetStart := time.Now()
	_, err := h.DB.RW().ExecContext(ctx, "UPDATE `keys` SET last_used_at = NULL")
	require.NoError(t, err)
	t.Logf("Reset complete in %s", time.Since(resetStart))

	// ── Run sync — single invocation with partitioned fan-out ────────────
	t.Logf("Running sync (partitioned fan-out)...")
	syncStart := time.Now()
	resp, syncErr := callRunSyncCtx(ctx, h)
	require.NoError(t, syncErr)

	syncDuration := time.Since(syncStart)
	t.Logf("Sync complete: %d keys in %s (%.0f keys/sec)",
		resp.GetKeysSynced(), syncDuration, float64(resp.GetKeysSynced())/syncDuration.Seconds())

	require.Equal(t, int32(len(keyIDs)), resp.GetKeysSynced())

	// ── Spot-check ───────────────────────────────────────────────────────
	const sampleSize = 1000
	t.Logf("Spot-checking %d keys...", sampleSize)
	for i := range sampleSize {
		idx := i * (totalKeys / sampleSize)
		key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyIDs[idx])
		require.NoError(t, err)
		require.Greater(t, key.LastUsedAt, uint64(0), "key %s (index %d) should have last_used_at", keyIDs[idx], idx)
	}
	t.Logf("All %d spot-checks passed", sampleSize)
}

// TestRunSync_IncrementalUpdate verifies that partition cursors persist across
// invocations. Requires TestRunSync_Performance to have been run first so that
// partition cursors already exist in Restate state. We insert newer timestamps
// for a subset of keys and run sync — only those keys should be processed.
//
// In prod, the orchestrator should be keyed with a fixed key like "global" so
// the partition objects (0 .. 7) keep their cursor state.
func TestRunSync_IncrementalUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping incremental test in short mode")
	}

	h := newDevComposeHarness(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	t.Cleanup(cancel)

	// We need existing data + partition cursors from a prior run.
	keyIDs, alreadySeeded := checkExistingPerfData(t, ctx, h)
	require.True(t, alreadySeeded, "run TestRunSync_Performance first to seed data")
	t.Logf("Found %d existing keys", len(keyIDs))

	// Look up workspace/keyspace from an existing key
	existingKey, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyIDs[0])
	require.NoError(t, err)
	wsID := existingKey.WorkspaceID
	ksID := existingKey.KeyAuthID

	// Record current last_used_at for our subset before inserting new data
	const updateCount = 50_000
	updateKeys := keyIDs[:updateCount]

	oldTimestamps := make(map[string]int64, updateCount)
	for _, kid := range updateKeys {
		k, kErr := db.Query.FindKeyByID(ctx, h.DB.RO(), kid)
		require.NoError(t, kErr)
		oldTimestamps[kid] = int64(k.LastUsedAt)
	}

	// ── Insert newer CH data for only 50K keys ──────────────────────────
	newerTime := time.Now().UnixMilli()
	t.Logf("Inserting newer ClickHouse data for %d keys (time=%d)...", updateCount, newerTime)

	chRows := make([]seed.KeyLastUsedRow, updateCount)
	for i, kid := range updateKeys {
		chRows[i] = seed.KeyLastUsedRow{
			WorkspaceID: wsID,
			KeySpaceID:  ksID,
			KeyID:       kid,
			Time:        newerTime + int64(i), // unique timestamps so cursor advances
			RequestID:   uid.New(uid.RequestPrefix),
			Outcome:     "VALID",
			Tags:        []string{},
		}
	}
	h.ClickHouseSeed.InsertKeyLastUsed(ctx, chRows)

	// ── Run sync — cursors carry over from the perf test ────────────────
	t.Logf("Running incremental sync (same 'global' key, cursors carry over)...")
	syncStart := time.Now()
	resp, err := callRunSyncCtx(ctx, h)
	require.NoError(t, err)

	t.Logf("Incremental sync complete: %d keys in %s (expected ~%d)",
		resp.GetKeysSynced(), time.Since(syncStart), updateCount)

	// Should have synced roughly the 50K updated keys, not all 1.5M+
	require.Greater(t, resp.GetKeysSynced(), int32(0), "should have synced some keys")
	require.LessOrEqual(t, resp.GetKeysSynced(), int32(updateCount*2),
		"incremental sync should only process the newly inserted keys, not the full dataset")

	// Verify the updated keys have the newer timestamp and actually changed
	for _, kid := range updateKeys {
		k, kErr := db.Query.FindKeyByID(ctx, h.DB.RO(), kid)
		require.NoError(t, kErr)
		require.Greater(t, k.LastUsedAt, uint64(0))
		newerMinute := (newerTime / 60_000) * 60_000
		require.GreaterOrEqual(t, int64(k.LastUsedAt), newerMinute,
			"key %s should have the newer timestamp after incremental sync", kid)
		require.Greater(t, int64(k.LastUsedAt), oldTimestamps[kid],
			"key %s last_used_at should have advanced from %d", kid, oldTimestamps[kid])
	}

	// The real cursor resume proof: synced count should be much less than total keys.
	// If cursors reset to zero, all keys would be re-scanned from ClickHouse.
	totalKeys := int32(len(keyIDs))
	require.Less(t, resp.GetKeysSynced(), totalKeys,
		"incremental sync should process fewer keys than total (%d) — proves cursor resume", totalKeys)

	t.Logf("Incremental sync verified: %d/%d keys synced, all %d updated keys have newer timestamps",
		resp.GetKeysSynced(), totalKeys, updateCount)
}

// newDevComposeHarness connects to the already-running dev docker compose
// containers and registers the worker handlers with Restate.
// Hardcoded addresses match dev/docker-compose.yaml.
func newDevComposeHarness(t *testing.T) *harness.Harness {
	t.Helper()

	const (
		mysqlDSN       = "unkey:password@tcp(localhost:3306)/unkey?parseTime=true&multiStatements=true"
		clickhouseDSN  = "clickhouse://default:password@localhost:9000?secure=false&skip_verify=true&dial_timeout=10s"
		restateIngress = "http://localhost:8081"
		restateAdmin   = "http://localhost:9070"
	)

	// Skip if dev compose isn't running (e.g. CI)
	probeConn, probeErr := net.DialTimeout("tcp", "localhost:9070", 2*time.Second)
	if probeErr != nil {
		t.Skip("skipping: dev compose not running (Restate admin not reachable on localhost:9070)")
	}
	_ = probeConn.Close()

	ctx := context.Background()

	// Connect to MySQL
	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	// Connect to ClickHouse
	chClient, err := clickhouse.New(clickhouse.Config{
		URL: clickhouseDSN,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, chClient.Close()) })

	chOpts, err := ch.ParseDSN(clickhouseDSN)
	require.NoError(t, err)
	conn, err := ch.Open(chOpts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	// Create services
	syncSvc, err := keylastusedsync.New(keylastusedsync.Config{
		Heartbeat: healthcheck.NewNoop(),
	})
	require.NoError(t, err)

	partitionSvc, err := keylastusedsync.NewPartitionService(keylastusedsync.PartitionConfig{
		DB:         database,
		Clickhouse: chClient,
	})
	require.NoError(t, err)

	// Start Restate worker server
	restateSrv := restateServer.NewRestate()
	restateSrv.Bind(hydrav1.NewKeyLastUsedSyncServiceServer(syncSvc))
	restateSrv.Bind(hydrav1.NewKeyLastUsedPartitionServiceServer(partitionSvc))

	restateHandler, err := restateSrv.Handler()
	require.NoError(t, err)

	workerMux := http.NewServeMux()
	workerMux.Handle("/", restateHandler)

	workerListener, err := net.Listen("tcp", "0.0.0.0:0") //nolint:gosec
	require.NoError(t, err)
	workerServer := httptest.NewUnstartedServer(h2c.NewHandler(workerMux, &http2.Server{})) //nolint:exhaustruct
	workerServer.Listener = workerListener
	workerServer.Start()
	t.Cleanup(workerServer.Close)

	tcpAddr, ok := workerListener.Addr().(*net.TCPAddr)
	require.True(t, ok)
	registerAs := fmt.Sprintf("http://host.docker.internal:%d", tcpAddr.Port)

	adminClient := restateadmin.New(restateadmin.Config{BaseURL: restateAdmin, APIKey: ""})
	require.NoError(t, adminClient.RegisterDeployment(ctx, registerAs))
	t.Logf("Worker registered with dev Restate at %s", registerAs)

	seeder := seed.New(t, database, nil)

	return &harness.Harness{
		Ctx:            ctx,
		DB:             database,
		Seed:           seeder,
		ClickHouseSeed: seed.NewClickHouseSeeder(t, conn),
		ClickHouse:     chClient,
		ClickHouseConn: conn,
		ClickHouseDSN:  clickhouseDSN,
		Restate:        ingress.NewClient(restateIngress),
		RestateIngress: restateIngress,
		RestateAdmin:   restateAdmin,
	}
}

// checkExistingPerfData checks if MySQL already has ~10M test keys and ClickHouse
// has matching data. Returns the key IDs and true if seeding can be skipped.
func checkExistingPerfData(t *testing.T, ctx context.Context, h *harness.Harness) ([]string, bool) {
	t.Helper()

	// Count keys with the test prefix
	var keyCount int
	err := h.DB.RO().QueryRowContext(ctx, "SELECT COUNT(*) FROM `keys` WHERE id LIKE 'key_%'").Scan(&keyCount)
	if err != nil || keyCount < 1_500_000 {
		return nil, false
	}

	// Check ClickHouse has data too
	var chCount uint64
	row := h.ClickHouseConn.QueryRow(ctx, "SELECT count() FROM default.key_last_used_v1")
	if err := row.Scan(&chCount); err != nil || chCount == 0 {
		return nil, false
	}

	t.Logf("Found %d keys in MySQL, %d rows in ClickHouse key_last_used_v1", keyCount, chCount)

	// Load all key IDs for spot-checking later
	rows, err := h.DB.RO().QueryContext(ctx, "SELECT id FROM `keys` WHERE id LIKE 'key_%' ORDER BY id")
	if err != nil {
		return nil, false
	}
	defer func() { _ = rows.Close() }()

	keyIDs := make([]string, 0, keyCount)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, false
		}
		keyIDs = append(keyIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, false
	}

	return keyIDs, len(keyIDs) >= 1_500_000
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

	// Split key IDs into worker chunks
	chunkSize := (count + workers - 1) / workers
	var wg sync.WaitGroup
	var inserted atomic.Int64
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

func callRunSyncCtx(ctx context.Context, h *harness.Harness) (*hydrav1.RunSyncResponse, error) {
	client := hydrav1.NewKeyLastUsedSyncServiceIngressClient(h.Restate)
	return client.RunSync().Request(ctx, &hydrav1.RunSyncRequest{})
}
