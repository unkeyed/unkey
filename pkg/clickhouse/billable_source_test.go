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

// Gateway-sourced verifications and ratelimits (Deploy's key-auth and
// ratelimit policies) must not bill as API usage, while analytics rollups
// keep them: gateway traffic is still the customer's traffic.
func TestBillableExcludesGatewaySource(t *testing.T) {
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

	// API: 100 VALID (billable) + 20 INVALID (never billable).
	verifications := createVerifications(workspaceID, 100, now, "VALID")
	verifications = append(verifications, createVerifications(workspaceID, 20, now, "INVALID")...)
	// Gateway: 40 VALID, excluded from billing by source.
	gatewayVerifications := createVerifications(workspaceID, 40, now, "VALID")
	for i := range gatewayVerifications {
		gatewayVerifications[i].Source = schema.SourceGateway
	}
	insertVerifications(t, ctx, conn, append(verifications, gatewayVerifications...))

	// API: 30 passed (billable) + 10 failed. Gateway: 50 passed, excluded.
	ratelimits := createRatelimits(workspaceID, 30, now, true)
	ratelimits = append(ratelimits, createRatelimits(workspaceID, 10, now, false)...)
	gatewayRatelimits := createRatelimits(workspaceID, 50, now, true)
	for i := range gatewayRatelimits {
		gatewayRatelimits[i].Source = schema.SourceGateway
	}
	insertRatelimits(t, ctx, conn, append(ratelimits, gatewayRatelimits...))

	year, month := now.Year(), int(now.Month())

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		billableVerifications, err := client.GetBillableVerifications(ctx, workspaceID, year, month)
		require.NoError(c, err)
		assert.Equal(c, int64(100), billableVerifications, "gateway and INVALID verifications must not bill")

		billableRatelimits, err := client.GetBillableRatelimits(ctx, workspaceID, year, month)
		require.NoError(c, err)
		assert.Equal(c, int64(30), billableRatelimits, "gateway and failed ratelimits must not bill")

		// Analytics rollups keep every source: total includes API + gateway.
		var totalCount, gatewayCount int64
		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = ?",
			workspaceID,
		).Scan(&totalCount))
		assert.Equal(c, int64(160), totalCount, "analytics rollups must include gateway traffic")

		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = ? AND source = ?",
			workspaceID, schema.SourceGateway,
		).Scan(&gatewayCount))
		assert.Equal(c, int64(40), gatewayCount, "rollups must be sliceable by source")
	}, time.Minute, time.Second)
}
