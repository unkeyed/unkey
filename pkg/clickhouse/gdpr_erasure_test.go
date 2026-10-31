package clickhouse

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestErasureTablesMatchSchema runs every erasure against the real schema for
// a subject with no rows, which proves each table and filter column exists,
// and requires that tables with a projection are erased with a mutation,
// because they reject a lightweight DELETE outright.
func TestErasureTablesMatchSchema(t *testing.T) {
	cfg := containers.ClickHouse(t)
	client, err := New(Config{URL: cfg.HTTPDSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	for _, table := range append(workspaceErasureTables, endUserVerificationTables...) {
		var createQuery string
		require.NoError(t, client.conn.QueryRow(ctx,
			"SELECT create_table_query FROM system.tables WHERE database = 'default' AND name = ?", table.name,
		).Scan(&createQuery), table.name)
		require.Equal(t, strings.Contains(createQuery, "PROJECTION"), table.mutation, "%s: mutation must match whether the table has a projection", table.name)
	}

	workspaceID := uid.New(uid.WorkspacePrefix)
	for _, table := range WorkspaceErasureTables() {
		rows, err := client.EraseWorkspaceRows(ctx, table, workspaceID)
		require.NoError(t, err, table)
		require.Zero(t, rows, table)
	}

	endUser := EndUserErasure{
		WorkspaceID: workspaceID,
		ExternalID:  uid.New(uid.TestPrefix),
		IdentityIDs: []string{uid.New(uid.IdentityPrefix)},
	}
	for _, table := range EndUserErasureTables() {
		rows, err := client.EraseEndUserRows(ctx, table, endUser)
		require.NoError(t, err, table)
		require.Zero(t, rows, table)
	}

	rows, err := client.PseudonymizeAuditActor(ctx, uid.New(uid.TestPrefix))
	require.NoError(t, err)
	require.Zero(t, rows)
}

func TestErasureRefusesUnlistedTablesAndEmptyIDs(t *testing.T) {
	client := &Client{}
	ctx := context.Background()
	workspaceID := uid.New(uid.WorkspacePrefix)

	_, err := client.EraseWorkspaceRows(ctx, "workspaces", workspaceID)
	require.Error(t, err)
	_, err = client.EraseWorkspaceRows(ctx, "api_requests_raw_v2", "")
	require.Error(t, err)

	_, err = client.EraseEndUserRows(ctx, "api_requests_raw_v2", EndUserErasure{WorkspaceID: workspaceID, ExternalID: uid.New(uid.TestPrefix), IdentityIDs: nil})
	require.Error(t, err)
	_, err = client.EraseEndUserRows(ctx, "key_last_used_v1", EndUserErasure{WorkspaceID: workspaceID, ExternalID: "", IdentityIDs: nil})
	require.Error(t, err)

	_, err = client.PseudonymizeAuditActor(ctx, "")
	require.Error(t, err)
}
