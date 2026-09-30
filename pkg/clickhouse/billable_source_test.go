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

// Gateway-sourced verifications (Deploy's key-auth policy) must not bill as
// API usage, while analytics rollups keep them: gateway traffic is still the
// customer's traffic.
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
		if i < 25 {
			gatewayVerifications[i].AppID = "app_a"
		} else {
			gatewayVerifications[i].AppID = "app_b"
		}
	}
	// Root keys: 30 VALID, marked by an empty workspace ID and excluded from
	// API-key billing.
	rootKeyVerifications := createVerifications(workspaceID, 30, now, "VALID")
	for i := range rootKeyVerifications {
		rootKeyVerifications[i].WorkspaceID = ""
	}
	allVerifications := append(verifications, gatewayVerifications...)
	insertVerifications(t, ctx, conn, append(allVerifications, rootKeyVerifications...))

	year, month := now.Year(), int(now.Month())

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		billableVerifications, err := client.GetBillableVerifications(ctx, workspaceID, year, month)
		require.NoError(c, err)
		assert.Equal(c, int64(100), billableVerifications, "gateway, root-key, and INVALID verifications must not bill")
		var rootKeyBillable int64
		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.billable_verifications_per_month_v2 WHERE workspace_id = ''",
		).Scan(&rootKeyBillable))
		assert.Zero(c, rootKeyBillable, "root-key verifications must not bill")

		// Customer analytics include API and gateway traffic. Root-key traffic
		// remains in the global rollup under the empty workspace marker.
		var totalCount, rootKeyCount, gatewayCount, appCount, unattributedCount int64
		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = ?",
			workspaceID,
		).Scan(&totalCount))
		assert.Equal(c, int64(160), totalCount, "customer analytics must include API and gateway traffic")

		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = ''",
		).Scan(&rootKeyCount))
		assert.Equal(c, int64(30), rootKeyCount, "analytics rollups must retain root-key traffic")

		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = ? AND source = ?",
			workspaceID, schema.SourceGateway,
		).Scan(&gatewayCount))
		assert.Equal(c, int64(40), gatewayCount, "rollups must be sliceable by source")

		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = ? AND app_id = ?",
			workspaceID, "app_a",
		).Scan(&appCount))
		assert.Equal(c, int64(25), appCount, "rollups must preserve the app attribution")

		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = ? AND app_id = ''",
			workspaceID,
		).Scan(&unattributedCount))
		assert.Equal(c, int64(120), unattributedCount, "API verifications must remain unattributed")
	}, time.Minute, time.Second)
}
