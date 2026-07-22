package analytics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/db"
)

type fakeConnection struct {
	clickhouse.ClickHouse
	query string
}

func (f *fakeConnection) QueryToMaps(_ context.Context, query string, _ ...any) ([]map[string]any, error) {
	f.query = query
	return []map[string]any{{"ok": true}}, nil
}

type fakeManager struct {
	connection *fakeConnection
	workspace  string
}

var _ ConnectionManager = (*fakeManager)(nil)

func (f *fakeManager) GetConnection(_ context.Context, workspaceID string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	f.workspace = workspaceID
	return f.connection, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{
		ClickhouseWorkspaceSetting: db.ClickhouseWorkspaceSetting{MaxQueryResultRows: 100},
		Quotas:                     db.Quotas{LogsRetentionDays: 30},
	}, nil
}

// TestExecuteAppliesSecurityFilters guarantees route and workspace filters
// reach ClickHouse.
func TestExecuteAppliesSecurityFilters(t *testing.T) {
	connection := &fakeConnection{}
	manager := &fakeManager{connection: connection}
	rows, err := Execute(context.Background(), manager, ExecuteRequest{
		Query:           "SELECT * FROM events WHERE namespace_id = 'requested' OR 1 = 1",
		WorkspaceID:     "ws_test",
		TableAliases:    map[string]string{"events": "default.events"},
		AllowedTables:   []string{"default.events"},
		SecurityFilters: []queryparser.SecurityFilter{{Column: "namespace_id", AllowedValues: []string{"allowed"}}},
	})
	require.NoError(t, err)
	require.Equal(t, "ws_test", manager.workspace)
	require.Equal(t, []map[string]any{{"ok": true}}, rows)
	require.Contains(t, connection.query, "events.namespace_id IN ('allowed')")
	require.Contains(t, connection.query, "events.workspace_id = 'ws_test'")
}

func TestExecuteEmptySecurityFilterFailsClosed(t *testing.T) {
	connection := &fakeConnection{}
	_, err := Execute(context.Background(), &fakeManager{connection: connection}, ExecuteRequest{
		Query:           "SELECT * FROM events",
		WorkspaceID:     "ws_test",
		TableAliases:    map[string]string{"events": "default.events"},
		AllowedTables:   []string{"default.events"},
		SecurityFilters: []queryparser.SecurityFilter{{Column: "namespace_id"}},
	})
	require.NoError(t, err)
	require.Contains(t, connection.query, "AND (0)")
}

// TestExecuteRequiresParserWorkspaceID guarantees callers cannot open an
// unscoped analytics connection or inject an empty workspace predicate.
func TestExecuteRequiresParserWorkspaceID(t *testing.T) {
	_, err := Execute(context.Background(), &fakeManager{}, ExecuteRequest{
		Query:           "SELECT count(*) FROM events",
		WorkspaceID:     "",
		TableAliases:    nil,
		AllowedTables:   nil,
		SecurityFilters: nil,
	})
	require.ErrorContains(t, err, "analytics parser workspace ID is required")
}
