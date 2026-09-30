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
	// Legacy root keys: 30 VALID, marked by an empty workspace ID but retaining
	// their keyspace so last-used synchronization continues to process them.
	legacyRootKeyVerifications := createVerifications(workspaceID, 30, now, "VALID")
	legacyRootKeyID := uid.New(uid.KeyPrefix)
	legacyRootKeySpaceID := uid.New(uid.KeySpacePrefix)
	for i := range legacyRootKeyVerifications {
		legacyRootKeyVerifications[i].WorkspaceID = ""
		legacyRootKeyVerifications[i].KeySpaceID = legacyRootKeySpaceID
		legacyRootKeyVerifications[i].KeyID = legacyRootKeyID
	}
	// New root keys: 10 VALID, marked by empty workspace and keyspace IDs.
	newRootKeyVerifications := createVerifications(workspaceID, 10, now, "VALID")
	newRootKeyID := uid.New(uid.KeyPrefix)
	for i := range newRootKeyVerifications {
		newRootKeyVerifications[i].WorkspaceID = ""
		newRootKeyVerifications[i].KeySpaceID = ""
		newRootKeyVerifications[i].KeyID = newRootKeyID
	}
	allVerifications := append(verifications, gatewayVerifications...)
	allVerifications = append(allVerifications, legacyRootKeyVerifications...)
	allVerifications = append(allVerifications, newRootKeyVerifications...)
	insertVerifications(t, ctx, conn, allVerifications)

	year, month := now.Year(), int(now.Month())

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		billableVerifications, err := client.GetBillableVerifications(ctx, workspaceID, year, month)
		require.NoError(c, err)
		assert.Equal(c, int64(100), billableVerifications, "gateway, root-key, and INVALID verifications must not bill the customer workspace")
		var rootKeyBillable int64
		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.billable_verifications_per_month_v2 WHERE workspace_id = ''",
		).Scan(&rootKeyBillable))
		assert.Equal(c, int64(40), rootKeyBillable, "root-key verifications must remain attributed to the empty workspace")

		// Customer analytics include API and gateway traffic. Root-key traffic
		// remains in the global rollup under the empty workspace marker.
		var totalCount, legacyRootKeyCount, newRootKeyCount, gatewayCount, appCount, unattributedCount int64
		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = ?",
			workspaceID,
		).Scan(&totalCount))
		assert.Equal(c, int64(160), totalCount, "customer analytics must include API and gateway traffic")

		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = '' AND key_id = ?",
			legacyRootKeyID,
		).Scan(&legacyRootKeyCount))
		assert.Equal(c, int64(30), legacyRootKeyCount, "analytics rollups must retain legacy root-key traffic")
		require.NoError(c, conn.QueryRow(ctx,
			"SELECT sum(count) FROM default.key_verifications_per_month_v3 WHERE workspace_id = '' AND key_id = ?",
			newRootKeyID,
		).Scan(&newRootKeyCount))
		assert.Equal(c, int64(10), newRootKeyCount, "analytics rollups must retain new root-key traffic")

		var legacyRootKeyLastUsedCount, newRootKeyLastUsedCount uint64
		require.NoError(c, conn.QueryRow(ctx,
			"SELECT count() FROM default.key_last_used_v1 WHERE key_id = ?",
			legacyRootKeyID,
		).Scan(&legacyRootKeyLastUsedCount))
		assert.Equal(c, uint64(1), legacyRootKeyLastUsedCount, "legacy root keys must retain last-used synchronization")
		require.NoError(c, conn.QueryRow(ctx,
			"SELECT count() FROM default.key_last_used_v1 WHERE key_id = ?",
			newRootKeyID,
		).Scan(&newRootKeyLastUsedCount))
		assert.Equal(c, uint64(1), newRootKeyLastUsedCount, "new root keys must enter last-used synchronization")

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
