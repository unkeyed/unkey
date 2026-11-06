package clickhouse

import (
	"context"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestKeyVerificationFailuresRollup inserts failure rows through the buffer
// and checks that the per-minute rollup counts sampled rows by their weight.
func TestKeyVerificationFailuresRollup(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	workspaceID := uid.New(uid.WorkspacePrefix)
	now := time.Now().UnixMilli()
	row := func(fingerprint uint64, weight uint32, cacheResult string) schema.KeyVerificationFailure {
		return schema.KeyVerificationFailure{
			RequestID:      uid.New(uid.RequestPrefix),
			Time:           now,
			WorkspaceID:    workspaceID,
			PrincipalType:  "API_KEY",
			RootKeyID:      "key_root",
			Region:         "test",
			Source:         schema.SourceAPI,
			IPAddress:      "192.0.2.1",
			UserAgent:      "test",
			Outcome:        "NOT_FOUND",
			Reason:         "unknown_hash",
			CacheResult:    cacheResult,
			KeyID:          "",
			KeySpaceID:     "",
			KeyWorkspaceID: "",
			KeyFingerprint: fingerprint,
			KeyLength:      35,
			KeyPrefix:      "",
			KeyCharset:     "base58",
			SampleWeight:   weight,
		}
	}

	buffer := NewBuffer[schema.KeyVerificationFailure](client, BufferConfig{
		Name:          "key_verification_failures_rollup_test",
		BatchSize:     10,
		BufferSize:    10,
		FlushInterval: time.Hour,
		Consumers:     1,
		Drop:          false,
		OnFlushError:  nil,
	})
	buffer.Buffer(row(1, 1, "miss"))
	buffer.Buffer(row(2, 1, "null"))
	buffer.Buffer(row(3, 100, "miss"))
	buffer.Close()

	var got struct {
		Count       uint64 `ch:"count"`
		CacheMisses uint64 `ch:"cache_misses"`
		Keys        uint64 `ch:"keys"`
		Rows        uint64 `ch:"rows"`
	}
	ctx := ch.Context(context.Background(), ch.WithParameters(map[string]string{"ws": workspaceID}))
	require.NoError(t, client.conn.QueryRow(ctx, `
		SELECT sum(count) AS count, sum(cache_misses) AS cache_misses,
			uniqCombined64Merge(key_fingerprints) AS keys,
			(SELECT count() FROM default.key_verification_failures_raw_v1 WHERE workspace_id = {ws:String}) AS rows
		FROM default.key_verification_failures_per_minute_v1
		WHERE workspace_id = {ws:String}`,
	).ScanStruct(&got))

	require.Equal(t, uint64(3), got.Rows)
	require.Equal(t, uint64(102), got.Count)
	require.Equal(t, uint64(101), got.CacheMisses)
	require.Equal(t, uint64(3), got.Keys)
}
