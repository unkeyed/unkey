package seed

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
)

// ClickHouseSeeder provides methods to seed test data in ClickHouse.
type ClickHouseSeeder struct {
	t    *testing.T
	conn ch.Conn
}

// NewClickHouseSeeder creates a new ClickHouseSeeder instance.
func NewClickHouseSeeder(t *testing.T, conn ch.Conn) *ClickHouseSeeder {
	return &ClickHouseSeeder{t: t, conn: conn}
}

// InsertVerifications inserts key verification records for a workspace.
func (s *ClickHouseSeeder) InsertVerifications(ctx context.Context, workspaceID string, count int, timestamp time.Time, outcome string) {
	const batchSize = 10_000
	for i := 0; i < count; i += batchSize {
		batchCount := min(batchSize, count-i)
		verifications := make([]schema.KeyVerification, batchCount)
		for j := range batchCount {
			verifications[j] = schema.KeyVerification{
				RequestID:    uid.New(uid.RequestPrefix),
				Time:         timestamp.Add(time.Duration(i+j) * time.Millisecond).UnixMilli(),
				WorkspaceID:  workspaceID,
				KeySpaceID:   uid.New(uid.KeySpacePrefix),
				IdentityID:   "",
				ExternalID:   "",
				KeyID:        uid.New(uid.KeyPrefix),
				Region:       "us-east-1",
				Outcome:      outcome,
				Tags:         []string{},
				SpentCredits: 0,
				Latency:      rand.Float64() * 100, //nolint:gosec
			}
		}

		batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO default.key_verifications_raw_v2")
		require.NoError(s.t, err)

		for _, v := range verifications {
			err = batch.AppendStruct(&v)
			require.NoError(s.t, err)
		}

		err = batch.Send()
		require.NoError(s.t, err)
	}
}

// InsertRatelimits inserts ratelimit records for a workspace.
func (s *ClickHouseSeeder) InsertRatelimits(ctx context.Context, workspaceID string, count int, timestamp time.Time, passed bool) {
	const batchSize = 10_000
	var remaining uint64 = 50
	if !passed {
		remaining = 0
	}

	for i := 0; i < count; i += batchSize {
		batchCount := min(batchSize, count-i)
		ratelimits := make([]schema.Ratelimit, batchCount)
		for j := range batchCount {
			ratelimits[j] = schema.Ratelimit{
				RequestID:   uid.New(uid.RequestPrefix),
				Time:        timestamp.Add(time.Duration(i+j) * time.Millisecond).UnixMilli(),
				WorkspaceID: workspaceID,
				NamespaceID: uid.New(uid.RatelimitNamespacePrefix),
				Identifier:  uid.New(uid.IdentityPrefix),
				Passed:      passed,
				Latency:     rand.Float64() * 10, //nolint:gosec
				OverrideID:  "",
				Limit:       100,
				Remaining:   remaining,
				ResetAt:     timestamp.Add(time.Minute).UnixMilli(),
			}
		}

		batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO default.ratelimits_raw_v2")
		require.NoError(s.t, err)

		for _, r := range ratelimits {
			err = batch.AppendStruct(&r)
			require.NoError(s.t, err)
		}

		err = batch.Send()
		require.NoError(s.t, err)
	}
}

// KeyLastUsedRow represents a row in the key_last_used_v1 table for direct seeding.
type KeyLastUsedRow struct {
	WorkspaceID string   `ch:"workspace_id"`
	KeySpaceID  string   `ch:"key_space_id"`
	KeyID       string   `ch:"key_id"`
	IdentityID  string   `ch:"identity_id"`
	Time        int64    `ch:"time"`
	RequestID   string   `ch:"request_id"`
	Outcome     string   `ch:"outcome"`
	Tags        []string `ch:"tags"`
}

// InsertKeyLastUsed inserts rows directly into key_last_used_v1, bypassing the MV.
// This is much faster for bulk seeding since it skips the raw table entirely.
// keyIDs should be pre-generated MySQL key IDs so both sides match.
func (s *ClickHouseSeeder) InsertKeyLastUsed(ctx context.Context, rows []KeyLastUsedRow) {
	const batchSize = 50_000
	for i := 0; i < len(rows); i += batchSize {
		end := min(i+batchSize, len(rows))
		chunk := rows[i:end]

		batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO default.key_last_used_v1")
		require.NoError(s.t, err)

		for idx := range chunk {
			err = batch.AppendStruct(&chunk[idx])
			require.NoError(s.t, err)
		}

		err = batch.Send()
		require.NoError(s.t, err)
	}
}

// InsertRawVerificationsForKeys inserts eventsPerKey raw verification events per key
// into key_verifications_raw_v2. The materialized view auto-populates key_last_used_v1.
// Uses parallel workers for fast bulk seeding at scale (e.g., 10M keys × 100 events = 1B).
func (s *ClickHouseSeeder) InsertRawVerificationsForKeys(
	ctx context.Context,
	workspaceID, keySpaceID string,
	keyIDs []string,
	eventsPerKey int,
) {
	s.t.Helper()

	const workers = 4
	chunkSize := (len(keyIDs) + workers - 1) / workers

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errs := make(chan error, workers)

	for w := range workers {
		start := w * chunkSize
		end := min(start+chunkSize, len(keyIDs))
		if start >= len(keyIDs) {
			break
		}

		wg.Add(1)
		go func(chunk []string, workerID int) {
			defer wg.Done()
			if err := s.insertVerificationsForKeyChunk(ctx, workspaceID, keySpaceID, chunk, eventsPerKey); err != nil {
				cancel()
				errs <- fmt.Errorf("worker %d: %w", workerID, err)
			}
		}(keyIDs[start:end], w)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(s.t, err)
	}
}

func (s *ClickHouseSeeder) insertVerificationsForKeyChunk(
	ctx context.Context,
	workspaceID, keySpaceID string,
	keyIDs []string,
	eventsPerKey int,
) error {
	// 1K keys × 100 events = 100K rows per ClickHouse batch
	const keysPerBatch = 1_000

	now := time.Now().UnixMilli()

	for i := 0; i < len(keyIDs); i += keysPerBatch {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		end := min(i+keysPerBatch, len(keyIDs))
		chunk := keyIDs[i:end]

		batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO default.key_verifications_raw_v2")
		if err != nil {
			return fmt.Errorf("prepare batch at offset %d: %w", i, err)
		}

		for keyIdx, keyID := range chunk {
			baseTime := now - int64(len(keyIDs)-i-keyIdx)*int64(eventsPerKey)
			for j := range eventsPerKey {
				v := schema.KeyVerification{
					RequestID:    fmt.Sprintf("perf-%d-%d", i+keyIdx, j),
					Time:         baseTime + int64(j),
					WorkspaceID:  workspaceID,
					KeySpaceID:   keySpaceID,
					KeyID:        keyID,
					IdentityID:   "",
					ExternalID:   "",
					Region:       "us-east-1",
					Outcome:      "VALID",
					Tags:         []string{},
					SpentCredits: 0,
					Latency:      0,
				}
				if err = batch.AppendStruct(&v); err != nil {
					return fmt.Errorf("append at key %d event %d: %w", i+keyIdx, j, err)
				}
			}
		}

		if err = batch.Send(); err != nil {
			return fmt.Errorf("send batch at offset %d: %w", i, err)
		}
	}
	return nil
}

// InsertSentinelRequests inserts sentinel request records for a workspace.
func (s *ClickHouseSeeder) InsertSentinelRequests(ctx context.Context, workspaceID, projectID, environmentID, deploymentID string, count int, timestamp time.Time) {
	const batchSize = 10_000
	for i := 0; i < count; i += batchSize {
		batchCount := min(batchSize, count-i)
		requests := make([]schema.SentinelRequest, batchCount)
		for j := range batchCount {
			requests[j] = schema.SentinelRequest{
				RequestID:       uid.New(uid.RequestPrefix),
				Time:            timestamp.Add(time.Duration(i+j) * time.Millisecond).UnixMilli(),
				WorkspaceID:     workspaceID,
				ProjectID:       projectID,
				EnvironmentID:   environmentID,
				DeploymentID:    deploymentID,
				SentinelID:      uid.New(uid.SentinelPrefix),
				InstanceID:      uid.New(uid.InstancePrefix),
				InstanceAddress: "10.0.0.1:8080",
				Region:          "us-east-1",
				Method:          "GET",
				Host:            "test.example.com",
				Path:            "/",
				QueryString:     "",
				QueryParams:     map[string][]string{},
				RequestHeaders:  []string{},
				RequestBody:     "",
				ResponseStatus:  200,
				ResponseHeaders: []string{},
				ResponseBody:    "",
				UserAgent:       "test-agent",
				IPAddress:       "127.0.0.1",
				TotalLatency:    10,
				InstanceLatency: 8,
				SentinelLatency: 2,
			}
		}

		batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO default.sentinel_requests_raw_v1")
		require.NoError(s.t, err)

		for _, r := range requests {
			err = batch.AppendStruct(&r)
			require.NoError(s.t, err)
		}

		err = batch.Send()
		require.NoError(s.t, err)
	}
}
