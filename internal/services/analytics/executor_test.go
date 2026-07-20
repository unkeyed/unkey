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
	events     *[]string
}

var _ ConnectionManager = (*fakeManager)(nil)

func (f *fakeManager) GetConnection(context.Context, string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	*f.events = append(*f.events, "connection")
	return f.connection, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{
		ClickhouseWorkspaceSetting: db.ClickhouseWorkspaceSetting{MaxQueryResultRows: 100},
		Quotas:                     db.Quotas{LogsRetentionDays: 30},
	}, nil
}

func TestExecuteOrderingAndFilterComposition(t *testing.T) {
	events := []string{}
	connection := &fakeConnection{}
	rows, err := Execute(context.Background(), &fakeManager{connection: connection, events: &events}, ExecuteRequest{
		WorkspaceID: "ws_test",
		Query:       "SELECT * FROM events WHERE namespace_id = 'requested' OR 1 = 1",
		ParserConfig: queryparser.Config{
			TableAliases:  map[string]string{"events": "default.events"},
			AllowedTables: []string{"default.events"},
		},
		FilterBuilder: func() ([]queryparser.SecurityFilter, error) {
			events = append(events, "pre-parse")
			return []queryparser.SecurityFilter{{Column: "environment", AllowedValues: []string{"prod"}}}, nil
		},
		Policy: func(parser *queryparser.Parser) ([]queryparser.SecurityFilter, error) {
			events = append(events, "post-parse-policy")
			require.Equal(t, []string{"requested"}, parser.ExtractColumn("namespace_id"))
			return []queryparser.SecurityFilter{{Column: "namespace_id", AllowedValues: []string{"allowed"}}}, nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"connection", "pre-parse", "post-parse-policy"}, events)
	require.Equal(t, []map[string]any{{"ok": true}}, rows)
	require.Contains(t, connection.query, "events.environment IN ('prod')")
	require.Contains(t, connection.query, "events.namespace_id IN ('allowed')")
	require.Contains(t, connection.query, "events.workspace_id = 'ws_test'")
}

func TestExecutePolicyExtractsInjectedQualifiedFilter(t *testing.T) {
	events := []string{}
	connection := &fakeConnection{}
	_, err := Execute(context.Background(), &fakeManager{connection: connection, events: &events}, ExecuteRequest{
		WorkspaceID: "ws_test",
		Query:       "SELECT count(*) FROM events",
		ParserConfig: queryparser.Config{
			TableAliases:  map[string]string{"events": "default.events"},
			AllowedTables: []string{"default.events"},
		},
		FilterBuilder: func() ([]queryparser.SecurityFilter, error) {
			return []queryparser.SecurityFilter{{Column: "key_space_id", AllowedValues: []string{"ks_scoped"}}}, nil
		},
		Policy: func(parser *queryparser.Parser) ([]queryparser.SecurityFilter, error) {
			require.Equal(t, []string{"ks_scoped"}, parser.ExtractColumn("key_space_id"))
			return nil, nil
		},
	})
	require.NoError(t, err)
}
