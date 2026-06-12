package clickhouse_test

import (
	"context"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Active keys = distinct keys verified through the gateway in the month,
// regardless of outcome. API-sourced verifications never count.
func TestGetActiveKeysUsage(t *testing.T) {
	chCfg := containers.ClickHouse(t)

	client, err := clickhouse.New(clickhouse.Config{URL: chCfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	opts, err := ch.ParseDSN(chCfg.DSN)
	require.NoError(t, err)
	conn, err := ch.Open(opts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	ctx := context.Background()
	require.NoError(t, conn.Ping(ctx))

	now := time.Now()
	workspaceID := uid.New(uid.WorkspacePrefix)

	// 3 distinct gateway keys: one VALID, one RATE_LIMITED (still active),
	// one verified twice (counted once). Plus 2 API-sourced keys (never
	// counted) and one gateway key in another workspace.
	gateway := func(keyID, outcome string) schema.KeyVerification {
		v := createVerifications(workspaceID, 1, now, outcome)[0]
		v.KeyID = keyID
		v.Source = schema.SourceGateway
		return v
	}
	rows := []schema.KeyVerification{
		gateway("key_a", "VALID"),
		gateway("key_b", "RATE_LIMITED"),
		gateway("key_c", "VALID"),
		gateway("key_c", "VALID"),
	}
	api := createVerifications(workspaceID, 2, now, "VALID")
	for i := range api {
		api[i].Source = schema.SourceAPI
	}
	other := createVerifications(uid.New(uid.WorkspacePrefix), 1, now, "VALID")
	other[0].Source = schema.SourceGateway
	insertVerifications(t, ctx, conn, append(append(rows, api...), other...))

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		usage, err := client.GetActiveKeysUsage(ctx, clickhouse.GetActiveKeysUsageRequest{
			WorkspaceID: workspaceID,
			Month:       now.UnixMilli(),
		})
		require.NoError(c, err)
		require.Len(c, usage, 1)
		assert.Equal(c, workspaceID, usage[0].WorkspaceID)
		assert.Equal(c, int64(3), usage[0].ActiveKeys)
	}, time.Minute, time.Second)

	// All-workspaces variant includes the other workspace's key.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		usage, err := client.GetActiveKeysUsage(ctx, clickhouse.GetActiveKeysUsageRequest{
			WorkspaceID: "",
			Month:       now.UnixMilli(),
		})
		require.NoError(c, err)
		assert.GreaterOrEqual(c, len(usage), 2)
	}, time.Minute, time.Second)
}
